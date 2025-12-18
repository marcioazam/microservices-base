package benchmark

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/libs/go/resilience/retry"
)

func BenchmarkHandler_CalculateDelay(b *testing.B) {
	h := retry.New(retry.Config{
		ServiceName: "bench-service",
		Config: resilience.RetryConfig{
			MaxAttempts:   5,
			BaseDelay:     100 * time.Millisecond,
			MaxDelay:      10 * time.Second,
			Multiplier:    2.0,
			JitterPercent: 0.1,
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.CalculateDelay(i % 5)
	}
}

func BenchmarkHandler_Execute_ImmediateSuccess(b *testing.B) {
	h := retry.New(retry.Config{
		ServiceName: "bench-service",
		Config: resilience.RetryConfig{
			MaxAttempts:   3,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Second,
			Multiplier:    2.0,
			JitterPercent: 0.1,
		},
	})

	ctx := context.Background()
	successOp := func() error { return nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Execute(ctx, successOp)
	}
}

func BenchmarkHandler_Execute_ImmediateFailure(b *testing.B) {
	h := retry.New(retry.Config{
		ServiceName: "bench-service",
		Config: resilience.RetryConfig{
			MaxAttempts:   1, // Only one attempt
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Second,
			Multiplier:    2.0,
			JitterPercent: 0.1,
		},
	})

	ctx := context.Background()
	failOp := func() error { return errors.New("fail") }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Execute(ctx, failOp)
	}
}

func BenchmarkHandler_Execute_Parallel(b *testing.B) {
	h := retry.New(retry.Config{
		ServiceName: "bench-service",
		Config: resilience.RetryConfig{
			MaxAttempts:   3,
			BaseDelay:     time.Millisecond,
			MaxDelay:      time.Second,
			Multiplier:    2.0,
			JitterPercent: 0.1,
		},
	})

	ctx := context.Background()
	successOp := func() error { return nil }

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = h.Execute(ctx, successOp)
		}
	})
}

func BenchmarkPolicyParsing_Parse(b *testing.B) {
	policyStr := `{
		"max_attempts": 5,
		"base_delay_ms": 100,
		"max_delay_ms": 10000,
		"multiplier": 2.0,
		"jitter_percent": 0.1
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = retry.ParsePolicy([]byte(policyStr))
	}
}

func BenchmarkPolicyParsing_Marshal(b *testing.B) {
	policy := &resilience.RetryConfig{
		MaxAttempts:   5,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		Multiplier:    2.0,
		JitterPercent: 0.1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = retry.MarshalPolicy(policy)
	}
}

func BenchmarkPolicyParsing_RoundTrip(b *testing.B) {
	policy := &resilience.RetryConfig{
		MaxAttempts:   5,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		Multiplier:    2.0,
		JitterPercent: 0.1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := retry.MarshalPolicy(policy)
		_, _ = retry.ParsePolicy(data)
	}
}
