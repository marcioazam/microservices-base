// Package ratelimit implements rate limiting algorithms.
package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/auth-platform/libs/go/resilience"
)

// TokenBucket implements the token bucket rate limiting algorithm.
type TokenBucket struct {
	mu           sync.Mutex
	capacity     float64
	tokens       float64
	refillRate   float64 // tokens per second
	lastRefill   time.Time
	eventEmitter resilience.EventEmitter
}

// TokenBucketConfig holds token bucket configuration.
type TokenBucketConfig struct {
	Capacity     int           // Maximum tokens (burst size)
	RefillRate   int           // Tokens per window
	Window       time.Duration // Time window for refill rate
	EventEmitter resilience.EventEmitter
}

// NewTokenBucket creates a new token bucket rate limiter.
func NewTokenBucket(cfg TokenBucketConfig) *TokenBucket {
	// Calculate tokens per second
	refillRate := float64(cfg.RefillRate) / cfg.Window.Seconds()

	return &TokenBucket{
		capacity:     float64(cfg.Capacity),
		tokens:       float64(cfg.Capacity), // Start full
		refillRate:   refillRate,
		lastRefill:   time.Now(),
		eventEmitter: cfg.EventEmitter,
	}
}

// Allow checks if a request should be allowed.
func (tb *TokenBucket) Allow(ctx context.Context, key string) (resilience.RateLimitDecision, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	decision := resilience.RateLimitDecision{
		Limit:     int(tb.capacity),
		Remaining: int(tb.tokens),
		ResetAt:   tb.calculateResetTime(),
	}

	if tb.tokens >= 1 {
		tb.tokens--
		decision.Allowed = true
		decision.Remaining = int(tb.tokens)
	} else {
		decision.Allowed = false
		decision.RetryAfter = tb.calculateRetryAfter()
		tb.emitRateLimitEvent(key, decision)
	}

	return decision, nil
}

// GetHeaders returns rate limit headers for response.
func (tb *TokenBucket) GetHeaders(ctx context.Context, key string) (resilience.RateLimitHeaders, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	return resilience.RateLimitHeaders{
		Limit:     int(tb.capacity),
		Remaining: int(tb.tokens),
		Reset:     tb.calculateResetTime().Unix(),
	}, nil
}

// refill adds tokens based on elapsed time. Must be called with lock held.
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()

	// Add tokens based on elapsed time
	tb.tokens += elapsed * tb.refillRate

	// Cap at capacity
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	tb.lastRefill = now
}

// calculateResetTime returns when the bucket will be full.
func (tb *TokenBucket) calculateResetTime() time.Time {
	if tb.tokens >= tb.capacity {
		return time.Now()
	}

	tokensNeeded := tb.capacity - tb.tokens
	secondsToFull := tokensNeeded / tb.refillRate

	return time.Now().Add(time.Duration(secondsToFull * float64(time.Second)))
}

// calculateRetryAfter returns how long to wait for a token.
func (tb *TokenBucket) calculateRetryAfter() time.Duration {
	if tb.tokens >= 1 {
		return 0
	}

	tokensNeeded := 1 - tb.tokens
	secondsToWait := tokensNeeded / tb.refillRate

	return time.Duration(secondsToWait * float64(time.Second))
}

// emitRateLimitEvent emits a rate limit event.
func (tb *TokenBucket) emitRateLimitEvent(key string, decision resilience.RateLimitDecision) {
	event := resilience.Event{
		ID:        resilience.GenerateEventID(),
		Type:      resilience.EventRateLimitHit,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"key":         key,
			"allowed":     decision.Allowed,
			"remaining":   decision.Remaining,
			"retry_after": decision.RetryAfter.String(),
		},
	}

	resilience.EmitEvent(tb.eventEmitter, event)
}

// GetTokenCount returns current token count (for testing).
func (tb *TokenBucket) GetTokenCount() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

// GetCapacity returns the bucket capacity.
func (tb *TokenBucket) GetCapacity() float64 {
	return tb.capacity
}
