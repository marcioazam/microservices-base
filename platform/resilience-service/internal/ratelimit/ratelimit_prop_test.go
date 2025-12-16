package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/testutil"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-microservice, Property 11: Rate Limit Enforcement**
// **Validates: Requirements 4.1**
func TestProperty_RateLimitEnforcement(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("requests_beyond_limit_rejected", prop.ForAll(
		func(limit int) bool {
			tb := NewTokenBucket(TokenBucketConfig{
				Capacity:   limit,
				RefillRate: limit,
				Window:     time.Minute,
			})

			ctx := context.Background()

			// Make exactly limit requests - all should succeed
			for i := 0; i < limit; i++ {
				decision, _ := tb.Allow(ctx, "test-key")
				if !decision.Allowed {
					return false
				}
			}

			// Next request should be rejected
			decision, _ := tb.Allow(ctx, "test-key")
			if decision.Allowed {
				return false
			}

			// Should have positive retry-after
			return decision.RetryAfter > 0
		},
		gen.IntRange(1, 20),
	))

	props.TestingRun(t)
}

// **Feature: resilience-microservice, Property 12: Token Bucket Invariants**
// **Validates: Requirements 4.2**
func TestProperty_TokenBucketInvariants(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("tokens_never_exceed_capacity", prop.ForAll(
		func(capacity int, requests int) bool {
			tb := NewTokenBucket(TokenBucketConfig{
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
			return tokens <= tb.GetCapacity()
		},
		gen.IntRange(5, 50),
		gen.IntRange(1, 100),
	))

	props.Property("consumption_decreases_tokens", prop.ForAll(
		func(capacity int) bool {
			tb := NewTokenBucket(TokenBucketConfig{
				Capacity:   capacity,
				RefillRate: 1, // Very slow refill
				Window:     time.Hour,
			})

			ctx := context.Background()

			initialTokens := tb.GetTokenCount()

			// Consume one token
			decision, _ := tb.Allow(ctx, "test-key")
			if !decision.Allowed {
				return false
			}

			finalTokens := tb.GetTokenCount()

			// Should have decreased by approximately 1
			return initialTokens-finalTokens >= 0.99 && initialTokens-finalTokens <= 1.01
		},
		gen.IntRange(5, 50),
	))

	props.TestingRun(t)
}

// **Feature: resilience-microservice, Property 13: Sliding Window Request Counting**
// **Validates: Requirements 4.3**
func TestProperty_SlidingWindowRequestCounting(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("only_counts_requests_within_window", prop.ForAll(
		func(limit int) bool {
			window := 100 * time.Millisecond

			sw := NewSlidingWindow(SlidingWindowConfig{
				Limit:  limit,
				Window: window,
			})

			ctx := context.Background()

			// Make some requests
			requestsMade := min(limit, 5)
			for i := 0; i < requestsMade; i++ {
				sw.Allow(ctx, "test-key")
			}

			// Count should equal requests made
			if sw.GetRequestCount() != requestsMade {
				return false
			}

			// Wait for window to expire
			time.Sleep(window + 50*time.Millisecond)

			// Count should be 0 after window expires
			return sw.GetRequestCount() == 0
		},
		gen.IntRange(5, 20),
	))

	props.Property("requests_within_window_counted", prop.ForAll(
		func(limit int, requests int) bool {
			sw := NewSlidingWindow(SlidingWindowConfig{
				Limit:  limit,
				Window: time.Hour, // Long window
			})

			ctx := context.Background()

			// Make requests
			actualRequests := min(requests, limit)
			for i := 0; i < actualRequests; i++ {
				sw.Allow(ctx, "test-key")
			}

			// All requests should be counted
			return sw.GetRequestCount() == actualRequests
		},
		gen.IntRange(10, 50),
		gen.IntRange(1, 20),
	))

	props.TestingRun(t)
}

// **Feature: resilience-microservice, Property 14: Rate Limit Response Headers**
// **Validates: Requirements 4.5**
func TestProperty_RateLimitResponseHeaders(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("headers_always_present", prop.ForAll(
		func(capacity int, requests int) bool {
			tb := NewTokenBucket(TokenBucketConfig{
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
				return false
			}

			// All headers must be present and valid
			if headers.Limit <= 0 {
				return false
			}
			if headers.Remaining < 0 {
				return false
			}
			if headers.Reset <= 0 {
				return false
			}

			// Limit should equal capacity
			if headers.Limit != capacity {
				return false
			}

			return true
		},
		gen.IntRange(5, 50),
		gen.IntRange(0, 30),
	))

	props.Property("remaining_decreases_with_requests", prop.ForAll(
		func(capacity int) bool {
			tb := NewTokenBucket(TokenBucketConfig{
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
			return finalRemaining < initialRemaining
		},
		gen.IntRange(5, 50),
	))

	props.TestingRun(t)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
