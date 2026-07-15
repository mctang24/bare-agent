package main

import (
	"bare-agent/internal/agent"
	"context"
	"errors"
	"strings"
	"testing"
)

type interactiveModel struct {
	responses []agent.ModelResponse
	requests  []agent.ModelRequest
}

func (model *interactiveModel) GenerateResponse(_ context.Context, request agent.ModelRequest) (agent.ModelResponse, error) {
	model.requests = append(model.requests, request)
	response := model.responses[0]
	model.responses = model.responses[1:]
	return response, nil
}

func TestRunInteractive(t *testing.T) {
	model := &interactiveModel{responses: []agent.ModelResponse{
		{Message: agent.Message{Role: "assistant", Content: "first answer"}},
		{Message: agent.Message{Role: "assistant", Content: "second answer"}},
	}}
	runner, err := agent.NewAgent(t.TempDir(), model, "")
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	var output strings.Builder

	err = runInteractive(context.Background(), runner, strings.NewReader("first question\n/new\nsecond question\n/exit\n"), &output, &output)
	if err != nil {
		t.Fatalf("runInteractive() error = %v", err)
	}
	if output.String() != "> first answer\n> new conversation\n> second answer\n> " {
		t.Fatalf("runInteractive() output = %q", output.String())
	}
	if len(model.requests) != 2 || len(model.requests[1].Messages) != 1 || model.requests[1].Messages[0].Content != "second question" {
		t.Fatalf("model requests = %#v", model.requests)
	}
}

type failingInteractiveModel struct {
	calls int
}

func (model *failingInteractiveModel) GenerateResponse(_ context.Context, _ agent.ModelRequest) (agent.ModelResponse, error) {
	model.calls++
	if model.calls == 1 {
		return agent.ModelResponse{}, errors.New("failed")
	}
	return agent.ModelResponse{Message: agent.Message{Role: "assistant", Content: "done"}}, nil
}

func TestRunInteractiveContinuesAfterRunError(t *testing.T) {
	model := &failingInteractiveModel{}
	runner, err := agent.NewAgent(t.TempDir(), model, "")
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	var output strings.Builder
	var errorOutput strings.Builder

	err = runInteractive(context.Background(), runner, strings.NewReader("first\nsecond\n/exit\n"), &output, &errorOutput)
	if err != nil {
		t.Fatalf("runInteractive() error = %v", err)
	}
	if model.calls != 2 || output.String() != "> > done\n> " || !strings.Contains(errorOutput.String(), "failed") {
		t.Fatalf("calls = %d, output = %q, error output = %q", model.calls, output.String(), errorOutput.String())
	}
}
