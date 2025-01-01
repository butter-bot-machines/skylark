package openai

import (
	"context"
	"testing"
	"time"

	"github.com/butter-bot-machines/skylark/pkg/config"
)

func TestOpenAIProvider(t *testing.T) {
	cfg := config.ModelConfig{
		Provider: "openai",
		Name:     "gpt-4",
		Parameters: map[string]interface{}{
			"max_tokens":   2048,
			"temperature": 0.7,
		},
		MaxTokens:        2048,
		Temperature:      0.7,
		TopP:            1.0,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
	}

	provider, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Test successful response
	t.Run("successful response", func(t *testing.T) {
		ctx := context.Background()
		response, err := provider.Send(ctx, "Test prompt")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if response == nil {
			t.Error("Expected response, got nil")
		}
	})

	// Test rate limiting
	t.Run("rate limiting", func(t *testing.T) {
		ctx := context.Background()
		// Send requests until we hit rate limit
		for i := 0; i < 10; i++ {
			_, err := provider.Send(ctx, "Test prompt")
			if err != nil {
				if err.Error() != "rate limit exceeded" {
					t.Errorf("Expected rate limit error, got %v", err)
				}
				return
			}
			time.Sleep(10 * time.Millisecond) // Small delay between requests
		}
		t.Error("Expected rate limit error")
	})

	// Test context cancellation
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := provider.Send(ctx, "Test prompt")
		if err == nil {
			t.Error("Expected error from cancelled context")
		}
	})

	// Test context timeout
	t.Run("context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		time.Sleep(2 * time.Millisecond) // Force timeout
		_, err := provider.Send(ctx, "Test prompt")
		if err == nil {
			t.Error("Expected error from timeout")
		}
	})
}
