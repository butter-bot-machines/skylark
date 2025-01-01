package openai

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimitConfig holds rate limit settings
type RateLimitConfig struct {
	RequestsPerMinute int // Max requests per minute
	TokensPerMinute   int // Max tokens per minute
}

// TokenBucketLimiter implements RateLimiting using token bucket algorithm
type TokenBucketLimiter struct {
	config RateLimitConfig

	requestTokens int
	tokenTokens   int
	lastReset    time.Time
	mu           sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) RateLimiting {
	return &TokenBucketLimiter{
		config:       config,
		requestTokens: config.RequestsPerMinute,
		tokenTokens:   config.TokensPerMinute,
		lastReset:    time.Now(),
	}
}

// Wait blocks until a request can be made
func (r *TokenBucketLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Reset tokens if minute has passed
	if time.Since(r.lastReset) > time.Minute {
		r.requestTokens = r.config.RequestsPerMinute
		r.tokenTokens = r.config.TokensPerMinute
		r.lastReset = time.Now()
	}

	// Check request limit
	if r.requestTokens <= 0 {
		// Calculate wait time
		waitTime := time.Minute - time.Since(r.lastReset)
		
		// Wait with context
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Reset tokens after waiting
			r.requestTokens = r.config.RequestsPerMinute
			r.tokenTokens = r.config.TokensPerMinute
			r.lastReset = time.Now()
		}
	}

	// Use request token
	r.requestTokens--
	return nil
}

// AddTokens records token usage and checks limits
func (r *TokenBucketLimiter) AddTokens(count int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Reset if minute has passed
	if time.Since(r.lastReset) > time.Minute {
		r.requestTokens = r.config.RequestsPerMinute
		r.tokenTokens = r.config.TokensPerMinute
		r.lastReset = time.Now()
	}

	// Check token limit
	if r.tokenTokens < count {
		return fmt.Errorf("token limit exceeded: used %d/%d this minute", 
			r.config.TokensPerMinute-r.tokenTokens+count, 
			r.config.TokensPerMinute)
	}

	// Use tokens
	r.tokenTokens -= count
	return nil
}
