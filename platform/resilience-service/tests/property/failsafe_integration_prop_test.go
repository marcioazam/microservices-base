package property

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/resilience"
	"github.com/auth-platform/platform/resilience-service/tests/testutil"
	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
	"github.com/failsafe-go/failsafe-go/timeout"
	"pgregory.net/rapid"
)

// TestFailsafeGoIntegrationProperty validates failsafe-go resilience integration.
func TestFailsafeGoIntegrationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		failureThreshold := rapid.IntRange(1, 10).Draw(t, "failure_threshold")
		maxAttempts := rapid.IntRange(1, 5).Draw(t, "max_attempts")
		timeoutDuration := time.Duration(rapid.Int64Range(100, 5000).Draw(t, "timeout_ms")) * time.Millisecond
		shouldFail := rapid.Bool().Draw(t, "should_fail")

		cb := circuitbreaker.Builder[any]().
			WithFailureThreshold(uint(failureThreshold)).
			WithDelay(100 * time.Millisecond).
			Build()

		retry := retrypolicy.Builder[any]().
			WithMaxAttempts(maxAttempts).
			WithDelay(50 * time.Millisecond).
			Build()

		timeoutPolicy := timeout.With[any](timeoutDuration)

		operation := func() (any, error) {
			if shouldFail {
				return nil, errors.New("simulated failure")
			}
			return "success", nil
		}

		ctx := context.Background()
		executor := failsafe.NewExecutor[any](cb, retry, timeoutPolicy)
		result, err := executor.Get(operation)

		if shouldFail {
			if err == nil {
				t.Fatalf("Expected error for failing operation, got success: %v", result)
			}
		} else {
			if err != nil {
				t.Fatalf("Expected success for non-failing operation, got error: %v", err)
			}
			if result != "success" {
				t.Fatalf("Expected 'success', got: %v", result)
			}
		}

		state := cb.State()
		validStates := state == circuitbreaker.ClosedState || state == circuitbreaker.OpenState || state == circuitbreaker.HalfOpenState
		if !validStates {
			t.Fatalf("Invalid circuit breaker state: %v", state)
		}

		cancelCtx, cancel := context.WithCancel(ctx)
		cancel()

		timeoutExecutor := failsafe.NewExecutor[any](timeoutPolicy)
		_, err = timeoutExecutor.GetWithExecution(func(exec failsafe.Execution[any]) (any, error) {
			select {
			case <-cancelCtx.Done():
				return nil, cancelCtx.Err()
			default:
				return "success", nil
			}
		})

		if err == nil {
			t.Fatal("Expected context cancellation error")
		}
	})
}

// TestFailsafeGoTypesProperty tests that failsafe-go types are used.
func TestFailsafeGoTypesProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cb := circuitbreaker.Builder[any]().Build()
		retry := retrypolicy.Builder[any]().Build()
		timeoutPolicy := timeout.With[any](time.Second)

		if cb == nil {
			t.Fatal("Circuit breaker should not be nil")
		}
		if retry == nil {
			t.Fatal("Retry policy should not be nil")
		}
		if timeoutPolicy == nil {
			t.Fatal("Timeout policy should not be nil")
		}

		executor := failsafe.NewExecutor[any](cb, retry, timeoutPolicy)
		if executor == nil {
			t.Fatal("Failsafe executor should not be nil")
		}

		result, err := executor.Get(func() (any, error) {
			return "test", nil
		})

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result != "test" {
			t.Fatalf("Expected 'test', got: %v", result)
		}
	})
}

// TestCircuitBreakerStateTransitionsProperty tests circuit breaker state transitions.
func TestCircuitBreakerStateTransitionsProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		failureThreshold := rapid.IntRange(2, 5).Draw(t, "failure_threshold")

		cb := circuitbreaker.Builder[any]().
			WithFailureThreshold(uint(failureThreshold)).
			WithDelay(50 * time.Millisecond).
			Build()

		if cb.State() != circuitbreaker.ClosedState {
			t.Fatalf("Circuit breaker should start in closed state, got: %v", cb.State())
		}

		failingOp := func() (any, error) {
			return nil, errors.New("failure")
		}

		executor := failsafe.NewExecutor[any](cb)

		for i := 0; i < failureThreshold+1; i++ {
			executor.Get(failingOp)
		}

		if cb.State() != circuitbreaker.OpenState {
			t.Fatalf("Circuit breaker should be open after %d failures, got: %v", failureThreshold, cb.State())
		}

		start := time.Now()
		_, err := executor.Get(func() (any, error) {
			return "success", nil
		})
		duration := time.Since(start)

		if err == nil {
			t.Fatal("Expected circuit breaker to reject call when open")
		}

		if duration > 10*time.Millisecond {
			t.Fatalf("Circuit breaker should fail fast when open, took: %v", duration)
		}
	})
}

// TestFailsafeExecutorIntegrationProperty tests FailsafeExecutor integration.
func TestFailsafeExecutorIntegrationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockMetrics := &testutil.MockMetricsRecorder{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		executor := resilience.NewFailsafeExecutor(mockMetrics, logger)

		policyName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_-]{2,20}$`).Draw(t, "policy_name")

		policy, err := entities.NewPolicy(policyName)
		if err != nil {
			t.Fatalf("Failed to create policy: %v", err)
		}

		failureThreshold := rapid.IntRange(2, 10).Draw(t, "failure_threshold")
		successThreshold := rapid.IntRange(1, failureThreshold).Draw(t, "success_threshold")
		cbTimeout := time.Duration(rapid.Int64Range(1, 30).Draw(t, "timeout_sec")) * time.Second

		cbResult := entities.NewCircuitBreakerConfig(failureThreshold, successThreshold, cbTimeout, 1)
		if cbResult.IsErr() {
			t.Fatalf("Failed to create circuit breaker config: %v", cbResult.UnwrapErr())
		}

		cbConfig := cbResult.Unwrap()
		setResult := policy.SetCircuitBreaker(cbConfig)
		if setResult.IsErr() {
			t.Fatalf("Failed to set circuit breaker: %v", setResult.UnwrapErr())
		}

		err = executor.RegisterPolicy(policy)
		if err != nil {
			t.Fatalf("Failed to register policy: %v", err)
		}

		ctx := context.Background()
		successCount := 0

		err = executor.Execute(ctx, policyName, func() error {
			successCount++
			return nil
		})

		if err != nil {
			t.Fatalf("Expected successful execution, got error: %v", err)
		}

		if successCount != 1 {
			t.Fatalf("Expected operation to be called once, got %d", successCount)
		}

		names := executor.GetPolicyNames()
		found := false
		for _, name := range names {
			if name == policyName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Policy %s should be in registered policies list", policyName)
		}

		policyExec, exists := executor.GetPolicyExecutor(policyName)
		if !exists {
			t.Fatal("Policy executor should exist")
		}

		if policyExec == nil {
			t.Fatal("Policy executor should not be nil")
		}
	})
}


// TestRetryPolicyIntegrationProperty tests retry policy integration with fast delays.
func TestRetryPolicyIntegrationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockMetrics := &testutil.MockMetricsRecorder{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		executor := resilience.NewFailsafeExecutor(mockMetrics, logger)

		policyName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_-]{2,20}$`).Draw(t, "policy_name")

		policy, err := entities.NewPolicy(policyName)
		if err != nil {
			t.Fatalf("Failed to create policy: %v", err)
		}

		// Use small delays for fast tests (1-50ms base, 1-5s max)
		maxAttempts := rapid.IntRange(2, 3).Draw(t, "max_attempts")
		baseDelay := time.Duration(rapid.Int64Range(1, 50).Draw(t, "base_delay_ms")) * time.Millisecond
		maxDelay := time.Duration(rapid.Int64Range(1000, 5000).Draw(t, "max_delay_ms")) * time.Millisecond

		retryResult := entities.NewRetryConfig(maxAttempts, baseDelay, maxDelay, 2.0, 0.1)
		if retryResult.IsErr() {
			t.Fatalf("Failed to create retry config: %v", retryResult.UnwrapErr())
		}

		retryConfig := retryResult.Unwrap()
		setResult := policy.SetRetry(retryConfig)
		if setResult.IsErr() {
			t.Fatalf("Failed to set retry: %v", setResult.UnwrapErr())
		}

		err = executor.RegisterPolicy(policy)
		if err != nil {
			t.Fatalf("Failed to register policy: %v", err)
		}

		ctx := context.Background()
		attemptCount := 0

		err = executor.Execute(ctx, policyName, func() error {
			attemptCount++
			if attemptCount < maxAttempts {
				return errors.New("simulated failure")
			}
			return nil
		})

		if err != nil {
			t.Fatalf("Expected eventual success after retries, got error: %v", err)
		}

		if attemptCount != maxAttempts {
			t.Fatalf("Expected %d attempts, got %d", maxAttempts, attemptCount)
		}
	})
}
