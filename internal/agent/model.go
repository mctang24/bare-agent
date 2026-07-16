package agent

import (
	"bare-agent/internal/tools"
	"context"
)

// Model is the provider-independent interface used by Agent.
type Model interface {
	GenerateResponse(context.Context, ModelRequest) (ModelResponse, error)
}

type ModelRequest struct {
	Instructions string
	Messages     []Message
	Tools        []ToolDefinition
}

type ModelResponse struct {
	Message Message
	Usage   TokenUsage
}

type TokenUsage struct {
	PromptTokens          int `json:"promptTokens,omitempty"`
	CompletionTokens      int `json:"completionTokens,omitempty"`
	TotalTokens           int `json:"totalTokens,omitempty"`
	PromptCacheHitTokens  int `json:"promptCacheHitTokens,omitempty"`
	PromptCacheMissTokens int `json:"promptCacheMissTokens,omitempty"`
}

type Message struct {
	Role        string
	Content     string
	ToolCalls   []ToolCall
	ToolResults []ToolResult
	RawMessage  []byte
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type ToolResult struct {
	ToolCallID string
	Content    string
	IsError    bool
}

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// modelTools returns the tool descriptions sent to the model.
func modelTools(available []tools.Tool) []ToolDefinition {
	definitions := make([]ToolDefinition, 0, len(available))
	for _, tool := range available {
		definitions = append(definitions, ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}
	return definitions
}
