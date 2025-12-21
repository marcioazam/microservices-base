// Package benchmark provides performance benchmarks for the resilience executor.
package benchmark

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/resilience"
	"github.com/authcorp/libs/go/src/fault"
)

// mockMetricsRecorder is a no-op metrics recorder for benchmarks.
type mockMetricsRecorder struct {
	executions atomic.Int64
}

func (m *mockMetricsRecorder) RecordExecution(ctx context.Context, metrics fault.ExecutionMetrics) {
	m.executions.Add(1)
}

func (m *mockMetricsRecorder) RecordCircuitState(ctx context.Context, policyName string, state string) {}
func (m *mockMetricsRecorder) RecordRetryAttempt(ctx context.Context, policyName string, attempt int) {}
func (m *mockMetricsRecorder) RecordRateLimit(ctx context.Context, policyName string, limited bool) {}
func (m *mockMetricsRecorder) RecordBulkheadQueue(ctx context.Context, policyName string, queued bool) {}
func (m *mockMetricsRecorder) RecordCacheStats(ctx context.Context, hits, misses, evictions int64) {}

// BenchmarkFailsafeExecutorRegisterPolicy benchmarks policy registration.
func BenchmarkFailsafeExecutorRegisterPolicy(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	metrics := &mockMetricsRecorder{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		executor := resilience.NewFailsafeExecutor(metrics, logger)

		policy, _ := entities.NewPolicy("test-policy")
		cbResult := entities.NewCircuitBreakerConfig(5, 3, 30*time.Second, 2)
		if cbResult.IsOk() {
			policy.SetCircuitBreaker(cbResult.Unwrap())
		}

		_ = executor.RegisterPolicy(policy)
	}
}

// BenchmarkFailsafeExecutorExecute benchmarks execution with circuit breaker.
func BenchmarkFailsafeExecutorExecute(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	metrics := &mockMetricsRecorder{}
	executor := resilience.NewFailsafeExecutor(metrics, logger)

	policy, _ := entities.NewPolicy("test-policy")
	cbResult := entities.NewCircuitBreakerConfig(5, 3, 30*time.Second, 2)
	if cbResult.IsOk() {
		policy.SetCircuitBreaker(cbResult.Unwrap())
	}
	_ = executor.RegisterPolicy(policy)

	ctx := context.Background()
	operation := func() error { return nil }

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = executor.Execute(ctx, "test-policy", operation)
	}
}

// BenchmarkFailsafeExecutorExecuteWithRetry benchmarks execution with retry.
func BenchmarkFailsafeExecutorExecuteWithRetry(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	metrics := &mockMetricsRecorder{}
	executor := resilience.NewFailsafeExecutor(metrics, logger)

	policy, _ := entities.NewPolicy("test-policy")
	retryResult := entities.NewRetryConfig(3, 1*time.Millisecond, 10*time.Millisecond, 2.0, 0.1)
	if retryResult.IsOk() {
		policy.SetRetry(retryResult.Unwrap())
	}
	_ = executor.RegisterPolicy(policy)

	ctx := context.Background()
	operation := func() error { return nil }

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = executor.Execute(ctx, "test-policy", operation)
	}
}

// BenchmarkFailsafeExecutorExecuteWithTimeout benchmarks execution with timeout.
func BenchmarkFailsafeExecutorExecuteWithTimeout(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	metrics := &mockMetricsRecorder{}
	executor := resilience.NewFailsafeExecutor(metrics, logger)

	policy, _ := entities.NewPolicy("test-policy")
	timeoutResult := entities.NewTimeoutConfig(5*time.Second, 30*time.Second)
	if timeoutResult.IsOk() {
		policy.SetTimeout(timeoutResult.Unwrap())
	}
	_ = executor.RegisterPolicy(policy)

	ctx := context.Background()
	operation := func() error { return nil }

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = executor.Execute(ctx, "test-policy", operation)
	}
}

// BenchmarkFailsafeExecutorExecuteParallel benchmarks parallel execution.
func BenchmarkFailsafeExecutorExecuteParallel(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	metrics := &mockMetricsRecorder{}
	executor := resilience.NewFailsafeExecutor(metrics, logger)

	policy, _ := entities.NewPolicy("test-policy")
	cbResult := entities.NewCircuitBreakerConfig(5, 3, 30*time.Second, 2)
	if cbResult.IsOk() {
		policy.SetCircuitBreaker(cbResult.Unwrap())
	}
	_ = executor.RegisterPolicy(policy)

	ctx := context.Background()
	operation := func() error { return nil }

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = executor.Execute(ctx, "test-policy", operation)
		}
	})
}

// BenchmarkFailsafeExecutorExecuteAllPatterns benchmarks execution with all patterns.
func BenchmarkFailsafeExecutorExecuteAllPatterns(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	metrics := &mockMetricsRecorder{}
	executor := resilience.NewFailsafeExecutor(metrics, logger)

	policy, _ := entities.NewPolicy("test-policy")

	cbResult := entities.NewCircuitBreakerConfig(5, 3, 30*time.Second, 2)
	if cbResult.IsOk() {
		policy.SetCircuitBreaker(cbResult.Unwrap())
	}

	retryResult := entities.NewRetryConfig(3, 1*time.Millisecond, 10*time.Millisecond, 2.0, 0.1)
	if retryResult.IsOk() {
		policy.SetRetry(retryResult.Unwrap())
	}

	timeoutResult := entities.NewTimeoutConfig(5*time.Second, 30*time.Second)
	if timeoutResult.IsOk() {
		policy.SetTimeout(timeoutResult.Unwrap())
	}

	_ = executor.RegisterPolicy(policy)

	ctx := context.Background()
	operation := func() error { return nil }

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = executor.Execute(ctx, "test-policy", operation)
	}
}
