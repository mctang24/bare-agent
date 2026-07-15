package deepseek

import "testing"

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		baseURL     string
		model       string
		wantBaseURL string
		wantModel   string
		wantErr     bool
	}{
		{
			name:        "configured values",
			apiKey:      "test-key",
			baseURL:     "https://example.com",
			model:       "test-model",
			wantBaseURL: "https://example.com",
			wantModel:   "test-model",
		},
		{
			name:        "default values",
			apiKey:      "test-key",
			wantBaseURL: defaultBaseURL,
			wantModel:   defaultModel,
		},
		{
			name:    "empty API key",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.apiKey, tt.baseURL, tt.model)
			if tt.wantErr {
				if err == nil {
					t.Fatal("NewClient() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			if client.httpClient == nil {
				t.Error("NewClient() httpClient = nil")
			}
			if client.baseURL != tt.wantBaseURL {
				t.Errorf("baseURL = %q, want %q", client.baseURL, tt.wantBaseURL)
			}
			if client.model != tt.wantModel {
				t.Errorf("model = %q, want %q", client.model, tt.wantModel)
			}
		})
	}
}
