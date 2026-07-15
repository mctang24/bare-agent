package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"bare-agent/internal/tools"
)

type modelStub struct {
	responses []ModelResponse
	requests  []ModelRequest
}

func TestNewAgent(t *testing.T) {
	model := &modelStub{}
	created, err := NewAgent(t.TempDir(), model, "inspect")
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	if created.model != model || created.instructions != "inspect" || created.maxTurns != defaultMaxTurns {
		t.Fatalf("NewAgent() = %#v", created)
	}
	if len(created.tools) != 3 {
		t.Fatalf("NewAgent() tool count = %d, want 3", len(created.tools))
	}

	configured, err := NewAgent(t.TempDir(), model, "", 3)
	if err != nil {
		t.Fatalf("NewAgent() with max turns error = %v", err)
	}
	if configured.maxTurns != 3 {
		t.Fatalf("NewAgent() max turns = %d, want 3", configured.maxTurns)
	}
}

func TestNewAgentErrors(t *testing.T) {
	model := &modelStub{}
	tests := []struct {
		name     string
		root     string
		model    Model
		maxTurns []int
		wantErr  string
	}{
		{name: "empty root", model: model, wantErr: "root is empty"},
		{name: "nil model", root: t.TempDir(), wantErr: "model is nil"},
		{name: "invalid max turns", root: t.TempDir(), model: model, maxTurns: []int{0}, wantErr: "max turns must be positive"},
		{name: "multiple max turns", root: t.TempDir(), model: model, maxTurns: []int{1, 2}, wantErr: "at most one"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAgent(tt.root, tt.model, "", tt.maxTurns...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("NewAgent() error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
}

func (stub *modelStub) GenerateResponse(_ context.Context, request ModelRequest) (ModelResponse, error) {
	stub.requests = append(stub.requests, request)
	response := stub.responses[0]
	stub.responses = stub.responses[1:]
	return response, nil
}

func TestAgentRun(t *testing.T) {
	model := &modelStub{responses: []ModelResponse{
		{Message: Message{Role: "assistant", ToolCalls: []ToolCall{{ID: "call_1", Name: "echo", Arguments: `{}`}}}},
		{Message: Message{Role: "assistant", Content: "done"}, Usage: TokenUsage{TotalTokens: 900_000}},
	}}
	agent := Agent{
		model:        model,
		maxTurns:     2,
		instructions: "inspect",
		tools: []tools.Tool{{Name: "echo", Execute: func(context.Context, string, string) (string, error) {
			return "result", nil
		}}},
	}

	result, err := agent.Run(context.Background(), "task")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Content != "done" || result.ContextUsagePercent != 90 {
		t.Fatalf("Run() = %#v", result)
	}
	if len(model.requests) != 2 || len(model.requests[1].Messages) != 3 {
		t.Fatalf("model requests = %#v", model.requests)
	}
	toolMessage := model.requests[1].Messages[2]
	if toolMessage.Role != "tool" || len(toolMessage.ToolResults) != 1 || toolMessage.ToolResults[0].Content != "result" {
		t.Fatalf("tool message = %#v", toolMessage)
	}
}

func TestAgentRunContinuesConversation(t *testing.T) {
	model := &modelStub{responses: []ModelResponse{
		{Message: Message{Role: "assistant", Content: "first answer"}},
		{Message: Message{Role: "assistant", Content: "second answer"}},
	}}
	agent := Agent{model: model, maxTurns: 1}

	if _, err := agent.Run(context.Background(), "first question"); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	if _, err := agent.Run(context.Background(), "second question"); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	messages := model.requests[1].Messages
	if len(messages) != 3 || messages[0].Content != "first question" || messages[1].Content != "first answer" || messages[2].Content != "second question" {
		t.Fatalf("second request messages = %#v", messages)
	}
}

func TestAgentRunDiscardsFailedConversation(t *testing.T) {
	model := &modelStub{responses: []ModelResponse{
		{Message: Message{Role: "assistant", ToolCalls: []ToolCall{{ID: "call_1"}}}},
		{Message: Message{Role: "assistant", Content: "done"}},
	}}
	agent := Agent{model: model, maxTurns: 1}

	if _, err := agent.Run(context.Background(), "failed question"); err == nil {
		t.Fatal("first Run() error = nil")
	}
	if _, err := agent.Run(context.Background(), "new question"); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	messages := model.requests[1].Messages
	if len(messages) != 1 || messages[0].Content != "new question" {
		t.Fatalf("second request messages = %#v", messages)
	}
}

func TestAgentReset(t *testing.T) {
	model := &modelStub{responses: []ModelResponse{
		{Message: Message{Role: "assistant", Content: "first answer"}},
		{Message: Message{Role: "assistant", Content: "second answer"}},
	}}
	agent := Agent{model: model, maxTurns: 1}

	if _, err := agent.Run(context.Background(), "first question"); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	agent.Reset()
	if _, err := agent.Run(context.Background(), "second question"); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	messages := model.requests[1].Messages
	if len(messages) != 1 || messages[0].Content != "second question" {
		t.Fatalf("second request messages = %#v", messages)
	}
}

func TestAgentRunErrors(t *testing.T) {
	tests := []struct {
		name    string
		agent   Agent
		ctx     context.Context
		task    string
		wantErr string
	}{
		{name: "empty task", agent: Agent{}, ctx: context.Background(), wantErr: "task is empty"},
		{name: "nil model", agent: Agent{maxTurns: 1}, ctx: context.Background(), task: "task", wantErr: "model is nil"},
		{name: "invalid max turns", agent: Agent{model: &modelStub{}}, ctx: context.Background(), task: "task", wantErr: "max turns must be positive"},
		{name: "maximum turns", agent: Agent{model: &modelStub{responses: []ModelResponse{{Message: Message{ToolCalls: []ToolCall{{ID: "call_1"}}}}}}, maxTurns: 1}, ctx: context.Background(), task: "task", wantErr: "reached maximum"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.agent.Run(tt.ctx, tt.task)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Run() error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestAgentRunModelError(t *testing.T) {
	model := errorModel{err: errors.New("failed")}
	agent := Agent{model: model, maxTurns: 1}
	_, err := agent.Run(context.Background(), "task")
	if err == nil || !strings.Contains(err.Error(), "generate response: failed") {
		t.Fatalf("Run() error = %v", err)
	}
}

type errorModel struct {
	err error
}

func (model errorModel) GenerateResponse(context.Context, ModelRequest) (ModelResponse, error) {
	return ModelResponse{}, model.err
}
