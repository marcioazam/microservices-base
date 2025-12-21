package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience/ratelimit"
)

func BenchmarkTokenBucket_Allow(b *testing.B) {
	tb := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
		Capacity:   1000000,
		RefillRate: 1000000,
	})

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = tb.Allow(ctx, "bench-key")
		}
	})
}

func BenchmarkTokenBucket_Allow_Limited(b *testing.B) {
	tb := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
		Capacity:   100,
		RefillRate: 10,
	})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tb.Allow(ctx, "bench-key")
	}
}

func BenchmarkTokenBucket_GetHeaders(b *testing.B) {
	tb := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
		Capacity:   1000,
		RefillRate: 100,
	})

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = tb.GetHeaders(ctx, "bench-key")
		}
	})
}

func BenchmarkSlidingWindow_Allow(b *testing.B) {
	sw := ratelimit.NewSlidingWindow(ratelimit.SlidingWindowConfig{
		Limit:  1000000,
		Window: time.Hour,
	})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sw.Allow(ctx, "bench-key")
	}
}

func BenchmarkSlidingWindow_Allow_Limited(b *testing.B) {
	sw := ratelimit.NewSlidingWindow(ratelimit.SlidingWindowConfig{
		Limit:  100,
		Window: time.Second,
	})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sw.Allow(ctx, "bench-key")
	}
}

func BenchmarkSlidingWindow_GetHeaders(b *testing.B) {
	sw := ratelimit.NewSlidingWindow(ratelimit.SlidingWindowConfig{
		Limit:  1000,
		Window: time.Minute,
	})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sw.GetHeaders(ctx, "bench-key")
	}
}

func BenchmarkSlidingWindow_Concurrent(b *testing.B) {
	sw := ratelimit.NewSlidingWindow(ratelimit.SlidingWindowConfig{
		Limit:  1000000,
		Window: time.Hour,
	})

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = sw.Allow(ctx, "bench-key")
		}
	})
}

func BenchmarkTokenBucket_Refill(b *testing.B) {
	tb := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
		Capacity:   100,
		RefillRate: 1000,
	})

	ctx := context.Background()

	// Drain tokens
	for i := 0; i < 100; i++ {
		tb.Allow(ctx, "bench-key")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Wait a bit for refill
		time.Sleep(time.Microsecond)
		_, _ = tb.Allow(ctx, "bench-key")
	}
}
