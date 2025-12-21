package property

import (
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/libs/go/resilience/ratelimit"
	"pgregory.net/rapid"
)

// **Feature: platform-resilience-modernization, Property 3: Rate Limiter Factory Correctness**
// **Validates: Requirements 5.2, 5.3**
func TestProperty_RateLimiterFactory(t *testing.T) {
	t.Run("token_bucket_algorithm_returns_token_bucket", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			limit := rapid.IntRange(1, 1000).Draw(t, "limit")
			windowMs := rapid.IntRange(1000, 60000).Draw(t, "windowMs")
			burstSize := rapid.IntRange(1, 100).Draw(t, "burstSize")

			cfg := resilience.RateLimitConfig{
				Algorithm: resilience.TokenBucket,
				Limit:     limit,
				Window:    time.Duration(windowMs) * time.Millisecond,
				BurstSize: burstSize,
			}

			limiter, err := ratelimit.NewRateLimiter(cfg, nil)
			if err != nil {
				t.Fatalf("NewRateLimiter error: %v", err)
			}

			_, ok := limiter.(*ratelimit.TokenBucket)
			if !ok {
				t.Fatal("expected TokenBucketLimiter")
			}
		})
	})

	t.Run("sliding_window_algorithm_returns_sliding_window", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			limit := rapid.IntRange(1, 1000).Draw(t, "limit")
			windowMs := rapid.IntRange(1000, 60000).Draw(t, "windowMs")

			cfg := resilience.RateLimitConfig{
				Algorithm: resilience.SlidingWindow,
				Limit:     limit,
				Window:    time.Duration(windowMs) * time.Millisecond,
			}

			limiter, err := ratelimit.NewRateLimiter(cfg, nil)
			if err != nil {
				t.Fatalf("NewRateLimiter error: %v", err)
			}

			_, ok := limiter.(*ratelimit.SlidingWindow)
			if !ok {
				t.Fatal("expected SlidingWindowLimiter")
			}
		})
	})

	t.Run("unknown_algorithm_returns_error", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			algorithm := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "algorithm")

			// Skip if it matches known algorithms
			if algorithm == string(resilience.TokenBucket) || algorithm == string(resilience.SlidingWindow) {
				return
			}

			cfg := resilience.RateLimitConfig{
				Algorithm: resilience.RateLimitAlgorithm(algorithm),
				Limit:     100,
				Window:    time.Second,
			}

			_, err := ratelimit.NewRateLimiter(cfg, nil)
			if err == nil {
				t.Fatal("expected error for unknown algorithm")
			}
		})
	})
}
