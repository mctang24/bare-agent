package deepseek

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

func (client *DeepSeekClient) createChatCompletion(_ context.Context, input chatCompletionRequest) (modelResponse, error) {
	if client.apiKey == "" {
		return modelResponse{}, fmt.Errorf("DeepSeek API key is empty")
	}
	if len(input.Messages) == 0 {
		return modelResponse{}, fmt.Errorf("DeepSeek chat completion has no messages")
	}

	requestBody, err := json.Marshal(struct {
		Model    string           `json:"model"`
		Messages []message        `json:"messages"`
		Tools    []toolDefinition `json:"tools,omitempty"`
		Stream   bool             `json:"stream"`
	}{
		Model:    client.model,
		Messages: input.Messages,
		Tools:    input.Tools,
		Stream:   true,
	})
	if err != nil {
		return modelResponse{}, fmt.Errorf("DeepSeek encode chat completion request: %w", err)
	}

	endpoint := strings.TrimRight(client.baseURL, "/") + "/chat/completions"
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return modelResponse{}, fmt.Errorf("DeepSeek create chat completion request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	request.Header.Set("Content-Type", "application/json")

	httpResponse, err := client.httpClient.Do(request)
	if err != nil {
		return modelResponse{}, fmt.Errorf("DeepSeek send chat completion request: %w", err)
	}
	if httpResponse.StatusCode >= http.StatusOK && httpResponse.StatusCode < http.StatusMultipleChoices {
		streamResponse, err := parseChatCompletionStream(httpResponse.Body)
		_ = httpResponse.Body.Close()
		return streamResponse, err
	}
	errorBody, err := io.ReadAll(httpResponse.Body)
	_ = httpResponse.Body.Close()
	if err != nil {
		return modelResponse{}, fmt.Errorf("DeepSeek read chat completion error response: %w", err)
	}
	return modelResponse{}, fmt.Errorf("DeepSeek chat completion returned %s: %s", httpResponse.Status, errorBody)
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

type chatCompletionChunk struct {
	Choices []struct {
		FinishReason *string `json:"finish_reason"`
		Delta        struct {
			Role             string  `json:"role"`
			Content          *string `json:"content"`
			ReasoningContent *string `json:"reasoning_content"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
}

type modelResponse struct {
	Message      assistantMessage
	FinishReason string
}

func parseChatCompletionStream(input io.Reader) (modelResponse, error) {
	response := modelResponse{}
	var content, reasoning strings.Builder
	contentSeen, reasoningSeen, roleSeen, done := false, false, false, false
	toolCalls := map[int]*toolCall{}

	reader := bufio.NewReader(input)
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return modelResponse{}, fmt.Errorf("DeepSeek read chat completion stream: %w", readErr)
		}

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				done = true
				break
			}

			var chunk chatCompletionChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				return modelResponse{}, fmt.Errorf("DeepSeek decode chat completion stream: %w", err)
			}
			for _, choice := range chunk.Choices {
				if role := choice.Delta.Role; role != "" && role != "assistant" {
					return modelResponse{}, fmt.Errorf("DeepSeek chat completion message role is %q, want assistant", role)
				}
				roleSeen = roleSeen || choice.Delta.Role == "assistant"
				if choice.Delta.Content != nil {
					contentSeen = true
					content.WriteString(*choice.Delta.Content)
				}
				if choice.Delta.ReasoningContent != nil {
					reasoningSeen = true
					reasoning.WriteString(*choice.Delta.ReasoningContent)
				}
				if choice.FinishReason != nil {
					response.FinishReason = *choice.FinishReason
				}
				for _, delta := range choice.Delta.ToolCalls {
					call := toolCalls[delta.Index]
					if call == nil {
						call = &toolCall{}
						toolCalls[delta.Index] = call
					}
					call.ID += delta.ID
					call.Type += delta.Type
					call.Function.Name += delta.Function.Name
					call.Function.Arguments += delta.Function.Arguments
				}
			}
		}
		if readErr == io.EOF {
			break
		}
	}
	if !done {
		return modelResponse{}, fmt.Errorf("DeepSeek chat completion stream ended before [DONE]")
	}
	if response.FinishReason == "" {
		return modelResponse{}, fmt.Errorf("DeepSeek chat completion stream has no finish reason")
	}
	if !roleSeen {
		return modelResponse{}, fmt.Errorf("DeepSeek chat completion stream has no message role")
	}
	response.Message.Role = "assistant"
	if contentSeen {
		value := content.String()
		response.Message.Content = &value
	}
	if reasoningSeen {
		value := reasoning.String()
		response.Message.ReasoningContent = &value
	}
	for index := 0; index < len(toolCalls); index++ {
		call := toolCalls[index]
		if call == nil {
			return modelResponse{}, fmt.Errorf("DeepSeek chat completion stream is missing tool call index %d", index)
		}
		response.Message.ToolCalls = append(response.Message.ToolCalls, *call)
	}
	return response, nil
}
