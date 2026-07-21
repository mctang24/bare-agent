package main

import (
	"bare-agent/internal/agent"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type interactiveModel struct {
	responses []agent.ModelResponse
	requests  []agent.ModelRequest
}

func (model *interactiveModel) GenerateResponse(_ context.Context, request agent.ModelRequest) (agent.ModelStream, error) {
	model.requests = append(model.requests, request)
	response := model.responses[0]
	model.responses = model.responses[1:]
	return &interactiveStream{response: response}, nil
}

type interactiveStream struct {
	response agent.ModelResponse
	sentText bool
	finished bool
	err      error
}

func (stream *interactiveStream) Recv() (agent.ModelStreamEvent, error) {
	if stream.finished {
		return agent.ModelStreamEvent{}, io.EOF
	}
	if stream.sentText {
		if stream.err != nil {
			return agent.ModelStreamEvent{}, stream.err
		}
		stream.finished = true
		return agent.ModelStreamEvent{Response: &stream.response}, nil
	}
	stream.sentText = true
	return agent.ModelStreamEvent{TextDelta: stream.response.Message.Content}, nil
}

func (stream *interactiveStream) Close() error { return nil }

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

func (model *failingInteractiveModel) GenerateResponse(context.Context, agent.ModelRequest) (agent.ModelStream, error) {
	model.calls++
	if model.calls == 1 {
		return &interactiveStream{
			response: agent.ModelResponse{Message: agent.Message{Role: "assistant", Content: "partial"}},
			err:      errors.New("failed"),
		}, nil
	}
	response := agent.ModelResponse{Message: agent.Message{Role: "assistant", Content: "done"}}
	return &interactiveStream{response: response}, nil
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
	if model.calls != 2 || output.String() != "> partial\n> done\n> " || !strings.Contains(errorOutput.String(), "failed") {
		t.Fatalf("calls = %d, output = %q, error output = %q", model.calls, output.String(), errorOutput.String())
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRunTaskReturnsOutputError(t *testing.T) {
	model := &interactiveModel{responses: []agent.ModelResponse{{Message: agent.Message{Role: "assistant", Content: "done"}}}}
	runner, err := agent.NewAgent(t.TempDir(), model, "")
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}

	err = runTask(context.Background(), runner, "task", failingWriter{})
	if err == nil || !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("runTask() error = %v, want write failure", err)
	}
}
