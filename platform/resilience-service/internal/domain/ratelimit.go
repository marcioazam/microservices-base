package domain

import (
	"context"
	"time"
)

// RateLimitAlgorithm defines the rate limiting algorithm.
type RateLimitAlgorithm string

const (
	TokenBucket   RateLimitAlgorithm = "token_bucket"
	SlidingWindow RateLimitAlgorithm = "sliding_window"
)

// RateLimitConfig defines rate limiting behavior.
type RateLimitConfig struct {
	Algorithm RateLimitAlgorithm `json:"algorithm" yaml:"algorithm"`
	Limit     int                `json:"limit" yaml:"limit"`
	Window    time.Duration      `json:"window" yaml:"window"`
	BurstSize int                `json:"burst_size" yaml:"burstSize"`
}

// RateLimitDecision represents allow/deny decision.
type RateLimitDecision struct {
	Allowed    bool
	Remaining  int
	Limit      int
	ResetAt    time.Time
	RetryAfter time.Duration
}

// RateLimitHeaders contains rate limit response headers.
type RateLimitHeaders struct {
	Limit     int   `json:"X-RateLimit-Limit"`
	Remaining int   `json:"X-RateLimit-Remaining"`
	Reset     int64 `json:"X-RateLimit-Reset"`
}

// RateLimiter controls request throughput.
type RateLimiter interface {
	// Allow checks if request should be allowed.
	Allow(ctx context.Context, key string) (RateLimitDecision, error)

	// GetHeaders returns rate limit headers for response.
	GetHeaders(ctx context.Context, key string) (RateLimitHeaders, error)
}

// RateLimitEvent represents a rate limit hit for observability.
type RateLimitEvent struct {
	Key           string        `json:"key"`
	Allowed       bool          `json:"allowed"`
	Remaining     int           `json:"remaining"`
	RetryAfter    time.Duration `json:"retry_after,omitempty"`
	CorrelationID string        `json:"correlation_id"`
	Timestamp     time.Time     `json:"timestamp"`
}
