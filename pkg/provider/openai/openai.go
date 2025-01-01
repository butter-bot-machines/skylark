package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/provider"
)

const (
	defaultBaseURL = "https://api.openai.com/v1"
	defaultTimeout = 30 * time.Second
)

// Provider implements the provider.Provider interface for OpenAI
type Provider struct {
	apiKey      string
	baseURL     string
	client      *http.Client
	rateLimiter *provider.RateLimiter
	modelLimits map[string]provider.ModelLimits
}

// NewProvider creates a new OpenAI provider
func NewProvider(apiKey string) *Provider {
	// Initialize with default rate limits (3 requests per minute)
	rateLimiter := provider.NewRateLimiter(3, 3, time.Minute)

	// Initialize model limits
	modelLimits := map[string]provider.ModelLimits{
		"gpt-4": {
			MaxTokens:      8192,
			MaxPromptSize:  24576,  // 75% of context window
			MaxOutputSize:  8192,   // 25% of context window
			MaxContextSize: 32768,
		},
		"gpt-3.5-turbo": {
			MaxTokens:      4096,
			MaxPromptSize:  12288,  // 75% of context window
			MaxOutputSize:  4096,   // 25% of context window
			MaxContextSize: 16384,
		},
	}

	return &Provider{
		apiKey:      apiKey,
		baseURL:     defaultBaseURL,
		client:      &http.Client{Timeout: defaultTimeout},
		rateLimiter: rateLimiter,
		modelLimits: modelLimits,
	}
}

// GenerateResponse generates a response from the OpenAI API
func (p *Provider) GenerateResponse(ctx context.Context, prompt string, config config.ModelConfig) (*provider.Response, error) {
	// Wait for rate limiter
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Validate prompt against model limits
	limits := p.modelLimits[config.Name]
	if err := provider.ValidatePrompt(prompt, limits); err != nil {
		return nil, err
	}

	// Prepare request
	reqBody := map[string]interface{}{
		"model":       config.Name,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":   config.MaxTokens,
		"temperature":  config.Temperature,
		"top_p":       config.TopP,
		"stream":      false,
	}

	// Add any additional parameters
	for k, v := range config.Parameters {
		reqBody[k] = v
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", strings.NewReader(string(reqJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:    "connection_error",
			Message: err.Error(),
		}
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}

		return nil, &provider.ProviderError{
			Code:    errResp.Error.Type,
			Message: errResp.Error.Message,
		}
	}

	// Parse response
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices available")
	}

	return &provider.Response{
		Content: strings.TrimSpace(apiResp.Choices[0].Message.Content),
	}, nil
}

// ValidateTokens checks if text is within token limits
func (p *Provider) ValidateTokens(text string) (int, error) {
	// TODO: Implement proper token counting
	// For now, use a rough approximation (4 chars per token)
	return len(text) / 4, nil
}

// GetModelLimits returns the model's token limits
func (p *Provider) GetModelLimits() provider.ModelLimits {
	// Return limits for GPT-4 by default
	return p.modelLimits["gpt-4"]
}

// SetBaseURL sets a custom base URL for the API
func (p *Provider) SetBaseURL(url string) {
	p.baseURL = url
}

// SetTimeout sets a custom timeout for API requests
func (p *Provider) SetTimeout(timeout time.Duration) {
	p.client.Timeout = timeout
}

// SetRateLimiter sets a custom rate limiter
func (p *Provider) SetRateLimiter(tokens, refillRate int, interval time.Duration) {
	p.rateLimiter = provider.NewRateLimiter(tokens, refillRate, interval)
}
