package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type message struct {
	Role             string     `json:"role"`
	Content          *string    `json:"content"`
	ReasoningContent *string    `json:"reasoning_content,omitempty"`
	ToolCalls        []toolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
}

type functionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type toolDefinition struct {
	Type     string             `json:"type"`
	Function functionDefinition `json:"function"`
}

type chatCompletionRequest struct {
	Messages []message
	Tools    []toolDefinition
}

func isRetryableStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusServiceUnavailable:
		return true
	default:
		return false
	}
}

func (client *DeepSeekClient) createChatCompletion(_ context.Context, input chatCompletionRequest) (modelResponse, error) {
	if client.apiKey == "" {
		return modelResponse{}, fmt.Errorf("DeepSeek API key is empty")
	}
	if len(input.Messages) == 0 {
		return modelResponse{}, fmt.Errorf("DeepSeek chat completion has no messages")
	}

	body, err := json.Marshal(struct {
		Model    string           `json:"model"`
		Messages []message        `json:"messages"`
		Tools    []toolDefinition `json:"tools,omitempty"`
		Stream   bool             `json:"stream"`
	}{
		Model:    client.model,
		Messages: input.Messages,
		Tools:    input.Tools,
		Stream:   false,
	})
	if err != nil {
		return modelResponse{}, fmt.Errorf("DeepSeek encode chat completion request: %w", err)
	}

	url := strings.TrimRight(client.baseURL, "/") + "/chat/completions"
	for attempt := 0; attempt <= 2; attempt++ {
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return modelResponse{}, fmt.Errorf("DeepSeek create chat completion request: %w", err)
		}
		request.Header.Set("Authorization", "Bearer "+client.apiKey)
		request.Header.Set("Content-Type", "application/json")

		response, err := client.httpClient.Do(request)
		if err != nil {
			return modelResponse{}, fmt.Errorf("DeepSeek send chat completion request: %w", err)
		}
		responseBody, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			return modelResponse{}, fmt.Errorf("DeepSeek read chat completion response: %w", err)
		}
		if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
			return parseChatCompletion(responseBody)
		}
		err = fmt.Errorf("DeepSeek chat completion returned %s: %s", response.Status, responseBody)
		if !isRetryableStatus(response.StatusCode) {
			return modelResponse{}, err
		}
		if attempt == 2 {
			return modelResponse{}, err
		}

		time.Sleep(time.Duration(1<<attempt) * 500 * time.Millisecond)
	}
	return modelResponse{}, fmt.Errorf("DeepSeek chat completion retry exhausted")
}

type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type assistantMessage struct {
	Role             string     `json:"role"`
	Content          *string    `json:"content"`
	ReasoningContent *string    `json:"reasoning_content"`
	ToolCalls        []toolCall `json:"tool_calls"`
}

type chatCompletion struct {
	Choices []struct {
		FinishReason string           `json:"finish_reason"`
		Message      assistantMessage `json:"message"`
	} `json:"choices"`
	Usage tokenUsage `json:"usage"`
}

type tokenUsage struct {
	PromptTokens          int `json:"prompt_tokens"`
	CompletionTokens      int `json:"completion_tokens"`
	TotalTokens           int `json:"total_tokens"`
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
}

type modelResponse struct {
	Message      assistantMessage
	FinishReason string
	Usage        tokenUsage
}

func parseChatCompletion(data []byte) (modelResponse, error) {
	var completion chatCompletion
	if err := json.Unmarshal(data, &completion); err != nil {
		return modelResponse{}, fmt.Errorf("DeepSeek decode chat completion: %w", err)
	}
	if len(completion.Choices) == 0 {
		return modelResponse{}, fmt.Errorf("DeepSeek chat completion has no choices")
	}
	choice := completion.Choices[0]
	if choice.Message.Role != "assistant" {
		return modelResponse{}, fmt.Errorf("DeepSeek chat completion message role is %q, want assistant", choice.Message.Role)
	}

	return modelResponse{
		Message:      choice.Message,
		FinishReason: choice.FinishReason,
		Usage:        completion.Usage,
	}, nil
}
