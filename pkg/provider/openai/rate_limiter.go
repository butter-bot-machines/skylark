package openai

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimitConfig defines rate limiting parameters
type RateLimitConfig struct {
	RequestsPerMinute int
	TokensPerMinute   int
}

// RateLimiter implements rate limiting for API requests
type RateLimiter struct {
	mu            sync.Mutex
	config        RateLimitConfig
	requestCount  int
	tokenCount    int
	lastReset     time.Time
	requestTokens chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	r := &RateLimiter{
		config:        config,
		lastReset:     time.Now(),
		requestTokens: make(chan struct{}, config.RequestsPerMinute),
	}

	// Initialize request tokens
	for i := 0; i < config.RequestsPerMinute; i++ {
		r.requestTokens <- struct{}{}
	}

	return r
}

// Wait waits for rate limit clearance
func (r *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.requestTokens:
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("rate limit exceeded")
	}
}

// AddTokens adds token usage and checks limits
func (r *RateLimiter) AddTokens(count int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if now.Sub(r.lastReset) >= time.Minute {
		r.tokenCount = 0
		r.lastReset = now
	}

	r.tokenCount += count
	if r.tokenCount > r.config.TokensPerMinute {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}
