package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

	"lopy/prompts"
)

const (
	// tokenBytes is a rough bytes-per-token ratio for mixed prose, code and JSON.
	tokenBytes   = 4
	imageTokens  = 1200 // charged per image so base64 payloads cannot dominate
	summaryLimit = 1200
)

const summaryInstruction = "Summarize this conversation so work can continue. " +
	"Keep decisions, file paths, commands, results and open questions. Do not use tools."

// estimateTokens approximates the prompt size of the whole history.
func estimateTokens(history []json.RawMessage) int64 {
	var total int64
	for _, item := range history {
		if strings.Contains(string(item), "input_image") {
			total += imageTokens
		}
		total += int64(len(item)) / tokenBytes
	}
	return total
}

// userTurn reports whether item starts a new conversational turn.
func userTurn(item json.RawMessage) bool {
	var v struct{ Role, Type string }
	if json.Unmarshal(item, &v) != nil {
		return false
	}
	return v.Role == "user" && v.Type != "function_call_output"
}

// turnStart returns the index where the last n user turns begin, or 0 when there are fewer.
func turnStart(history []json.RawMessage, turns int) int {
	found := 0
	for i := len(history) - 1; i >= 0; i-- {
		if userTurn(history[i]) {
			found++
			if found == turns {
				return i
			}
		}
	}
	return 0
}

// instructions is the system prompt plus prefix source used by every request.
func (a *Agent) instructions() string {
	system := a.System
	if system == "" {
		system = prompts.System
	}
	return system + "\n\n<prefix>\n" + a.REPL.Prefix + "\n</prefix>"
}

// summarize condenses the old part into one summary written under the session instructions.
func (a *Agent) summarize(ctx context.Context, old []json.RawMessage, prompt string) (string, error) {
	var text strings.Builder
	for _, item := range old {
		for _, r := range RecordsForItem(item) {
			fmt.Fprintf(&text, "[%s] %s\n", r.Kind, r.Text)
		}
	}
	instruction := summaryInstruction
	if prompt = strings.TrimSpace(prompt); prompt != "" {
		instruction += " Instruction: " + prompt
	}
	request := responses.ResponseNewParams{
		Model:           a.settings.Model,
		Instructions:    openai.String(a.instructions()),
		Store:           openai.Bool(false),
		MaxOutputTokens: openai.Int(summaryLimit),
		Input: responses.ResponseNewParamsInputUnion{OfInputItemList: responses.ResponseInputParam{
			{OfMessage: &responses.EasyInputMessageParam{
				Role:    responses.EasyInputMessageRoleUser,
				Content: responses.EasyInputMessageContentUnionParam{OfString: openai.String(instruction + "\n\n" + text.String())},
			}},
		}},
	}
	if a.settings.Effort != "" {
		request.Reasoning.Effort = shared.ReasoningEffort(a.settings.Effort)
	}
	response, err := a.client.Responses.New(ctx, request)
	if err != nil {
		return "", err
	}
	var summary strings.Builder
	for _, item := range response.Output {
		for _, part := range item.Content {
			if part.Type == "output_text" {
				summary.WriteString(part.Text)
			}
		}
	}
	if strings.TrimSpace(summary.String()) == "" {
		return "", fmt.Errorf("provider returned an empty summary")
	}
	return strings.TrimSpace(summary.String()), nil
}

// Compact replaces everything before the last CompactKeepTurns turns with one summary.
// Callers ensure an idle, writable session.
func (a *Agent) Compact(ctx context.Context, prompt string) (string, error) {
	keep := max(1, a.settings.CompactKeepTurns)
	cut := turnStart(a.History, keep)
	if cut == 0 {
		return "Nothing to compact.", nil
	}
	old := append([]json.RawMessage(nil), a.History[:cut]...)
	summary, err := a.summarize(ctx, old, prompt)
	if err != nil {
		if strings.TrimSpace(prompt) != "" {
			return "", err
		}
		summary = "Earlier conversation was removed to fit the context limit."
	}
	item := raw(map[string]any{"role": "user", "content": "<compact_summary>\n" + summary + "\n</compact_summary>"})
	a.History = append([]json.RawMessage{item}, a.History[cut:]...)
	a.records = nil
	if err := a.Save(); err != nil {
		return "", err
	}
	return fmt.Sprintf("Compacted %d items into a summary; kept the last %d turns.", len(old), keep), nil
}

// compactIfNeeded summarizes older turns before a request would exceed the token budget.
func (a *Agent) compactIfNeeded(ctx context.Context, emit func(Event)) {
	limit := a.settings.MaxContextTokens
	if limit <= 0 || estimateTokens(a.History) <= limit {
		return
	}
	result, err := a.Compact(ctx, "")
	if err != nil {
		emit(Event{Kind: "error", Text: "Compaction failed: " + err.Error()})
		return
	}
	emit(Event{Kind: "info", Text: result})
}

// Reload re-reads settings and prompts and restarts Monty while keeping the history.
func (a *Agent) Reload() error {
	if err := a.Writable(); err != nil {
		return err
	}
	c, err := loadConfiguration(a.home)
	if err != nil {
		return err
	}
	a.REPL.Close()
	a.REPL.Prefix, a.REPL.PrefixOutput = c.prefix, ""
	a.System = c.system
	a.applySettings(c.settings)
	return nil
}
