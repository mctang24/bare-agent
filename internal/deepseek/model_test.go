package deepseek

import (
	"bare-agent/internal/agent"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		for _, expected := range []string{
			`"role":"system","content":"inspect before answering"`,
			`"role":"assistant","content":null,"reasoning_content":"inspect first"`,
			`"role":"tool","content":"found","tool_call_id":"call_1"`,
			`"name":"search_text"`,
		} {
			if !strings.Contains(string(body), expected) {
				t.Errorf("request body = %s, want to contain %s", body, expected)
			}
		}
		if strings.Contains(string(body), `"parameters":null`) {
			t.Errorf("request body = %s, want nil parameters omitted", body)
		}
		if count := strings.Count(string(body), `"role":"system"`); count != 1 {
			t.Errorf("system message count = %d, want 1", count)
		}
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":null,"reasoning_content":"read it","tool_calls":[{"id":"call_2","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}}]}}],"usage":{"prompt_tokens":120,"completion_tokens":30,"total_tokens":150,"prompt_cache_hit_tokens":80,"prompt_cache_miss_tokens":40}}`))
	}))
	defer server.Close()

	client := DeepSeekClient{httpClient: server.Client(), baseURL: server.URL, apiKey: "test-key", model: "test-model"}
	response, err := client.GenerateResponse(context.Background(), agent.ModelRequest{
		Instructions: "inspect before answering",
		Messages: []agent.Message{
			{Role: "user", Content: "find target"},
			{RawMessage: []byte(`{"role":"assistant","content":null,"reasoning_content":"inspect first","tool_calls":[{"id":"call_1","type":"function","function":{"name":"search_text","arguments":"{\"query\":\"target\"}"}}]}`)},
			{Role: "tool", ToolResults: []agent.ToolResult{{ToolCallID: "call_1", Content: "found"}}},
		},
		Tools: []agent.ToolDefinition{
			{Name: "search_text", Description: "search text", Parameters: map[string]any{"type": "object"}},
			{Name: "no_args", Description: "no arguments"},
		},
	})
	if err != nil {
		t.Fatalf("GenerateResponse() error = %v", err)
	}
	if len(response.Message.ToolCalls) != 1 || response.Message.ToolCalls[0].Name != "read_file" {
		t.Fatalf("tool calls = %#v, want read_file", response.Message.ToolCalls)
	}
	if response.Usage != (agent.TokenUsage{PromptTokens: 120, CompletionTokens: 30, TotalTokens: 150, PromptCacheHitTokens: 80, PromptCacheMissTokens: 40}) {
		t.Fatalf("usage = %#v", response.Usage)
	}
	var raw map[string]any
	if err := json.Unmarshal(response.Message.RawMessage, &raw); err != nil {
		t.Fatalf("decode raw message: %v", err)
	}
	if raw["reasoning_content"] != "read it" {
		t.Fatalf("raw reasoning_content = %#v, want read it", raw["reasoning_content"])
	}
}

func TestGenerateResponseOmitsEmptyInstructions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if strings.Contains(string(body), `"role":"system"`) {
			t.Errorf("request body = %s, want no system message", body)
		}
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"done"}}]}`))
	}))
	defer server.Close()

	client := DeepSeekClient{httpClient: server.Client(), baseURL: server.URL, apiKey: "test-key", model: "test-model"}
	_, err := client.GenerateResponse(context.Background(), agent.ModelRequest{
		Messages: []agent.Message{{Role: "user", Content: "find target"}},
	})
	if err != nil {
		t.Fatalf("GenerateResponse() error = %v", err)
	}
}

func TestGenerateResponseErrors(t *testing.T) {
	tests := []struct {
		name       string
		request    agent.ModelRequest
		response   string
		wantErr    string
		wantCalled bool
	}{
		{name: "unsupported role", request: agent.ModelRequest{Messages: []agent.Message{{Role: "system"}}}, wantErr: "unsupported role"},
		{name: "user message with tool data", request: agent.ModelRequest{Messages: []agent.Message{{Role: "user", ToolCalls: []agent.ToolCall{{ID: "call_1"}}}}}, wantErr: "user message 0 contains tool data"},
		{name: "assistant message with tool results", request: agent.ModelRequest{Messages: []agent.Message{{Role: "assistant", ToolResults: []agent.ToolResult{{ToolCallID: "call_1"}}}}}, wantErr: "assistant message 0 contains tool results"},
		{name: "invalid raw message", request: agent.ModelRequest{Messages: []agent.Message{{RawMessage: []byte(`{`)}}}, wantErr: "decode raw message"},
		{name: "raw message with conflicting role", request: agent.ModelRequest{Messages: []agent.Message{{Role: "user", RawMessage: []byte(`{"role":"assistant","content":"done"}`)}}}, wantErr: "conflicting role"},
		{name: "raw assistant message with tool results", request: agent.ModelRequest{Messages: []agent.Message{{Role: "assistant", RawMessage: []byte(`{"role":"assistant","content":"done"}`), ToolResults: []agent.ToolResult{{ToolCallID: "call_1"}}}}}, wantErr: "contains tool results"},
		{name: "tool message without results", request: agent.ModelRequest{Messages: []agent.Message{{Role: "tool"}}}, wantErr: "has no results"},
		{name: "tool message with content", request: agent.ModelRequest{Messages: []agent.Message{{Role: "tool", Content: "duplicate", ToolResults: []agent.ToolResult{{ToolCallID: "call_1"}}}}}, wantErr: "contains non-result data"},
		{name: "tool message with tool calls", request: agent.ModelRequest{Messages: []agent.Message{{Role: "tool", ToolCalls: []agent.ToolCall{{ID: "call_2"}}, ToolResults: []agent.ToolResult{{ToolCallID: "call_1"}}}}}, wantErr: "contains non-result data"},
		{name: "unsafe finish reason", request: agent.ModelRequest{Messages: []agent.Message{{Role: "user", Content: "test"}}}, response: `{"choices":[{"finish_reason":"length","message":{"role":"assistant","content":"partial"}}]}`, wantErr: `finish reason "length"`, wantCalled: true},
		{name: "unknown finish reason", request: agent.ModelRequest{Messages: []agent.Message{{Role: "user", Content: "test"}}}, response: `{"choices":[{"finish_reason":"future_reason","message":{"role":"assistant","content":"partial"}}]}`, wantErr: `finish reason "future_reason"`, wantCalled: true},
		{name: "missing tool calls", request: agent.ModelRequest{Messages: []agent.Message{{Role: "user", Content: "test"}}}, response: `{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":null}}]}`, wantErr: "returned none", wantCalled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()
			client := DeepSeekClient{httpClient: server.Client(), baseURL: server.URL, apiKey: "test-key", model: "test-model"}

			_, err := client.GenerateResponse(context.Background(), tt.request)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("GenerateResponse() error = %v, want to contain %q", err, tt.wantErr)
			}
			if called != tt.wantCalled {
				t.Fatalf("server called = %v, want %v", called, tt.wantCalled)
			}
		})
	}
}
