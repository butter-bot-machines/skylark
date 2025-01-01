package provider

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	// Create a rate limiter with 2 tokens per second
	limiter := NewRateLimiter(2, 2, time.Second)

	// Test initial tokens
	ctx := context.Background()
	if err := limiter.Wait(ctx); err != nil {
		t.Errorf("First wait failed: %v", err)
	}
	if err := limiter.Wait(ctx); err != nil {
		t.Errorf("Second wait failed: %v", err)
	}

	// Test waiting for refill
	start := time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Errorf("Third wait failed: %v", err)
	}
	elapsed := time.Since(start)

	// Should have waited close to 1 second
	if elapsed < 900*time.Millisecond {
		t.Errorf("Wait time too short: %v", elapsed)
	}
}

func TestRateLimiterContext(t *testing.T) {
	limiter := NewRateLimiter(1, 1, time.Second)

	// Use up the token
	ctx := context.Background()
	if err := limiter.Wait(ctx); err != nil {
		t.Errorf("First wait failed: %v", err)
	}

	// Test cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got: %v", err)
	}
}

func TestProviderError(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		message    string
		retryable  bool
	}{
		{
			name:      "rate limit error",
			code:      "rate_limit_exceeded",
			message:   "Too many requests",
			retryable: true,
		},
		{
			name:      "server error",
			code:      "server_error",
			message:   "Internal server error",
			retryable: true,
		},
		{
			name:      "validation error",
			code:      "validation_error",
			message:   "Invalid input",
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ProviderError{
				Code:    tt.code,
				Message: tt.message,
			}

			if err.Error() != tt.code+": "+tt.message {
				t.Errorf("Error() = %v, want %v", err.Error(), tt.code+": "+tt.message)
			}

			if err.IsRetryable() != tt.retryable {
				t.Errorf("IsRetryable() = %v, want %v", err.IsRetryable(), tt.retryable)
			}
		})
	}
}

func TestFormatResponse(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		err      error
		expected string
	}{
		{
			name:     "successful response",
			content:  "Hello, world!",
			err:      nil,
			expected: "> Hello, world!",
		},
		{
			name:     "error response",
			content:  "",
			err:      &ProviderError{Code: "error", Message: "Something went wrong"},
			expected: "> Error: error: Something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatResponse(tt.content, tt.err)
			if result != tt.expected {
				t.Errorf("FormatResponse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatPrompt(t *testing.T) {
	tests := []struct {
		name         string
		systemPrompt string
		userPrompt   string
		context      map[string]string
		tools       []string
		wantContains []string
	}{
		{
			name:         "basic prompt",
			systemPrompt: "You are a helpful assistant.",
			userPrompt:   "Hello!",
			context:      nil,
			tools:       nil,
			wantContains: []string{
				"You are a helpful assistant.",
				"User: Hello!",
			},
		},
		{
			name:         "prompt with context",
			systemPrompt: "You are a helpful assistant.",
			userPrompt:   "Summarize this.",
			context: map[string]string{
				"Introduction": "This is a test document.",
			},
			wantContains: []string{
				"Context:",
				"# Introduction",
				"This is a test document.",
			},
		},
		{
			name:         "prompt with tools",
			systemPrompt: "You are a helpful assistant.",
			userPrompt:   "Help me.",
			tools:       []string{"summarize", "translate"},
			wantContains: []string{
				"Available tools:",
				"- summarize",
				"- translate",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPrompt(tt.systemPrompt, tt.userPrompt, tt.context, tt.tools)
			
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatPrompt() missing %q", want)
				}
			}
		})
	}
}

func TestValidatePrompt(t *testing.T) {
	limits := ModelLimits{
		MaxPromptSize: 100,
	}

	tests := []struct {
		name      string
		prompt    string
		wantError bool
	}{
		{
			name:      "valid prompt",
			prompt:    "Hello, world!",
			wantError: false,
		},
		{
			name:      "prompt too long",
			prompt:    strings.Repeat("x", 101),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrompt(tt.prompt, limits)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePrompt() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
