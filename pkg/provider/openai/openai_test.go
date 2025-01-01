package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
)

func TestNewProvider(t *testing.T) {
	provider := NewProvider("test-key")

	if provider.apiKey != "test-key" {
		t.Errorf("API key not set correctly")
	}

	if provider.baseURL != defaultBaseURL {
		t.Errorf("Base URL not set to default")
	}

	if provider.client.Timeout != defaultTimeout {
		t.Errorf("Client timeout not set to default")
	}

	if provider.rateLimiter == nil {
		t.Error("Rate limiter not initialized")
	}

	if len(provider.modelLimits) == 0 {
		t.Error("Model limits not initialized")
	}
}

func TestGenerateResponse(t *testing.T) {
	tests := []struct {
		name        string
		config      config.ModelConfig
		apiResponse string
		wantError   bool
		wantContent string
	}{
		{
			name: "successful response",
			config: config.ModelConfig{
				Name:        "gpt-4",
				MaxTokens:   100,
				Temperature: 0.7,
				TopP:       0.9,
			},
			apiResponse: `{
				"choices": [
					{
						"message": {
							"content": "Hello, world!"
						}
					}
				]
			}`,
			wantError:   false,
			wantContent: "Hello, world!",
		},
		{
			name: "api error",
			config: config.ModelConfig{
				Name: "gpt-4",
			},
			apiResponse: `{
				"error": {
					"message": "Rate limit exceeded",
					"type": "rate_limit_exceeded"
				}
			}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Error("Authorization header not set correctly")
				}

				// Parse request body
				var reqBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}

				// Verify request parameters
				if reqBody["model"] != tt.config.Name {
					t.Errorf("Model name = %v, want %v", reqBody["model"], tt.config.Name)
				}

				// Return test response
				if strings.Contains(tt.apiResponse, "error") {
					w.WriteHeader(http.StatusBadRequest)
				}
				w.Write([]byte(tt.apiResponse))
			}))
			defer server.Close()

			// Create provider with test server
			provider := NewProvider("test-key")
			provider.SetBaseURL(server.URL)
			provider.SetTimeout(1 * time.Second)

			// Test response generation
			resp, err := provider.GenerateResponse(context.Background(), "Test prompt", tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("GenerateResponse() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && resp.Content != tt.wantContent {
				t.Errorf("Response content = %v, want %v", resp.Content, tt.wantContent)
			}
		})
	}
}

func TestModelLimits(t *testing.T) {
	provider := NewProvider("test-key")

	// Test GPT-4 limits
	gpt4Limits := provider.modelLimits["gpt-4"]
	if gpt4Limits.MaxTokens != 8192 {
		t.Errorf("GPT-4 max tokens = %v, want 8192", gpt4Limits.MaxTokens)
	}

	// Test GPT-3.5 limits
	gpt35Limits := provider.modelLimits["gpt-3.5-turbo"]
	if gpt35Limits.MaxTokens != 4096 {
		t.Errorf("GPT-3.5 max tokens = %v, want 4096", gpt35Limits.MaxTokens)
	}
}

func TestValidateTokens(t *testing.T) {
	provider := NewProvider("test-key")

	tests := []struct {
		name  string
		text  string
		want  int
	}{
		{
			name: "empty text",
			text: "",
			want: 0,
		},
		{
			name: "short text",
			text: "Hello, world!",
			want: 3, // ~12/4 characters per token
		},
		{
			name: "longer text",
			text: "This is a longer piece of text that should be counted as multiple tokens.",
			want: len("This is a longer piece of text that should be counted as multiple tokens.") / 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := provider.ValidateTokens(tt.text)
			if err != nil {
				t.Errorf("ValidateTokens() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateTokens() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomization(t *testing.T) {
	provider := NewProvider("test-key")

	// Test base URL customization
	customURL := "https://custom.openai.com"
	provider.SetBaseURL(customURL)
	if provider.baseURL != customURL {
		t.Errorf("SetBaseURL() failed, got %v, want %v", provider.baseURL, customURL)
	}

	// Test timeout customization
	customTimeout := 60 * time.Second
	provider.SetTimeout(customTimeout)
	if provider.client.Timeout != customTimeout {
		t.Errorf("SetTimeout() failed, got %v, want %v", provider.client.Timeout, customTimeout)
	}

	// Test rate limiter customization
	provider.SetRateLimiter(5, 5, time.Minute)
	if provider.rateLimiter == nil {
		t.Error("SetRateLimiter() failed to create new rate limiter")
	}
}
