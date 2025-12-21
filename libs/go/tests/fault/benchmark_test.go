package fault_test

import (
	"context"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/fault"
)

func BenchmarkCircuitBreaker_Execute_Closed(b *testing.B) {
	config := fault.NewCircuitBreakerConfig("bench",
		fault.WithFailureThreshold(5),
		fault.WithCircuitTimeout(time.Second),
	)
	cb, _ := fault.NewCircuitBreaker(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkCircuitBreaker_Execute_Open(b *testing.B) {
	config := fault.NewCircuitBreakerConfig("bench",
		fault.WithFailureThreshold(1),
		fault.WithCircuitTimeout(time.Hour),
	)
	cb, _ := fault.NewCircuitBreaker(config)
	ctx := context.Background()

	// Open the circuit
	cb.Execute(ctx, func(ctx context.Context) error {
		return fault.NewTimeoutError("", "", 0, 0, nil)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkRateLimiter_Allow(b *testing.B) {
	config := fault.RateLimitConfig{
		Rate:      1000000,
		Window:    time.Second,
		BurstSize: 1000000,
	}
	limiter, _ := fault.NewRateLimiter(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}

func BenchmarkBulkhead_Execute(b *testing.B) {
	config := fault.BulkheadConfig{
		MaxConcurrent: 100,
		QueueSize:     1000,
		MaxWait:       time.Second,
	}
	bulkhead, _ := fault.NewBulkhead(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bulkhead.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkRetry_Success(b *testing.B) {
	config := fault.NewRetryConfig(
		fault.WithMaxAttempts(3),
		fault.WithInitialInterval(time.Millisecond),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fault.Retry(ctx, config, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	config := fault.DefaultCircuitBreakerConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.Validate()
	}
}

func BenchmarkErrorCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fault.NewCircuitOpenError("service", "corr-id", time.Now(), time.Second, 0.5)
	}
}

func BenchmarkErrorTypeCheck(b *testing.B) {
	err := fault.NewCircuitOpenError("service", "corr-id", time.Now(), time.Second, 0.5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fault.IsCircuitOpen(err)
	}
}
