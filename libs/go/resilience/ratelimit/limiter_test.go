package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	t.Run("allows requests within capacity", func(t *testing.T) {
		tb := NewTokenBucket[string](5, 1.0)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			decision, err := tb.Allow(ctx, "user1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !decision.Allowed {
				t.Errorf("request %d should be allowed", i)
			}
		}
	})

	t.Run("denies requests over capacity", func(t *testing.T) {
		tb := NewTokenBucket[string](2, 0.1)
		ctx := context.Background()

		// Exhaust tokens
		tb.Allow(ctx, "user1")
		tb.Allow(ctx, "user1")

		decision, err := tb.Allow(ctx, "user1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if decision.Allowed {
			t.Error("request should be denied")
		}
		if decision.RetryAfter <= 0 {
			t.Error("should have retry-after duration")
		}
	})

	t.Run("refills tokens over time", func(t *testing.T) {
		tb := NewTokenBucket[string](2, 100.0) // 100 tokens/sec
		ctx := context.Background()

		// Exhaust tokens
		tb.Allow(ctx, "user1")
		tb.Allow(ctx, "user1")

		// Wait for refill
		time.Sleep(50 * time.Millisecond)

		decision, _ := tb.Allow(ctx, "user1")
		if !decision.Allowed {
			t.Error("request should be allowed after refill")
		}
	})

	t.Run("separate buckets per key", func(t *testing.T) {
		tb := NewTokenBucket[string](1, 0.1)
		ctx := context.Background()

		d1, _ := tb.Allow(ctx, "user1")
		d2, _ := tb.Allow(ctx, "user2")

		if !d1.Allowed || !d2.Allowed {
			t.Error("different keys should have separate buckets")
		}
	})

	t.Run("Headers returns correct values", func(t *testing.T) {
		tb := NewTokenBucket[string](10, 1.0)
		ctx := context.Background()

		headers, err := tb.Headers(ctx, "user1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if headers.Limit != 10 {
			t.Errorf("expected limit 10, got %d", headers.Limit)
		}
		if headers.Remaining != 10 {
			t.Errorf("expected remaining 10, got %d", headers.Remaining)
		}
	})
}

func TestSlidingWindow(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		sw := NewSlidingWindow[string](5, time.Minute)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			decision, err := sw.Allow(ctx, "user1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !decision.Allowed {
				t.Errorf("request %d should be allowed", i)
			}
		}
	})

	t.Run("denies requests over limit", func(t *testing.T) {
		sw := NewSlidingWindow[string](2, time.Minute)
		ctx := context.Background()

		sw.Allow(ctx, "user1")
		sw.Allow(ctx, "user1")

		decision, err := sw.Allow(ctx, "user1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if decision.Allowed {
			t.Error("request should be denied")
		}
	})

	t.Run("allows requests after window expires", func(t *testing.T) {
		sw := NewSlidingWindow[string](1, 50*time.Millisecond)
		ctx := context.Background()

		sw.Allow(ctx, "user1")

		time.Sleep(60 * time.Millisecond)

		decision, _ := sw.Allow(ctx, "user1")
		if !decision.Allowed {
			t.Error("request should be allowed after window expires")
		}
	})

	t.Run("separate windows per key", func(t *testing.T) {
		sw := NewSlidingWindow[string](1, time.Minute)
		ctx := context.Background()

		d1, _ := sw.Allow(ctx, "user1")
		d2, _ := sw.Allow(ctx, "user2")

		if !d1.Allowed || !d2.Allowed {
			t.Error("different keys should have separate windows")
		}
	})
}

func TestFixedWindow(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		fw := NewFixedWindow[string](5, time.Minute)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			decision, err := fw.Allow(ctx, "user1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !decision.Allowed {
				t.Errorf("request %d should be allowed", i)
			}
		}
	})

	t.Run("denies requests over limit", func(t *testing.T) {
		fw := NewFixedWindow[string](2, time.Minute)
		ctx := context.Background()

		fw.Allow(ctx, "user1")
		fw.Allow(ctx, "user1")

		decision, err := fw.Allow(ctx, "user1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if decision.Allowed {
			t.Error("request should be denied")
		}
	})

	t.Run("resets after window expires", func(t *testing.T) {
		fw := NewFixedWindow[string](1, 50*time.Millisecond)
		ctx := context.Background()

		fw.Allow(ctx, "user1")

		time.Sleep(60 * time.Millisecond)

		decision, _ := fw.Allow(ctx, "user1")
		if !decision.Allowed {
			t.Error("request should be allowed after window resets")
		}
	})
}

func TestRateLimiterWithDifferentKeyTypes(t *testing.T) {
	t.Run("works with int keys", func(t *testing.T) {
		tb := NewTokenBucket[int](5, 1.0)
		ctx := context.Background()

		decision, err := tb.Allow(ctx, 123)
		if err != nil || !decision.Allowed {
			t.Error("should work with int keys")
		}
	})

	t.Run("works with struct keys", func(t *testing.T) {
		type UserKey struct {
			TenantID string
			UserID   int
		}

		tb := NewTokenBucket[UserKey](5, 1.0)
		ctx := context.Background()

		key := UserKey{TenantID: "tenant1", UserID: 123}
		decision, err := tb.Allow(ctx, key)
		if err != nil || !decision.Allowed {
			t.Error("should work with struct keys")
		}
	})
}
