package deepseek

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateChatCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		if authorization := r.Header.Get("Authorization"); authorization != "Bearer test-key" {
			t.Errorf("Authorization = %q, want Bearer test-key", authorization)
		}
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		for _, expected := range []string{`"model":"deepseek-v4-flash"`, `"role":"user"`, `"content":"find target"`, `"name":"search_text"`, `"stream":false`} {
			if !strings.Contains(string(body), expected) {
				t.Errorf("body = %q, want to contain %q", body, expected)
			}
		}
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"done"}}],"usage":{"prompt_tokens":120,"completion_tokens":30,"total_tokens":150,"prompt_cache_hit_tokens":80,"prompt_cache_miss_tokens":40}}`))
	}))
	defer server.Close()

	client := DeepSeekClient{
		httpClient: server.Client(),
		baseURL:    server.URL,
		apiKey:     "test-key",
		model:      "deepseek-v4-flash",
	}
	response, err := client.createChatCompletion(context.Background(), chatCompletionRequest{
		Messages: []message{{Role: "user", Content: new("find target")}},
		Tools: []toolDefinition{{
			Type: "function",
			Function: functionDefinition{
				Name:        "search_text",
				Description: "search text",
				Parameters:  map[string]any{"type": "object"},
			},
		}},
	})
	if err != nil {
		t.Fatalf("createChatCompletion() error = %v", err)
	}
	if response.Message.Content == nil || *response.Message.Content != "done" {
		t.Errorf("response content = %v, want done", response.Message.Content)
	}
	if response.Usage != (tokenUsage{PromptTokens: 120, CompletionTokens: 30, TotalTokens: 150, PromptCacheHitTokens: 80, PromptCacheMissTokens: 40}) {
		t.Errorf("response usage = %#v", response.Usage)
	}
}

func TestCreateChatCompletionErrors(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		http.Error(w, "busy", http.StatusTooManyRequests)
	}))
	defer server.Close()

	tests := []struct {
		name     string
		ctx      context.Context
		apiKey   string
		messages []message
		wantErr  string
	}{
		{name: "empty API key", ctx: context.Background(), messages: []message{{Role: "user", Content: new("test")}}, wantErr: "API key is empty"},
		{name: "empty messages", ctx: context.Background(), apiKey: "test-key", wantErr: "has no messages"},
		{name: "non-success status", ctx: context.Background(), apiKey: "test-key", messages: []message{{Role: "user", Content: new("test")}}, wantErr: "429 Too Many Requests"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := DeepSeekClient{
				httpClient: server.Client(),
				baseURL:    server.URL,
				apiKey:     tt.apiKey,
				model:      "deepseek-v4-flash",
			}
			_, err := client.createChatCompletion(tt.ctx, chatCompletionRequest{Messages: tt.messages})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("createChatCompletion() error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
	if calls != 3 {
		t.Fatalf("server calls = %d, want 3", calls)
	}
}

func TestCreateChatCompletionRetries(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "busy", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"done"}}]}`))
	}))
	defer server.Close()

	client := DeepSeekClient{httpClient: server.Client(), baseURL: server.URL, apiKey: "test-key", model: "deepseek-v4-flash"}
	_, err := client.createChatCompletion(context.Background(), chatCompletionRequest{Messages: []message{{Role: "user", Content: new("test")}}})
	if err != nil {
		t.Fatalf("createChatCompletion() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("server calls = %d, want 2", calls)
	}
}

func TestCreateChatCompletionDoesNotRetryBadRequest(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		http.Error(w, "invalid request", http.StatusBadRequest)
	}))
	defer server.Close()

	client := DeepSeekClient{httpClient: server.Client(), baseURL: server.URL, apiKey: "test-key", model: "deepseek-v4-flash"}
	_, err := client.createChatCompletion(context.Background(), chatCompletionRequest{Messages: []message{{Role: "user", Content: new("test")}}})
	if err == nil || !strings.Contains(err.Error(), "400 Bad Request") {
		t.Fatalf("createChatCompletion() error = %v, want 400 Bad Request", err)
	}
	if calls != 1 {
		t.Fatalf("server calls = %d, want 1", calls)
	}
}

func TestParseChatCompletion(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		wantFinish string
		wantText   string
		wantTool   string
		wantErr    bool
	}{
		{
			name:       "text response",
			data:       `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"done"}}]}`,
			wantFinish: "stop",
			wantText:   "done",
		},
		{
			name:       "tool call response",
			data:       `{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":null,"reasoning_content":"search first","tool_calls":[{"id":"call_1","type":"function","function":{"name":"search_text","arguments":"{\"query\":\"target\"}"}}]}}]}`,
			wantFinish: "tool_calls",
			wantTool:   "search_text",
		},
		{
			name:    "invalid JSON",
			data:    `{`,
			wantErr: true,
		},
		{
			name:    "no choices",
			data:    `{"choices":[]}`,
			wantErr: true,
		},
		{
			name:    "no assistant message",
			data:    `{"choices":[{"finish_reason":"stop","message":null}]}`,
			wantErr: true,
		},
		{
			name:    "invalid message role",
			data:    `{"choices":[{"finish_reason":"stop","message":{"role":"user","content":"done"}}]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := parseChatCompletion([]byte(tt.data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseChatCompletion() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseChatCompletion() error = %v", err)
			}
			if response.FinishReason != tt.wantFinish {
				t.Errorf("FinishReason = %q, want %q", response.FinishReason, tt.wantFinish)
			}
			if tt.wantText != "" && (response.Message.Content == nil || *response.Message.Content != tt.wantText) {
				t.Errorf("Content = %v, want %q", response.Message.Content, tt.wantText)
			}
			if tt.wantTool != "" && (len(response.Message.ToolCalls) != 1 || response.Message.ToolCalls[0].Function.Name != tt.wantTool) {
				t.Errorf("ToolCalls = %#v, want tool %q", response.Message.ToolCalls, tt.wantTool)
			}
		})
	}
}

func TestIsRetryableStatus(t *testing.T) {
	tests := map[int]bool{
		http.StatusBadRequest:          false,
		http.StatusUnauthorized:        false,
		http.StatusPaymentRequired:     false,
		http.StatusNotFound:            false,
		http.StatusUnprocessableEntity: false,
		http.StatusTooManyRequests:     true,
		http.StatusInternalServerError: true,
		http.StatusNotImplemented:      false,
		http.StatusBadGateway:          false,
		http.StatusServiceUnavailable:  true,
		http.StatusGatewayTimeout:      false,
	}

	for status, want := range tests {
		if got := isRetryableStatus(status); got != want {
			t.Errorf("isRetryableStatus(%d) = %t, want %t", status, got, want)
		}
	}
}
