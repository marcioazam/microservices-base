package benchmark

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/libs/go/resilience/circuitbreaker"
)

func BenchmarkCircuitBreaker_RecordSuccess(b *testing.B) {
	cb := circuitbreaker.New(circuitbreaker.Config{
		ServiceName: "bench-service",
		Config: resilience.CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          time.Second,
		},
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.RecordSuccess()
		}
	})
}

func BenchmarkCircuitBreaker_RecordFailure(b *testing.B) {
	cb := circuitbreaker.New(circuitbreaker.Config{
		ServiceName: "bench-service",
		Config: resilience.CircuitBreakerConfig{
			FailureThreshold: 1000000, // High threshold to avoid state changes
			SuccessThreshold: 2,
			Timeout:          time.Second,
		},
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.RecordFailure()
		}
	})
}

func BenchmarkCircuitBreaker_Execute_Success(b *testing.B) {
	cb := circuitbreaker.New(circuitbreaker.Config{
		ServiceName: "bench-service",
		Config: resilience.CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          time.Second,
		},
	})

	ctx := context.Background()
	successOp := func() error { return nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(ctx, successOp)
	}
}

func BenchmarkCircuitBreaker_Execute_Failure(b *testing.B) {
	cb := circuitbreaker.New(circuitbreaker.Config{
		ServiceName: "bench-service",
		Config: resilience.CircuitBreakerConfig{
			FailureThreshold: 1000000,
			SuccessThreshold: 2,
			Timeout:          time.Second,
		},
	})

	ctx := context.Background()
	failOp := func() error { return errors.New("fail") }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(ctx, failOp)
	}
}

func BenchmarkCircuitBreaker_GetState(b *testing.B) {
	cb := circuitbreaker.New(circuitbreaker.Config{
		ServiceName: "bench-service",
		Config: resilience.CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          time.Second,
		},
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.GetState()
		}
	})
}

func BenchmarkStateSerialization_Marshal(b *testing.B) {
	now := time.Now()
	state := resilience.CircuitBreakerState{
		ServiceName:     "bench-service",
		State:           resilience.StateOpen,
		FailureCount:    5,
		SuccessCount:    0,
		LastFailureTime: &now,
		LastStateChange: now,
		Version:         1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = circuitbreaker.MarshalState(state)
	}
}

func BenchmarkStateSerialization_Unmarshal(b *testing.B) {
	now := time.Now()
	state := resilience.CircuitBreakerState{
		ServiceName:     "bench-service",
		State:           resilience.StateOpen,
		FailureCount:    5,
		SuccessCount:    0,
		LastFailureTime: &now,
		LastStateChange: now,
		Version:         1,
	}

	data, _ := circuitbreaker.MarshalState(state)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = circuitbreaker.UnmarshalState(data)
	}
}

func BenchmarkStateSerialization_RoundTrip(b *testing.B) {
	now := time.Now()
	state := resilience.CircuitBreakerState{
		ServiceName:     "bench-service",
		State:           resilience.StateOpen,
		FailureCount:    5,
		SuccessCount:    0,
		LastFailureTime: &now,
		LastStateChange: now,
		Version:         1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := circuitbreaker.MarshalState(state)
		_, _ = circuitbreaker.UnmarshalState(data)
	}
}
