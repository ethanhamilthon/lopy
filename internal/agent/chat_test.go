package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"lopy/internal/repl"
)

func TestChatCompletionsStreamToolsAndCanonicalHistory(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/chat/completions" {
			t.Error(r.URL.Path)
		}
		var body struct {
			Messages []map[string]any
			Tools    []map[string]any
			Effort   string `json:"reasoning_effort"`
			Stream   bool
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if !body.Stream || body.Effort != "max" || len(body.Tools) != 1 {
			t.Errorf("bad request: %+v", body)
		}
		if len(body.Messages) == 0 || body.Messages[0]["role"] != "system" || !strings.Contains(body.Messages[0]["content"].(string), "<prefix>") {
			t.Error("missing prompt")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		send := func(delta any, finish any) {
			fmt.Fprintf(w, "data: %s\n\n", raw(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}}))
		}
		if requests == 1 {
			send(map[string]any{"content": "Checking"}, nil)
			send(map[string]any{"tool_calls": []any{
				map[string]any{"index": 0, "id": "c0", "type": "function", "function": map[string]any{"name": "python_repl", "arguments": `{"code":"x = 40`}},
				map[string]any{"index": 1, "id": "c1", "type": "function", "function": map[string]any{"name": "python_repl", "arguments": `{"code":"x + 2`}},
			}}, nil)
			send(map[string]any{"tool_calls": []any{
				map[string]any{"index": 1, "function": map[string]any{"arguments": `"}`}},
				map[string]any{"index": 0, "function": map[string]any{"arguments": `"}`}},
			}}, nil)
			send(map[string]any{}, "tool_calls")
		} else {
			tools := []string{}
			for _, m := range body.Messages {
				if m["role"] == "tool" {
					tools = append(tools, m["tool_call_id"].(string))
					if m["tool_call_id"] == "c1" && !strings.Contains(m["content"].(string), "42") {
						t.Error("calls did not share sequential state")
					}
				}
			}
			if strings.Join(tools, ",") != "c0,c1" {
				t.Errorf("tool pairing: %v", tools)
			}
			send(map[string]any{"content": "Done"}, nil)
			send(map[string]any{}, "stop")
		}
		fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":10,\"prompt_tokens_details\":{\"cached_tokens\":50}}}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	a := &Agent{REPL: &repl.REPL{}, HistoryPath: filepath.Join(t.TempDir(), "history.json")}
	a.applySettings(Settings{Provider: "p", Model: "m", Effort: "max", Providers: map[string]Provider{"p": {BaseURL: server.URL, APIType: "chat_completions"}}})
	defer a.Close()
	previews := 0
	if err := a.Run(context.Background(), "start", func(e Event) {
		if e.Provisional {
			previews++
		}
	}); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || previews < 2 {
		t.Fatalf("requests=%d previews=%d", requests, previews)
	}
	if a.lastRequest.Usage == nil || *a.lastRequest.Usage != (Usage{Input: 100, Output: 10, Cached: 50}) {
		t.Fatal(a.lastRequest)
	}
	for _, item := range a.History {
		if strings.Contains(string(item), "tool_calls") {
			t.Fatal("wire format leaked into canonical history")
		}
	}
}

func TestChatFailureNeverExecutesPartialTools(t *testing.T) {
	for _, finish := range []string{"", "length", "content_filter"} {
		t.Run(finish, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				delta := map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "c", "function": map[string]any{"name": "python_repl", "arguments": string(raw(map[string]any{"code": "x = 99"}))}}}}
				fmt.Fprintf(w, "data: %s\n\n", raw(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}}))
			}))
			defer server.Close()
			a := &Agent{REPL: &repl.REPL{}, HistoryPath: filepath.Join(t.TempDir(), "history.json")}
			defer a.Close()
			a.applySettings(Settings{Provider: "p", Model: "m", Providers: map[string]Provider{"p": {BaseURL: server.URL, APIType: "chat_completions"}}})
			if err := a.Run(context.Background(), "start", func(Event) {}); err == nil {
				t.Fatal("failed stream accepted")
			}
			result, err := a.REPL.Execute(context.Background(), "x")
			if err != nil || !result.Error {
				t.Fatalf("partial tool executed: %+v %v", result, err)
			}
		})
	}
}

func TestChatImageAndRefusalTranslation(t *testing.T) {
	items := []json.RawMessage{
		raw(map[string]any{"role": "user", "content": []any{map[string]any{"type": "input_text", "text": "image"}, map[string]any{"type": "input_image", "image_url": "data:image/png;base64,example"}}}),
		raw(map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "refusal", "refusal": "Cannot do that"}}}),
	}
	messages, err := chatMessages(items, "system")
	if err != nil {
		t.Fatal(err)
	}
	data := string(raw(messages))
	if !strings.Contains(data, `"image_url":{"url":"data:image/png;base64,example"}`) || !strings.Contains(data, "Cannot do that") {
		t.Fatal(data)
	}
}
