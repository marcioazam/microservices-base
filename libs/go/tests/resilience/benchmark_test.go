package resilience_test

import (
	"context"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/resilience"
)

func BenchmarkCircuitBreaker_Execute_Closed(b *testing.B) {
	config := resilience.NewCircuitBreakerConfig("bench",
		resilience.WithFailureThreshold(5),
		resilience.WithCircuitTimeout(time.Second),
	)
	cb, _ := resilience.NewCircuitBreaker(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkCircuitBreaker_Execute_Open(b *testing.B) {
	config := resilience.NewCircuitBreakerConfig("bench",
		resilience.WithFailureThreshold(1),
		resilience.WithCircuitTimeout(time.Hour),
	)
	cb, _ := resilience.NewCircuitBreaker(config)
	ctx := context.Background()

	// Open the circuit
	cb.Execute(ctx, func(ctx context.Context) error {
		return resilience.NewTimeoutError("", "", 0, 0, nil)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkRateLimiter_Allow(b *testing.B) {
	config := resilience.RateLimitConfig{
		Rate:      1000000,
		Window:    time.Second,
		BurstSize: 1000000,
	}
	limiter, _ := resilience.NewRateLimiter(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}

func BenchmarkBulkhead_Execute(b *testing.B) {
	config := resilience.BulkheadConfig{
		MaxConcurrent: 100,
		QueueSize:     1000,
		MaxWait:       time.Second,
	}
	bulkhead, _ := resilience.NewBulkhead(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bulkhead.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkRetry_Success(b *testing.B) {
	config := resilience.NewRetryConfig(
		resilience.WithMaxAttempts(3),
		resilience.WithInitialInterval(time.Millisecond),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resilience.Retry(ctx, config, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	config := resilience.DefaultCircuitBreakerConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.Validate()
	}
}

func BenchmarkErrorCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resilience.NewCircuitOpenError("service", "corr-id", time.Now(), time.Second, 0.5)
	}
}

func BenchmarkErrorTypeCheck(b *testing.B) {
	err := resilience.NewCircuitOpenError("service", "corr-id", time.Now(), time.Second, 0.5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resilience.IsCircuitOpen(err)
	}
}
