package property

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience/ratelimit"
	"pgregory.net/rapid"
)

// **Feature: resilience-service-state-of-art-2025, Property 11: Rate Limit Enforcement**
// **Validates: Requirements 4.1**
func TestProperty_RateLimitEnforcement(t *testing.T) {
	t.Run("requests_beyond_limit_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			limit := rapid.IntRange(1, 20).Draw(t, "limit")

			tb := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
				Capacity:   limit,
				RefillRate: limit,
				Window:     time.Minute,
			})

			ctx := context.Background()

			// Make exactly limit requests - all should succeed
			for i := 0; i < limit; i++ {
				decision, _ := tb.Allow(ctx, "test-key")
				if !decision.Allowed {
					t.Fatalf("request %d should be allowed", i)
				}
			}

			// Next request should be rejected
			decision, _ := tb.Allow(ctx, "test-key")
			if decision.Allowed {
				t.Fatal("request beyond limit should be rejected")
			}

			// Should have positive retry-after
			if decision.RetryAfter <= 0 {
				t.Fatal("retry-after should be positive")
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 12: Token Bucket Invariants**
// **Validates: Requirements 4.2**
func TestProperty_TokenBucketInvariants(t *testing.T) {
	t.Run("tokens_never_exceed_capacity", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			capacity := rapid.IntRange(5, 50).Draw(t, "capacity")
			requests := rapid.IntRange(1, 100).Draw(t, "requests")

			tb := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
				Capacity:   capacity,
				RefillRate: capacity * 10, // Fast refill
				Window:     time.Second,
			})

			ctx := context.Background()

			// Make some requests
			for i := 0; i < requests; i++ {
				tb.Allow(ctx, "test-key")
			}

			// Wait for refill
			time.Sleep(200 * time.Millisecond)

			// Tokens should never exceed capacity
			tokens := tb.GetTokenCount()
			if tokens > tb.GetCapacity() {
				t.Fatalf("tokens %f exceeded capacity %f", tokens, tb.GetCapacity())
			}
		})
	})

	t.Run("consumption_decreases_tokens", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			capacity := rapid.IntRange(5, 50).Draw(t, "capacity")

			tb := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
				Capacity:   capacity,
				RefillRate: 1, // Very slow refill
				Window:     time.Hour,
			})

			ctx := context.Background()

			initialTokens := tb.GetTokenCount()

			// Consume one token
			decision, _ := tb.Allow(ctx, "test-key")
			if !decision.Allowed {
				t.Fatal("first request should be allowed")
			}

			finalTokens := tb.GetTokenCount()

			// Should have decreased by approximately 1
			diff := initialTokens - finalTokens
			if diff < 0.99 || diff > 1.01 {
				t.Fatalf("token decrease %f not approximately 1", diff)
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 13: Sliding Window Request Counting**
// **Validates: Requirements 4.3**
func TestProperty_SlidingWindowRequestCounting(t *testing.T) {
	t.Run("only_counts_requests_within_window", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			limit := rapid.IntRange(5, 20).Draw(t, "limit")
			window := 100 * time.Millisecond

			sw := ratelimit.NewSlidingWindow(ratelimit.SlidingWindowConfig{
				Limit:  limit,
				Window: window,
			})

			ctx := context.Background()

			// Make some requests
			requestsMade := minInt(limit, 5)
			for i := 0; i < requestsMade; i++ {
				sw.Allow(ctx, "test-key")
			}

			// Count should equal requests made
			if sw.GetRequestCount() != requestsMade {
				t.Fatalf("request count %d != %d", sw.GetRequestCount(), requestsMade)
			}

			// Wait for window to expire
			time.Sleep(window + 50*time.Millisecond)

			// Count should be 0 after window expires
			if sw.GetRequestCount() != 0 {
				t.Fatalf("request count %d != 0 after window", sw.GetRequestCount())
			}
		})
	})

	t.Run("requests_within_window_counted", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			limit := rapid.IntRange(10, 50).Draw(t, "limit")
			requests := rapid.IntRange(1, 20).Draw(t, "requests")

			sw := ratelimit.NewSlidingWindow(ratelimit.SlidingWindowConfig{
				Limit:  limit,
				Window: time.Hour, // Long window
			})

			ctx := context.Background()

			// Make requests
			actualRequests := minInt(requests, limit)
			for i := 0; i < actualRequests; i++ {
				sw.Allow(ctx, "test-key")
			}

			// All requests should be counted
			if sw.GetRequestCount() != actualRequests {
				t.Fatalf("request count %d != %d", sw.GetRequestCount(), actualRequests)
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 14: Rate Limit Response Headers**
// **Validates: Requirements 4.5**
func TestProperty_RateLimitResponseHeaders(t *testing.T) {
	t.Run("headers_always_present", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			capacity := rapid.IntRange(5, 50).Draw(t, "capacity")
			requests := rapid.IntRange(0, 30).Draw(t, "requests")

			tb := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
				Capacity:   capacity,
				RefillRate: capacity,
				Window:     time.Minute,
			})

			ctx := context.Background()

			// Make some requests
			for i := 0; i < requests; i++ {
				tb.Allow(ctx, "test-key")
			}

			// Get headers
			headers, err := tb.GetHeaders(ctx, "test-key")
			if err != nil {
				t.Fatalf("GetHeaders error: %v", err)
			}

			// All headers must be present and valid
			if headers.Limit <= 0 {
				t.Fatal("limit should be positive")
			}
			if headers.Remaining < 0 {
				t.Fatal("remaining should be non-negative")
			}
			if headers.Reset <= 0 {
				t.Fatal("reset should be positive")
			}

			// Limit should equal capacity
			if headers.Limit != capacity {
				t.Fatalf("limit %d != capacity %d", headers.Limit, capacity)
			}
		})
	})

	t.Run("remaining_decreases_with_requests", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			capacity := rapid.IntRange(5, 50).Draw(t, "capacity")

			tb := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
				Capacity:   capacity,
				RefillRate: 1, // Very slow refill
				Window:     time.Hour,
			})

			ctx := context.Background()

			headers1, _ := tb.GetHeaders(ctx, "test-key")
			initialRemaining := headers1.Remaining

			// Make a request
			tb.Allow(ctx, "test-key")

			headers2, _ := tb.GetHeaders(ctx, "test-key")
			finalRemaining := headers2.Remaining

			// Remaining should have decreased
			if finalRemaining >= initialRemaining {
				t.Fatalf("remaining %d should be less than %d", finalRemaining, initialRemaining)
			}
		})
	})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
