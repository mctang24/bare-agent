package agent

import (
	"reflect"
	"testing"

	"bare-agent/internal/tools"
)

func TestModelTools(t *testing.T) {
	parameters := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string"},
		},
	}
	available := []tools.Tool{
		{
			Name:        "read_file",
			Description: "Read a file.",
			Parameters:  parameters,
		},
	}

	got := modelTools(available)
	want := []ToolDefinition{
		{
			Name:        "read_file",
			Description: "Read a file.",
			Parameters:  parameters,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("modelTools() = %#v, want %#v", got, want)
	}
}

func TestMessageKeepsMultipleToolResults(t *testing.T) {
	message := Message{
		Role: "tool",
		ToolResults: []ToolResult{
			{ToolCallID: "call_1", Content: "first"},
			{ToolCallID: "call_2", Content: "failed", IsError: true},
		},
	}

	if len(message.ToolResults) != 2 {
		t.Fatalf("tool results = %d, want 2", len(message.ToolResults))
	}
	if message.ToolResults[0].ToolCallID != "call_1" || message.ToolResults[0].Content != "first" {
		t.Fatalf("first tool result = %#v", message.ToolResults[0])
	}
	if message.ToolResults[1].ToolCallID != "call_2" || !message.ToolResults[1].IsError {
		t.Fatalf("second tool result = %#v", message.ToolResults[1])
	}
}

func TestModelRequestKeepsInstructions(t *testing.T) {
	request := ModelRequest{Instructions: "Use tools before answering."}

	if request.Instructions != "Use tools before answering." {
		t.Fatalf("instructions = %q", request.Instructions)
	}
}

func TestMessageKeepsRawMessage(t *testing.T) {
	raw := []byte(`{"reasoning_content":"inspect files first"}`)
	message := Message{RawMessage: raw}

	if string(message.RawMessage) != string(raw) {
		t.Fatalf("raw message = %s, want %s", message.RawMessage, raw)
	}
}

func TestContextUsagePercent(t *testing.T) {
	tests := []struct {
		tokens int
		want   int
	}{
		{tokens: 0, want: 0},
		{tokens: 800_000, want: 80},
		{tokens: 900_000, want: 90},
		{tokens: 1_100_000, want: 110},
	}

	for _, tt := range tests {
		if got := contextUsagePercent(tt.tokens); got != tt.want {
			t.Errorf("contextUsagePercent(%d) = %d, want %d", tt.tokens, got, tt.want)
		}
	}
}
