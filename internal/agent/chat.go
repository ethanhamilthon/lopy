package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"lopy/internal/repl"
)

var codeField = regexp.MustCompile(`"code"\s*:\s*"`)

// Decode only the complete prefix of a JSON string, withholding split escapes,
// UTF-8 characters and surrogate pairs until the next delta completes them.
func partialCode(arguments string) string {
	loc := codeField.FindStringIndex(arguments)
	if loc == nil {
		return ""
	}
	quoted := arguments[loc[1]-1:]
	var text string
	if json.NewDecoder(strings.NewReader(quoted)).Decode(&text) == nil {
		return text
	}
	for end := len(quoted); end > 0 && end >= len(quoted)-12; end-- {
		candidate := quoted[:end]
		if !utf8.ValidString(candidate) {
			continue
		}
		if json.Unmarshal([]byte(candidate+`"`), &text) == nil {
			if len(candidate) >= 6 {
				tail := strings.ToLower(candidate[len(candidate)-6:])
				if strings.HasPrefix(tail, `\ud`) && strings.ContainsAny(tail[3:4], "89ab") {
					slashes := 0
					for i := len(candidate) - 6; i >= 0 && candidate[i] == '\\'; i-- {
						slashes++
					}
					if slashes%2 == 1 {
						continue
					}
				}
			}
			return text
		}
	}
	return ""
}

// History stays in one canonical format; only the wire adapter changes.
func chatMessages(history []json.RawMessage, prompt string) ([]any, error) {
	messages := []any{map[string]any{"role": "system", "content": prompt}}
	var calls []any
	flush := func() {
		if len(calls) > 0 {
			messages = append(messages, map[string]any{"role": "assistant", "tool_calls": calls})
			calls = nil
		}
	}
	for _, data := range history {
		var item struct {
			Type, Role, Name, Arguments, Output string
			CallID                              string `json:"call_id"`
			Content                             json.RawMessage
		}
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		if item.Type == "function_call" {
			calls = append(calls, map[string]any{"id": item.CallID, "type": "function", "function": map[string]any{"name": item.Name, "arguments": item.Arguments}})
			continue
		}
		flush()
		if item.Type == "function_call_output" {
			messages = append(messages, map[string]any{"role": "tool", "tool_call_id": item.CallID, "content": item.Output})
		} else if item.Role == "user" || item.Role == "assistant" {
			var text string
			var content any
			if json.Unmarshal(item.Content, &text) == nil {
				content = text
			} else {
				var parts []struct {
					Type, Text, Refusal string
					URL                 string `json:"image_url"`
				}
				if err := json.Unmarshal(item.Content, &parts); err != nil {
					return nil, err
				}
				converted := []any{}
				for _, p := range parts {
					if p.Type == "input_image" {
						converted = append(converted, map[string]any{"type": "image_url", "image_url": map[string]any{"url": p.URL}})
					}
					if p.Text != "" {
						converted = append(converted, map[string]any{"type": "text", "text": p.Text})
						text += p.Text
					}
					if p.Refusal != "" {
						converted = append(converted, map[string]any{"type": "text", "text": p.Refusal})
						text += p.Refusal
					}
				}
				content = converted
				if item.Role == "assistant" {
					content = text
				}
				if len(converted) == 0 {
					continue
				}
			}
			messages = append(messages, map[string]any{"role": item.Role, "content": content})
		}
	}
	flush()
	return messages, nil
}

func (a *Agent) requestChat(ctx context.Context, prompt string, emit func(Event)) ([]json.RawMessage, error) {
	messages, err := chatMessages(a.History, prompt)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"model": a.settings.Model, "messages": messages, "store": false, "stream_options": map[string]any{"include_usage": true},
		"tools": []any{map[string]any{"type": "function", "function": map[string]any{"name": "python_repl", "description": repl.ToolDescription, "strict": true, "parameters": pythonParameters}}}}
	if a.settings.Effort != "" {
		body["reasoning_effort"] = a.settings.Effort
	}
	started := time.Now()
	a.lastRequest = Metadata{RequestID: fmt.Sprintf("r-%d", len(a.History))}
	stream := a.client.Chat.Completions.NewStreaming(ctx, param.Override[openai.ChatCompletionNewParams](raw(body)))
	defer stream.Close()
	calls := map[int64]openai.ChatCompletionChunkChoiceDeltaToolCall{}
	text, refusal, finish := "", "", ""
	for stream.Next() {
		chunk := stream.Current()
		if chunk.JSON.Usage.Valid() {
			n, cached := chunk.Usage.PromptTokens, int64(-1)
			if chunk.Usage.PromptTokensDetails.JSON.CachedTokens.Valid() {
				cached = chunk.Usage.PromptTokensDetails.CachedTokens
			}
			a.lastRequest.ContextInput = &n
			a.lastRequest.Usage = &Usage{Input: n, Output: chunk.Usage.CompletionTokens, Cached: cached}
		}
		for _, choice := range chunk.Choices {
			if choice.Index != 0 {
				return nil, fmt.Errorf("unexpected chat choice index %d", choice.Index)
			}
			if choice.FinishReason != "" {
				finish = choice.FinishReason
			}
			for _, part := range []struct{ kind, text string }{{"delta", choice.Delta.Content}, {"refusal_delta", choice.Delta.Refusal}} {
				if part.text != "" {
					emit(Event{Kind: part.kind, Text: part.text, ID: a.lastRequest.RequestID + "-0"})
				}
			}
			text += choice.Delta.Content
			refusal += choice.Delta.Refusal
			for _, delta := range choice.Delta.ToolCalls {
				call := calls[delta.Index]
				if delta.ID != "" {
					call.ID = delta.ID
				}
				call.Function.Name += delta.Function.Name
				call.Function.Arguments += delta.Function.Arguments
				calls[delta.Index] = call
				if call.Function.Name == "python_repl" {
					emit(Event{Kind: "code", Text: partialCode(call.Function.Arguments), ID: fmt.Sprintf("%s-%d", a.lastRequest.RequestID, delta.Index+1), Provisional: true, Final: true})
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	if finish != "stop" && finish != "tool_calls" {
		return nil, fmt.Errorf("chat stream ended without success: %q", finish)
	}
	a.lastRequest.Duration = time.Since(started)
	content := []any{}
	if text != "" {
		content = append(content, map[string]any{"type": "output_text", "text": text})
	}
	if refusal != "" {
		content = append(content, map[string]any{"type": "refusal", "refusal": refusal})
	}
	items := []json.RawMessage{raw(map[string]any{"type": "message", "role": "assistant", "content": content})}
	indices := []int{}
	for index := range calls {
		indices = append(indices, int(index))
	}
	sort.Ints(indices)
	for position, index := range indices {
		call := calls[int64(index)]
		if index != position || call.ID == "" || call.Function.Name == "" {
			return nil, fmt.Errorf("invalid chat tool call at index %d", index)
		}
		items = append(items, raw(map[string]any{"type": "function_call", "call_id": call.ID, "name": call.Function.Name, "arguments": call.Function.Arguments}))
	}
	return items, nil
}
