package ratelimit

import (
	"fmt"

	"github.com/auth-platform/libs/go/resilience"
)

// NewRateLimiter creates a rate limiter based on the algorithm configuration.
func NewRateLimiter(cfg resilience.RateLimitConfig, emitter resilience.EventEmitter) (resilience.RateLimiter, error) {
	switch cfg.Algorithm {
	case resilience.TokenBucket:
		return NewTokenBucket(TokenBucketConfig{
			Capacity:     cfg.BurstSize,
			RefillRate:   cfg.Limit,
			Window:       cfg.Window,
			EventEmitter: emitter,
		}), nil
	case resilience.SlidingWindow:
		return NewSlidingWindow(SlidingWindowConfig{
			Limit:        cfg.Limit,
			Window:       cfg.Window,
			EventEmitter: emitter,
		}), nil
	default:
		return nil, fmt.Errorf("unknown rate limit algorithm: %s", cfg.Algorithm)
	}
}
