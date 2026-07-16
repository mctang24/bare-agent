package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bare-agent/internal/tools"
	"bare-agent/internal/trace"
)

type modelStub struct {
	responses []ModelResponse
	requests  []ModelRequest
}

func TestAgentEnableTrace(t *testing.T) {
	agent := Agent{}
	if err := agent.EnableTrace(trace.Writer{Path: filepath.Join(t.TempDir(), "trace.jsonl")}); err != nil {
		t.Fatalf("EnableTrace() error = %v", err)
	}
	if agent.traceWriter == nil || !strings.HasPrefix(agent.sessionID, "session_") {
		t.Fatalf("trace writer = %#v, session ID = %q", agent.traceWriter, agent.sessionID)
	}
	if err := agent.EnableTrace(trace.Writer{}); err == nil {
		t.Fatal("EnableTrace() error = nil, want empty path error")
	}
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
		{Message: Message{Role: "assistant", Content: "done"}},
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
	if result.Content != "done" {
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

func TestAgentRunTrace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	model := &modelStub{responses: []ModelResponse{
		{Message: Message{Role: "assistant", ToolCalls: []ToolCall{{ID: "call_1", Name: "fail", Arguments: `{}`}}}, Usage: TokenUsage{PromptTokens: 120, CompletionTokens: 30, TotalTokens: 150, PromptCacheHitTokens: 80, PromptCacheMissTokens: 40}},
		{Message: Message{Role: "assistant", Content: "done"}},
	}}
	agent := Agent{
		model:        model,
		maxTurns:     2,
		instructions: "inspect carefully",
		tools: []tools.Tool{{Name: "fail", Description: "always fails", Parameters: map[string]any{"type": "object"}, Execute: func(context.Context, string, string) (string, error) {
			return "", errors.New("tool failed")
		}}},
	}
	if err := agent.EnableTrace(trace.Writer{Path: path}); err != nil {
		t.Fatalf("EnableTrace() error = %v", err)
	}

	if _, err := agent.Run(context.Background(), "task"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	events := readTraceEvents(t, path)
	wantTypes := []string{"run_start", "model_request", "model_response", "tool_call", "tool_result", "model_request", "model_response", "run_end"}
	if len(events) != len(wantTypes) {
		t.Fatalf("trace event count = %d, want %d", len(events), len(wantTypes))
	}
	runID := events[0].RunID
	for index, event := range events {
		if event.Type != wantTypes[index] || event.SessionID != agent.sessionID || event.RunID != runID || event.Timestamp.IsZero() {
			t.Fatalf("trace event %d = %#v", index, event)
		}
	}
	if !strings.HasPrefix(runID, "run_") || events[1].Turn != 1 || events[5].Turn != 2 {
		t.Fatalf("run ID = %q, turns = %d, %d", runID, events[1].Turn, events[5].Turn)
	}
	if events[1].Data != nil || events[5].Data != nil {
		t.Fatalf("model request data = %#v, %#v", events[1].Data, events[5].Data)
	}
	modelResponse := events[2].Data.(map[string]any)
	usage := modelResponse["usage"].(map[string]any)
	if usage["promptTokens"] != float64(120) || usage["completionTokens"] != float64(30) || usage["totalTokens"] != float64(150) || usage["promptCacheHitTokens"] != float64(80) || usage["promptCacheMissTokens"] != float64(40) {
		t.Fatalf("model response usage = %#v", usage)
	}
	if modelResponseWithoutUsage := events[6].Data.(map[string]any); modelResponseWithoutUsage["usage"] != nil {
		t.Fatalf("model response without usage = %#v", modelResponseWithoutUsage)
	}
	runStart := events[0].Data.(map[string]any)
	traceTools := runStart["tools"].([]any)
	traceTool := traceTools[0].(map[string]any)
	if len(runStart) != 3 || runStart["task"] != "task" || runStart["instructions"] != "inspect carefully" || len(traceTools) != 1 || traceTool["name"] != "fail" || traceTool["description"] != "always fails" {
		t.Fatalf("run start data = %#v", runStart)
	}
	toolResult := events[4].Data.(map[string]any)
	if toolResult["isError"] != true || !strings.Contains(toolResult["content"].(string), "tool failed") {
		t.Fatalf("tool result data = %#v", toolResult)
	}
	runEnd := events[7].Data.(map[string]any)
	if runEnd["status"] != "success" {
		t.Fatalf("run end data = %#v", runEnd)
	}
}

func TestAgentRunTraceError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	agent := Agent{model: errorModel{err: errors.New("model failed")}, maxTurns: 1}
	if err := agent.EnableTrace(trace.Writer{Path: path}); err != nil {
		t.Fatalf("EnableTrace() error = %v", err)
	}

	if _, err := agent.Run(context.Background(), "task"); err == nil {
		t.Fatal("Run() error = nil")
	}

	events := readTraceEvents(t, path)
	wantTypes := []string{"run_start", "model_request", "model_response", "run_end"}
	if len(events) != len(wantTypes) {
		t.Fatalf("trace event count = %d, want %d", len(events), len(wantTypes))
	}
	for index, event := range events {
		if event.Type != wantTypes[index] {
			t.Fatalf("trace event %d type = %q, want %q", index, event.Type, wantTypes[index])
		}
	}
	response := events[2].Data.(map[string]any)
	runEnd := events[3].Data.(map[string]any)
	if response["error"] != "model failed" || runEnd["status"] != "error" || !strings.Contains(runEnd["error"].(string), "model failed") {
		t.Fatalf("model response = %#v, run end = %#v", response, runEnd)
	}
}

func readTraceEvents(t *testing.T, path string) []trace.Event {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	events := make([]trace.Event, 0, len(lines))
	for _, line := range lines {
		var event trace.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode trace: %v", err)
		}
		events = append(events, event)
	}
	return events
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
	if err := agent.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if _, err := agent.Run(context.Background(), "second question"); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	messages := model.requests[1].Messages
	if len(messages) != 1 || messages[0].Content != "second question" {
		t.Fatalf("second request messages = %#v", messages)
	}
}

func TestAgentResetTrace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	agent := Agent{messages: []Message{{Role: "user", Content: "old"}}}
	if err := agent.EnableTrace(trace.Writer{Path: path}); err != nil {
		t.Fatalf("EnableTrace() error = %v", err)
	}
	oldSessionID := agent.sessionID
	if err := agent.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if len(agent.messages) != 0 || agent.sessionID == oldSessionID || !strings.HasPrefix(agent.sessionID, "session_") {
		t.Fatalf("messages = %#v, session ID = %q", agent.messages, agent.sessionID)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	var event trace.Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("decode trace: %v", err)
	}
	if event.Type != "session_reset" || event.SessionID != oldSessionID {
		t.Fatalf("trace event = %#v", event)
	}

	failed := Agent{messages: []Message{{Role: "user", Content: "keep"}}}
	badPath := filepath.Join(t.TempDir(), "missing", "trace.jsonl")
	if err := failed.EnableTrace(trace.Writer{Path: badPath}); err != nil {
		t.Fatalf("EnableTrace() error = %v", err)
	}
	failedSessionID := failed.sessionID
	if err := failed.Reset(); err == nil {
		t.Fatal("Reset() error = nil, want trace write error")
	}
	if len(failed.messages) != 1 || failed.sessionID != failedSessionID {
		t.Fatalf("failed reset messages = %#v, session ID = %q", failed.messages, failed.sessionID)
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
