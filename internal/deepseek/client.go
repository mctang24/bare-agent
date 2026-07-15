package deepseek

import (
	"fmt"
	"net/http"
)

const (
	defaultBaseURL = "https://api.deepseek.com"
	defaultModel   = "deepseek-v4-flash"
)

type DeepSeekClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
}

func NewClient(apiKey, baseURL, model string) (*DeepSeekClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is empty")
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if model == "" {
		model = defaultModel
	}

	return &DeepSeekClient{
		httpClient: &http.Client{},
		baseURL:    baseURL,
		apiKey:     apiKey,
		model:      model,
	}, nil
}
