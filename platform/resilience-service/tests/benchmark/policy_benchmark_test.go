// Package benchmark provides performance benchmarks for the resilience service.
package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
)

// BenchmarkPolicyCreation benchmarks policy entity creation.
func BenchmarkPolicyCreation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = entities.NewPolicy("test-policy")
	}
}

// BenchmarkCircuitBreakerConfigCreation benchmarks circuit breaker config creation.
func BenchmarkCircuitBreakerConfigCreation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = entities.NewCircuitBreakerConfig(5, 3, 30*time.Second, 2)
	}
}

// BenchmarkRetryConfigCreation benchmarks retry config creation.
func BenchmarkRetryConfigCreation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = entities.NewRetryConfig(3, 100*time.Millisecond, 10*time.Second, 2.0, 0.1)
	}
}

// BenchmarkPolicyWithAllPatterns benchmarks policy with all resilience patterns.
func BenchmarkPolicyWithAllPatterns(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		policy, _ := entities.NewPolicy("test-policy")

		cbResult := entities.NewCircuitBreakerConfig(5, 3, 30*time.Second, 2)
		if cbResult.IsOk() {
			policy.SetCircuitBreaker(cbResult.Unwrap())
		}

		retryResult := entities.NewRetryConfig(3, 100*time.Millisecond, 10*time.Second, 2.0, 0.1)
		if retryResult.IsOk() {
			policy.SetRetry(retryResult.Unwrap())
		}

		timeoutResult := entities.NewTimeoutConfig(5*time.Second, 30*time.Second)
		if timeoutResult.IsOk() {
			policy.SetTimeout(timeoutResult.Unwrap())
		}

		rateLimitResult := entities.NewRateLimitConfig("token_bucket", 1000, time.Minute, 100)
		if rateLimitResult.IsOk() {
			policy.SetRateLimit(rateLimitResult.Unwrap())
		}

		bulkheadResult := entities.NewBulkheadConfig(100, 50, 5*time.Second)
		if bulkheadResult.IsOk() {
			policy.SetBulkhead(bulkheadResult.Unwrap())
		}
	}
}

// BenchmarkPolicyClone benchmarks policy cloning.
func BenchmarkPolicyClone(b *testing.B) {
	policy, _ := entities.NewPolicy("test-policy")
	cbResult := entities.NewCircuitBreakerConfig(5, 3, 30*time.Second, 2)
	if cbResult.IsOk() {
		policy.SetCircuitBreaker(cbResult.Unwrap())
	}
	retryResult := entities.NewRetryConfig(3, 100*time.Millisecond, 10*time.Second, 2.0, 0.1)
	if retryResult.IsOk() {
		policy.SetRetry(retryResult.Unwrap())
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = policy.Clone()
	}
}

// BenchmarkPolicyValidation benchmarks policy validation.
func BenchmarkPolicyValidation(b *testing.B) {
	policy, _ := entities.NewPolicy("test-policy")
	cbResult := entities.NewCircuitBreakerConfig(5, 3, 30*time.Second, 2)
	if cbResult.IsOk() {
		policy.SetCircuitBreaker(cbResult.Unwrap())
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = policy.ValidateResult()
	}
}

// BenchmarkCircuitBreakerConfigValidation benchmarks config validation.
func BenchmarkCircuitBreakerConfigValidation(b *testing.B) {
	config := &entities.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ProbeCount:       2,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = config.ValidateResult()
	}
}

// BenchmarkPolicyCreationParallel benchmarks parallel policy creation.
func BenchmarkPolicyCreationParallel(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = entities.NewPolicy("test-policy")
		}
	})
}

// BenchmarkContextPropagation benchmarks context propagation overhead.
func BenchmarkContextPropagation(b *testing.B) {
	baseCtx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, cancel := context.WithTimeout(baseCtx, 5*time.Second)
		cancel()
	}
}
