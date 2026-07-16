package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"bare-agent/internal/trace"
)

type runTrace struct {
	writer    *trace.Writer
	sessionID string
	runID     string
	startedAt time.Time
}

type traceToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

func newTraceID(prefix string) (string, error) {
	if prefix == "" {
		return "", fmt.Errorf("generate trace ID: prefix is empty")
	}
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate trace ID: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(bytes), nil
}

func (agent *Agent) startRunTrace(task string, definitions []ToolDefinition) (*runTrace, error) {
	if agent.traceWriter == nil {
		return nil, nil
	}
	runID, err := newTraceID("run")
	if err != nil {
		return nil, fmt.Errorf("agent run: %w", err)
	}
	startedAt := time.Now()
	current := &runTrace{writer: agent.traceWriter, sessionID: agent.sessionID, runID: runID, startedAt: startedAt}
	traceTools := make([]traceToolDefinition, 0, len(definitions))
	for _, definition := range definitions {
		traceTools = append(traceTools, traceToolDefinition{
			Name:        definition.Name,
			Description: definition.Description,
			Parameters:  definition.Parameters,
		})
	}
	if err := current.append(trace.Event{
		Timestamp: startedAt.UTC(),
		Type:      "run_start",
		Data: map[string]any{
			"task":         task,
			"instructions": agent.instructions,
			"tools":        traceTools,
		},
	}); err != nil {
		return nil, fmt.Errorf("agent run: %w", err)
	}
	return current, nil
}

func (current *runTrace) append(event trace.Event) error {
	if current == nil {
		return nil
	}
	event.SessionID = current.sessionID
	event.RunID = current.runID
	return current.writer.Append(event)
}

func (current *runTrace) finish(runErr error) error {
	if current == nil {
		return nil
	}
	data := map[string]any{"status": "success"}
	if runErr != nil {
		data["status"] = "error"
		data["error"] = runErr.Error()
	}
	return current.append(trace.Event{
		Timestamp:  time.Now().UTC(),
		Type:       "run_end",
		DurationMS: time.Since(current.startedAt).Milliseconds(),
		Data:       data,
	})
}

func (agent *Agent) callModel(ctx context.Context, request ModelRequest, current *runTrace, turn int) (ModelResponse, error) {
	startedAt := time.Now()
	if err := current.append(trace.Event{
		Timestamp: startedAt.UTC(),
		Type:      "model_request",
		Turn:      turn,
	}); err != nil {
		return ModelResponse{}, fmt.Errorf("agent run: %w", err)
	}

	response, err := agent.model.GenerateResponse(ctx, request)
	if err != nil {
		traceErr := current.append(trace.Event{
			Timestamp:  time.Now().UTC(),
			Type:       "model_response",
			Turn:       turn,
			DurationMS: time.Since(startedAt).Milliseconds(),
			Data:       map[string]any{"error": err.Error()},
		})
		modelErr := fmt.Errorf("agent run: generate response: %w", err)
		if traceErr != nil {
			return ModelResponse{}, errors.Join(modelErr, fmt.Errorf("agent run: %w", traceErr))
		}
		return ModelResponse{}, modelErr
	}

	data := map[string]any{"content": response.Message.Content, "toolCalls": response.Message.ToolCalls}
	if response.Usage != (TokenUsage{}) {
		data["usage"] = response.Usage
	}
	if err := current.append(trace.Event{
		Timestamp:  time.Now().UTC(),
		Type:       "model_response",
		Turn:       turn,
		DurationMS: time.Since(startedAt).Milliseconds(),
		Data:       data,
	}); err != nil {
		return ModelResponse{}, fmt.Errorf("agent run: %w", err)
	}
	return response, nil
}

func (agent *Agent) callTool(ctx context.Context, call ToolCall, current *runTrace, turn int) (ToolResult, error) {
	startedAt := time.Now()
	if err := current.append(trace.Event{
		Timestamp: startedAt.UTC(),
		Type:      "tool_call",
		Turn:      turn,
		Data:      map[string]any{"id": call.ID, "name": call.Name, "arguments": call.Arguments},
	}); err != nil {
		return ToolResult{}, fmt.Errorf("agent run: %w", err)
	}

	result, err := agent.executeToolCall(ctx, call)
	if err != nil {
		traceErr := current.append(trace.Event{
			Timestamp:  time.Now().UTC(),
			Type:       "tool_result",
			Turn:       turn,
			DurationMS: time.Since(startedAt).Milliseconds(),
			Data:       map[string]any{"id": call.ID, "error": err.Error()},
		})
		toolErr := fmt.Errorf("agent run: %w", err)
		if traceErr != nil {
			return ToolResult{}, errors.Join(toolErr, fmt.Errorf("agent run: %w", traceErr))
		}
		return ToolResult{}, toolErr
	}

	if err := current.append(trace.Event{
		Timestamp:  time.Now().UTC(),
		Type:       "tool_result",
		Turn:       turn,
		DurationMS: time.Since(startedAt).Milliseconds(),
		Data:       map[string]any{"id": call.ID, "content": result.Content, "isError": result.IsError},
	}); err != nil {
		return ToolResult{}, fmt.Errorf("agent run: %w", err)
	}
	return result, nil
}
