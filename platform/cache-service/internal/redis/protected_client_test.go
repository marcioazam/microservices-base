package redis

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/authcorp/libs/go/src/fault"
	"pgregory.net/rapid"
)

// mockClient implements a minimal Redis client for testing.
type mockClient struct {
	failCount  int32
	shouldFail bool
}

func (m *mockClient) Get(ctx context.Context, key string) ([]byte, error) {
	if m.shouldFail {
		atomic.AddInt32(&m.failCount, 1)
		return nil, errors.New("redis error")
	}
	return []byte("value"), nil
}

func (m *mockClient) Set(ctx context.Context, key string, value []byte, ttl int64) error {
	if m.shouldFail {
		atomic.AddInt32(&m.failCount, 1)
		return errors.New("redis error")
	}
	return nil
}

// TestCircuitBreakerThresholdProperty validates that circuit breaker opens after threshold failures.
// Property 3: Circuit Breaker Threshold
// Validates: Requirements 1.3, 5.1
func TestCircuitBreakerThresholdProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate threshold between 1 and 10
		threshold := rapid.IntRange(1, 10).Draw(t, "threshold")

		// Create circuit breaker with generated threshold
		cfg := fault.NewCircuitBreakerConfig("test",
			fault.WithFailureThreshold(threshold),
			fault.WithSuccessThreshold(1),
		)
		cb, err := fault.NewCircuitBreaker(cfg)
		if err != nil {
			t.Fatalf("failed to create circuit breaker: %v", err)
		}

		ctx := context.Background()
		failingOp := func(ctx context.Context) error {
			return errors.New("simulated failure")
		}

		// Execute failing operations up to threshold
		for i := 0; i < threshold; i++ {
			_ = cb.Execute(ctx, failingOp)
		}

		// Property: After threshold failures, circuit should be open
		state := cb.State()
		if state != fault.StateOpen {
			t.Errorf("expected circuit to be open after %d failures, got state: %s", threshold, state)
		}

		// Property: When open, operations should fail with CircuitOpenError
		err = cb.Execute(ctx, func(ctx context.Context) error {
			return nil // Would succeed if allowed
		})
		if err == nil {
			t.Error("expected error when circuit is open")
		}
		_, isCircuitOpen := err.(*fault.CircuitOpenError)
		if !isCircuitOpen {
			t.Errorf("expected CircuitOpenError, got: %T", err)
		}
	})
}

// TestCircuitBreakerRecoveryProperty validates circuit breaker recovery behavior.
func TestCircuitBreakerRecoveryProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		successThreshold := rapid.IntRange(1, 5).Draw(t, "successThreshold")

		cfg := fault.NewCircuitBreakerConfig("test",
			fault.WithFailureThreshold(1),
			fault.WithSuccessThreshold(successThreshold),
		)
		cb, err := fault.NewCircuitBreaker(cfg)
		if err != nil {
			t.Fatalf("failed to create circuit breaker: %v", err)
		}

		ctx := context.Background()

		// Trip the circuit
		_ = cb.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})

		// Reset to simulate timeout passing (half-open state)
		cb.Reset()

		// Execute successful operations
		successCount := 0
		for i := 0; i < successThreshold; i++ {
			err := cb.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
			if err == nil {
				successCount++
			}
		}

		// Property: After success threshold, circuit should be closed
		state := cb.State()
		if state != fault.StateClosed {
			t.Errorf("expected circuit to be closed after %d successes, got state: %s", successThreshold, state)
		}
	})
}

// TestCircuitBreakerStateTransitionsProperty validates state transitions.
func TestCircuitBreakerStateTransitionsProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		failureThreshold := rapid.IntRange(2, 5).Draw(t, "failureThreshold")
		failureCount := rapid.IntRange(0, failureThreshold+2).Draw(t, "failureCount")

		cfg := fault.NewCircuitBreakerConfig("test",
			fault.WithFailureThreshold(failureThreshold),
		)
		cb, err := fault.NewCircuitBreaker(cfg)
		if err != nil {
			t.Fatalf("failed to create circuit breaker: %v", err)
		}

		ctx := context.Background()

		// Execute failures
		for i := 0; i < failureCount; i++ {
			_ = cb.Execute(ctx, func(ctx context.Context) error {
				return errors.New("failure")
			})
		}

		state := cb.State()

		// Property: State depends on failure count vs threshold
		if failureCount < failureThreshold {
			if state != fault.StateClosed {
				t.Errorf("expected closed state with %d failures (threshold: %d), got: %s",
					failureCount, failureThreshold, state)
			}
		} else {
			if state != fault.StateOpen {
				t.Errorf("expected open state with %d failures (threshold: %d), got: %s",
					failureCount, failureThreshold, state)
			}
		}
	})
}
