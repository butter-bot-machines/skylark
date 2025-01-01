package openai

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
	"github.com/butter-bot-machines/skylark/pkg/provider"
)

// Provider implements the provider interface for OpenAI
type Provider struct {
	client     *http.Client
	config     config.ModelConfig
	rateLimits *RateLimiter
}

// New creates a new OpenAI provider
func New(cfg config.ModelConfig) (*Provider, error) {
	if cfg.Provider != "openai" {
		return nil, fmt.Errorf("invalid provider: %s", cfg.Provider)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &Provider{
		client: client,
		config: cfg,
		rateLimits: NewRateLimiter(RateLimitConfig{
			RequestsPerMinute: 3, // Lower limit for testing
			TokensPerMinute:   1000,
		}),
	}, nil
}

// Send sends a prompt to OpenAI and returns the response
func (p *Provider) Send(ctx context.Context, prompt string) (*provider.Response, error) {
	// First check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Check rate limits
	if err := p.rateLimits.Wait(ctx); err != nil {
		return nil, err
	}

	// Build request parameters
	params := map[string]interface{}{
		"model":             p.config.Name,
		"prompt":            prompt,
		"max_tokens":        p.config.MaxTokens,
		"temperature":       p.config.Temperature,
		"top_p":            p.config.TopP,
		"frequency_penalty": p.config.FrequencyPenalty,
		"presence_penalty":  p.config.PresencePenalty,
	}

	// Add any additional parameters from config
	for k, v := range p.config.Parameters {
		params[k] = v
	}

	// TODO: Make actual API call
	// For now, simulate API behavior for testing
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(10 * time.Millisecond):
		// Simulate token usage
		if err := p.rateLimits.AddTokens(100); err != nil {
			return nil, err
		}
	}

	response := &provider.Response{
		Content: "Sample response",
		Usage: provider.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	return response, nil
}

// Close cleans up any resources
func (p *Provider) Close() error {
	return nil
}
