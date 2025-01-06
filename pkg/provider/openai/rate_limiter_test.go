package openai

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucketLimiter(t *testing.T) {
	// Create limiter with low limits for testing
	limiter := &TokenBucketLimiter{
		config: RateLimitConfig{
			RequestsPerMinute: 2,
			TokensPerMinute:   100,
		},
		requestTokens: 2,
		tokenTokens:   100,
		lastReset:     time.Now(),
	}

	// Test request limiting
	t.Run("request limits", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // Ensure goroutine cleanup

		// First request should succeed
		if err := limiter.Wait(ctx); err != nil {
			t.Errorf("first request failed: %v", err)
		}

		// Second request should succeed
		if err := limiter.Wait(ctx); err != nil {
			t.Errorf("second request failed: %v", err)
		}

		// Third request should block
		done := make(chan struct{}, 1) // Buffered to avoid goroutine leak
		cleanup := make(chan struct{})

		go func() {
			defer close(cleanup)
			limiter.Wait(ctx) // Will be cancelled by deferred cancel()
			done <- struct{}{}
		}()

		// Should not complete within 100ms
		timer := time.NewTimer(100 * time.Millisecond)
		defer timer.Stop()

		select {
		case <-done:
			t.Error("third request should have blocked")
		case <-timer.C:
			// Expected behavior - request blocked
			cancel() // Cancel context to clean up goroutine
		}

		// Wait for goroutine cleanup
		<-cleanup
	})

	// Test token limiting
	t.Run("token limits", func(t *testing.T) {
		// Reset limiter
		limiter = &TokenBucketLimiter{
			config: RateLimitConfig{
				RequestsPerMinute: 10,
				TokensPerMinute:   100,
			},
			requestTokens: 10,
			tokenTokens:   100,
			lastReset:     time.Now(),
		}

		// First batch of tokens should succeed
		if err := limiter.AddTokens(50); err != nil {
			t.Errorf("first token batch failed: %v", err)
		}

		// Second batch should succeed
		if err := limiter.AddTokens(40); err != nil {
			t.Errorf("second token batch failed: %v", err)
		}

		// Third batch should fail (over limit)
		if err := limiter.AddTokens(20); err == nil {
			t.Error("expected token limit error")
		}
	})

	// Test context cancellation
	t.Run("context cancellation", func(t *testing.T) {
		// Reset limiter
		limiter = &TokenBucketLimiter{
			config: RateLimitConfig{
				RequestsPerMinute: 1,
				TokensPerMinute:   100,
			},
			requestTokens: 1,
			tokenTokens:   100,
			lastReset:     time.Now(),
		}

		// Use up the request
		ctx := context.Background()
		if err := limiter.Wait(ctx); err != nil {
			t.Errorf("first request failed: %v", err)
		}

		// Try with cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if err := limiter.Wait(ctx); err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	// Test reset behavior
	t.Run("minute reset", func(t *testing.T) {
		// Create limiter that's already used up
		limiter = &TokenBucketLimiter{
			config: RateLimitConfig{
				RequestsPerMinute: 1,
				TokensPerMinute:   50,
			},
			requestTokens: 0,
			tokenTokens:   0,
			lastReset:     time.Now().Add(-61 * time.Second),
		}

		// Should succeed after reset
		ctx := context.Background()
		if err := limiter.Wait(ctx); err != nil {
			t.Errorf("request after reset failed: %v", err)
		}
		if err := limiter.AddTokens(50); err != nil {
			t.Errorf("tokens after reset failed: %v", err)
		}
	})
}
