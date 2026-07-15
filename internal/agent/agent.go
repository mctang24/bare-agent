package agent

import (
	"bare-agent/internal/tools"
	"context"
	"fmt"
)

const defaultMaxTurns = 20

type Agent struct {
	root         string
	tools        []tools.Tool
	model        Model
	instructions string
	maxTurns     int
	messages     []Message
}

type RunResult struct {
	Content             string
	ContextUsagePercent int
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
func (agent *Agent) Run(ctx context.Context, task string) (RunResult, error) {
	if task == "" {
		return RunResult{}, fmt.Errorf("agent run: task is empty")
	}
	if agent.model == nil {
		return RunResult{}, fmt.Errorf("agent run: model is nil")
	}
	if agent.maxTurns <= 0 {
		return RunResult{}, fmt.Errorf("agent run: max turns must be positive")
	}

	messages := agent.messages
	messages = append(messages, Message{Role: "user", Content: task})
	for turn := 0; turn < agent.maxTurns; turn++ {
		response, err := agent.model.GenerateResponse(ctx, ModelRequest{
			Instructions: agent.instructions,
			Messages:     messages,
			Tools:        modelTools(agent.tools),
		})
		if err != nil {
			return RunResult{}, fmt.Errorf("agent run: generate response: %w", err)
		}

		messages = append(messages, response.Message)
		if len(response.Message.ToolCalls) == 0 {
			agent.messages = messages
			return RunResult{
				Content:             response.Message.Content,
				ContextUsagePercent: contextUsagePercent(response.Usage.TotalTokens),
			}, nil
		}
		if turn+1 == agent.maxTurns {
			return RunResult{}, fmt.Errorf("agent run: reached maximum of %d turns", agent.maxTurns)
		}

		results := make([]ToolResult, 0, len(response.Message.ToolCalls))
		for _, call := range response.Message.ToolCalls {
			result, err := agent.executeToolCall(ctx, call)
			if err != nil {
				return RunResult{}, fmt.Errorf("agent run: %w", err)
			}
			results = append(results, result)
		}
		messages = append(messages, Message{Role: "tool", ToolResults: results})
	}

	return RunResult{}, fmt.Errorf("agent run: reached maximum of %d turns", agent.maxTurns)
}

// Reset clears the conversation history.
func (agent *Agent) Reset() {
	agent.messages = nil
}
