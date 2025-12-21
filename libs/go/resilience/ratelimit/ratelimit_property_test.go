package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 5: Token Bucket Capacity Invariant**
// **Validates: Requirements 1.3**
func TestTokenBucketCapacityInvariant(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("token count never exceeds capacity", prop.ForAll(
		func(capacity int, refillRate int, windowMs int, numRequests int) bool {
			if capacity < 1 {
				capacity = 1
			}
			if capacity > 100 {
				capacity = 100
			}
			if refillRate < 1 {
				refillRate = 1
			}
			if windowMs < 100 {
				windowMs = 100
			}
			if numRequests < 1 {
				numRequests = 1
			}
			if numRequests > 50 {
				numRequests = 50
			}

			tb := NewTokenBucket(TokenBucketConfig{
				Capacity:   capacity,
				RefillRate: refillRate,
				Window:     time.Duration(windowMs) * time.Millisecond,
			})

			ctx := context.Background()
			for i := 0; i < numRequests; i++ {
				tb.Allow(ctx, "test-key")
				tokenCount := tb.GetTokenCount()
				if tokenCount > float64(capacity) {
					return false
				}
			}
			return true
		},
		gen.IntRange(1, 100),
		gen.IntRange(1, 1000),
		gen.IntRange(100, 10000),
		gen.IntRange(1, 50),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 6: Sliding Window Request Count**
// **Validates: Requirements 1.3**
func TestSlidingWindowRequestCount(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("request count never exceeds limit within window", prop.ForAll(
		func(limit int, windowMs int, numRequests int) bool {
			if limit < 1 {
				limit = 1
			}
			if limit > 50 {
				limit = 50
			}
			if windowMs < 100 {
				windowMs = 100
			}
			if numRequests < 1 {
				numRequests = 1
			}
			if numRequests > 100 {
				numRequests = 100
			}

			sw := NewSlidingWindow(SlidingWindowConfig{
				Limit:  limit,
				Window: time.Duration(windowMs) * time.Millisecond,
			})

			ctx := context.Background()
			allowedCount := 0
			for i := 0; i < numRequests; i++ {
				decision, _ := sw.Allow(ctx, "test-key")
				if decision.Allowed {
					allowedCount++
				}
				// Request count should never exceed limit
				requestCount := sw.GetRequestCount()
				if requestCount > limit {
					return false
				}
			}
			// Total allowed should not exceed limit
			return allowedCount <= limit
		},
		gen.IntRange(1, 50),
		gen.IntRange(100, 10000),
		gen.IntRange(1, 100),
	))

	properties.TestingRun(t)
}

func TestTokenBucketAllowDeny(t *testing.T) {
	tb := NewTokenBucket(TokenBucketConfig{
		Capacity:   3,
		RefillRate: 10,
		Window:     time.Second,
	})

	ctx := context.Background()

	// Should allow first 3 requests
	for i := 0; i < 3; i++ {
		decision, err := tb.Allow(ctx, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !decision.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	decision, err := tb.Allow(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Allowed {
		t.Error("4th request should be denied")
	}
}

func TestSlidingWindowAllowDeny(t *testing.T) {
	sw := NewSlidingWindow(SlidingWindowConfig{
		Limit:  3,
		Window: time.Second,
	})

	ctx := context.Background()

	// Should allow first 3 requests
	for i := 0; i < 3; i++ {
		decision, err := sw.Allow(ctx, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !decision.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	decision, err := sw.Allow(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Allowed {
		t.Error("4th request should be denied")
	}
}

func TestRateLimiterFactory(t *testing.T) {
	tests := []struct {
		name      string
		algorithm resilience.RateLimitAlgorithm
		wantErr   bool
	}{
		{"token_bucket", resilience.TokenBucket, false},
		{"sliding_window", resilience.SlidingWindow, false},
		{"unknown", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRateLimiter(resilience.RateLimitConfig{
				Algorithm: tt.algorithm,
				Limit:     100,
				Window:    time.Second,
				BurstSize: 10,
			}, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRateLimiter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
