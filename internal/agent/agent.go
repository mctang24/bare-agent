package agent

import (
	"bare-agent/internal/tools"
	"bare-agent/internal/trace"
	"context"
	"errors"
	"fmt"
	"time"
)

const defaultMaxTurns = 20

type Agent struct {
	root         string
	tools        []tools.Tool
	model        Model
	instructions string
	maxTurns     int
	messages     []Message
	traceWriter  *trace.Writer
	sessionID    string
}

// EnableTrace enables JSONL tracing for the agent session.
func (agent *Agent) EnableTrace(writer trace.Writer) error {
	if writer.Path == "" {
		return fmt.Errorf("enable trace: path is empty")
	}
	sessionID, err := newTraceID("session")
	if err != nil {
		return fmt.Errorf("enable trace: %w", err)
	}
	agent.traceWriter = &writer
	agent.sessionID = sessionID
	return nil
}

type RunResult struct {
	Content string
}

// NewAgent creates an agent with the read-only tools.
func NewAgent(root string, model Model, instructions string, maxTurns ...int) (*Agent, error) {
	if root == "" {
		return nil, fmt.Errorf("new agent: root is empty")
	}
	if model == nil {
		return nil, fmt.Errorf("new agent: model is nil")
	}
	if len(maxTurns) > 1 {
		return nil, fmt.Errorf("new agent: max turns accepts at most one value")
	}
	turns := defaultMaxTurns
	if len(maxTurns) == 1 {
		if maxTurns[0] <= 0 {
			return nil, fmt.Errorf("new agent: max turns must be positive")
		}
		turns = maxTurns[0]
	}

	return &Agent{
		root:         root,
		tools:        tools.ReadOnlyTools(),
		model:        model,
		instructions: instructions,
		maxTurns:     turns,
	}, nil
}

// Run continues the conversation until the model returns a final response.
func (agent *Agent) Run(ctx context.Context, task string) (result RunResult, runErr error) {
	if task == "" {
		return RunResult{}, fmt.Errorf("agent run: task is empty")
	}
	if agent.model == nil {
		return RunResult{}, fmt.Errorf("agent run: model is nil")
	}
	if agent.maxTurns <= 0 {
		return RunResult{}, fmt.Errorf("agent run: max turns must be positive")
	}

	definitions := modelTools(agent.tools)
	currentTrace, err := agent.startRunTrace(task, definitions)
	if err != nil {
		return RunResult{}, err
	}
	defer func() {
		if err := currentTrace.finish(runErr); err != nil {
			result = RunResult{}
			runErr = errors.Join(runErr, fmt.Errorf("agent run: %w", err))
		}
	}()

	messages := agent.messages
	messages = append(messages, Message{Role: "user", Content: task})
	for turn := 0; turn < agent.maxTurns; turn++ {
		request := ModelRequest{
			Instructions: agent.instructions,
			Messages:     messages,
			Tools:        definitions,
		}
		response, err := agent.callModel(ctx, request, currentTrace, turn+1)
		if err != nil {
			return RunResult{}, err
		}

		messages = append(messages, response.Message)
		if len(response.Message.ToolCalls) == 0 {
			agent.messages = messages
			return RunResult{Content: response.Message.Content}, nil
		}
		if turn+1 == agent.maxTurns {
			return RunResult{}, fmt.Errorf("agent run: reached maximum of %d turns", agent.maxTurns)
		}

		results := make([]ToolResult, 0, len(response.Message.ToolCalls))
		for _, call := range response.Message.ToolCalls {
			result, err := agent.callTool(ctx, call, currentTrace, turn+1)
			if err != nil {
				return RunResult{}, err
			}
			results = append(results, result)
		}
		messages = append(messages, Message{Role: "tool", ToolResults: results})
	}

	return RunResult{}, fmt.Errorf("agent run: reached maximum of %d turns", agent.maxTurns)
}

// Reset clears the conversation history and starts a new traced session.
func (agent *Agent) Reset() error {
	if agent.traceWriter != nil {
		newSessionID, err := newTraceID("session")
		if err != nil {
			return fmt.Errorf("reset agent: %w", err)
		}
		if err := agent.traceWriter.Append(trace.Event{
			Timestamp: time.Now().UTC(),
			SessionID: agent.sessionID,
			Type:      "session_reset",
			Data:      map[string]any{"newSessionId": newSessionID},
		}); err != nil {
			return fmt.Errorf("reset agent: %w", err)
		}
		agent.sessionID = newSessionID
	}
	agent.messages = nil
	return nil
}
