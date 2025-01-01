package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
)

// Response represents an AI model's response
type Response struct {
	Content string
	Error   error
}

// Provider defines the interface for AI model providers
type Provider interface {
	// GenerateResponse generates a response from the AI model
	GenerateResponse(ctx context.Context, prompt string, config config.ModelConfig) (*Response, error)
	// ValidateTokens checks if text is within token limits
	ValidateTokens(text string) (int, error)
	// GetModelLimits returns the model's token limits
	GetModelLimits() ModelLimits
}

// ModelLimits represents token limits for a model
type ModelLimits struct {
	MaxTokens      int
	MaxPromptSize  int
	MaxOutputSize  int
	MaxContextSize int
}

// RateLimiter implements rate limiting for API calls
type RateLimiter struct {
	tokens         int
	refillRate     int
	refillInterval time.Duration
	lastRefill     time.Time
	mu            sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(tokens, refillRate int, refillInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:         tokens,
		refillRate:     refillRate,
		refillInterval: refillInterval,
		lastRefill:     time.Now(),
	}
}

// Wait waits until a token is available
func (r *RateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Refill tokens if enough time has passed
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	if elapsed >= r.refillInterval {
		intervals := int(elapsed / r.refillInterval)
		r.tokens += r.refillRate * intervals
		r.lastRefill = now.Add(-elapsed % r.refillInterval)
	}

	// Wait for token if none available
	for r.tokens <= 0 {
		waitTime := r.refillInterval - elapsed
		r.mu.Unlock()

		select {
		case <-time.After(waitTime):
			r.mu.Lock()
			now = time.Now()
			elapsed = now.Sub(r.lastRefill)
			intervals := int(elapsed / r.refillInterval)
			r.tokens += r.refillRate * intervals
			r.lastRefill = now.Add(-elapsed % r.refillInterval)
		case <-ctx.Done():
			r.mu.Lock()
			return ctx.Err()
		}
	}

	r.tokens--
	return nil
}

// ProviderError represents an error from the AI provider
type ProviderError struct {
	Code    string
	Message string
	Retries int
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// IsRetryable returns true if the error is retryable
func (e *ProviderError) IsRetryable() bool {
	switch e.Code {
	case "rate_limit_exceeded",
		"server_error",
		"connection_error":
		return true
	default:
		return false
	}
}

// FormatResponse formats the AI response according to the plan
func FormatResponse(content string, err error) string {
	if err != nil {
		return fmt.Sprintf("> Error: %s", err.Error())
	}
	return fmt.Sprintf("> %s", content)
}

// FormatPrompt assembles a prompt with context and tools
func FormatPrompt(systemPrompt, userPrompt string, context map[string]string, tools []string) string {
	var prompt string

	// Add system prompt
	prompt += systemPrompt + "\n\n"

	// Add context if available
	if len(context) > 0 {
		prompt += "Context:\n"
		for section, content := range context {
			prompt += fmt.Sprintf("# %s\n%s\n\n", section, content)
		}
	}

	// Add available tools if any
	if len(tools) > 0 {
		prompt += "Available tools:\n"
		for _, tool := range tools {
			prompt += fmt.Sprintf("- %s\n", tool)
		}
		prompt += "\n"
	}

	// Add user prompt
	prompt += "User: " + userPrompt

	return prompt
}

// ValidatePrompt checks if a prompt is within model limits
func ValidatePrompt(prompt string, limits ModelLimits) error {
	if len(prompt) > limits.MaxPromptSize {
		return fmt.Errorf("prompt size %d exceeds limit of %d", len(prompt), limits.MaxPromptSize)
	}
	return nil
}
