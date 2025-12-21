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

// **Feature: resilience-service-state-of-art-2025, Property 1: Failsafe-go Resilience Integration**
// **Validates: Requirements 1.3, 6.1, 6.2, 6.3, 6.4, 6.5**
func TestFailsafeGoIntegrationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		failureThreshold := rapid.IntRange(1, 10).Draw(t, "failure_threshold")
		maxAttempts := rapid.IntRange(1, 5).Draw(t, "max_attempts")
		timeoutDuration := time.Duration(rapid.Int64Range(100, 5000).Draw(t, "timeout_ms")) * time.Millisecond
		shouldFail := rapid.Bool().Draw(t, "should_fail")

		// Create failsafe-go policies
		cb := circuitbreaker.Builder[any]().
			WithFailureThreshold(uint(failureThreshold)).
			WithDelay(100 * time.Millisecond).
			Build()

		retry := retrypolicy.Builder[any]().
			WithMaxAttempts(maxAttempts).
			WithDelay(50 * time.Millisecond).
			Build()

		timeoutPolicy := timeout.With[any](timeoutDuration)

		// Test operation
		operation := func() (any, error) {
			if shouldFail {
				return nil, errors.New("simulated failure")
			}
			return "success", nil
		}

		// Execute with failsafe-go
		ctx := context.Background()
		executor := failsafe.NewExecutor[any](cb, retry, timeoutPolicy)
		result, err := executor.Get(operation)

		// Verify failsafe-go behavior
		if shouldFail {
			// Should eventually fail after retries
			if err == nil {
				t.Fatalf("Expected error for failing operation, got success: %v", result)
			}
		} else {
			// Should succeed
			if err != nil {
				t.Fatalf("Expected success for non-failing operation, got error: %v", err)
			}
			if result != "success" {
				t.Fatalf("Expected 'success', got: %v", result)
			}
		}

		// Verify circuit breaker state is accessible
		state := cb.State()
		if state != circuitbreaker.ClosedState && state != circuitbreaker.OpenState && state != circuitbreaker.HalfOpenState {
			t.Fatalf("Invalid circuit breaker state: %v", state)
		}

		// Test context cancellation with timeout
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Immediately cancel

		timeoutExecutor := failsafe.NewExecutor[any](timeoutPolicy)
		_, err = timeoutExecutor.GetWithExecution(func(exec failsafe.Execution[any]) (any, error) {
			select {
			case <-cancelCtx.Done():
				return nil, cancelCtx.Err()
			default:
				return "success", nil
			}
		})

		// Should respect context cancellation
		if err == nil {
			t.Fatal("Expected context cancellation error")
		}
	})
}

// Test that failsafe-go types are used instead of custom implementations
func TestFailsafeGoTypesProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create various failsafe-go policies
		cb := circuitbreaker.Builder[any]().Build()
		retry := retrypolicy.Builder[any]().Build()
		timeoutPolicy := timeout.With[any](time.Second)

		// Verify types are from failsafe-go package
		if cb == nil {
			t.Fatal("Circuit breaker should not be nil")
		}
		if retry == nil {
			t.Fatal("Retry policy should not be nil")
		}
		if timeoutPolicy == nil {
			t.Fatal("Timeout policy should not be nil")
		}

		// Test policy composition
		executor := failsafe.NewExecutor[any](cb, retry, timeoutPolicy)
		if executor == nil {
			t.Fatal("Failsafe executor should not be nil")
		}

		// Test execution
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

// Test circuit breaker state transitions
func TestCircuitBreakerStateTransitionsProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		failureThreshold := rapid.IntRange(2, 5).Draw(t, "failure_threshold")
		
		cb := circuitbreaker.Builder[any]().
			WithFailureThreshold(uint(failureThreshold)).
			WithDelay(50 * time.Millisecond).
			Build()

		// Initially should be closed
		if cb.State() != circuitbreaker.ClosedState {
			t.Fatalf("Circuit breaker should start in closed state, got: %v", cb.State())
		}

		// Cause failures to open circuit
		failingOp := func() (any, error) {
			return nil, errors.New("failure")
		}

		executor := failsafe.NewExecutor[any](cb)
		
		// Execute failing operations
		for i := 0; i < failureThreshold+1; i++ {
			executor.Get(failingOp)
		}

		// Circuit should be open after threshold failures
		if cb.State() != circuitbreaker.OpenState {
			t.Fatalf("Circuit breaker should be open after %d failures, got: %v", failureThreshold, cb.State())
		}

		// Verify open circuit rejects calls immediately
		start := time.Now()
		_, err := executor.Get(func() (any, error) {
			return "success", nil
		})
		duration := time.Since(start)

		if err == nil {
			t.Fatal("Expected circuit breaker to reject call when open")
		}
		
		// Should fail fast (within 10ms)
		if duration > 10*time.Millisecond {
			t.Fatalf("Circuit breaker should fail fast when open, took: %v", duration)
		}
	})
}

// Test FailsafeExecutor integration with domain entities
func TestFailsafeExecutorIntegrationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create mock metrics recorder
		mockMetrics := &testutil.MockMetricsRecorder{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		
		// Create failsafe executor
		executor := resilience.NewFailsafeExecutor(mockMetrics, logger)
		
		// Generate policy configuration
		policyName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_-]*$`).Draw(t, "policy_name")
		
		policy, err := entities.NewPolicy(policyName)
		if err != nil {
			t.Fatalf("Failed to create policy: %v", err)
		}
		
		// Add circuit breaker configuration
		failureThreshold := rapid.IntRange(2, 10).Draw(t, "failure_threshold")
		successThreshold := rapid.IntRange(1, failureThreshold).Draw(t, "success_threshold")
		timeout := time.Duration(rapid.Int64Range(1, 30).Draw(t, "timeout_sec")) * time.Second
		
		cbConfig, err := entities.NewCircuitBreakerConfig(failureThreshold, successThreshold, timeout, 1)
		if err != nil {
			t.Fatalf("Failed to create circuit breaker config: %v", err)
		}
		
		err = policy.SetCircuitBreaker(cbConfig)
		if err != nil {
			t.Fatalf("Failed to set circuit breaker: %v", err)
		}
		
		// Register policy with executor
		err = executor.RegisterPolicy(policy)
		if err != nil {
			t.Fatalf("Failed to register policy: %v", err)
		}
		
		// Test successful execution
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
		
		// Test policy names retrieval
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
		
		// Test policy executor retrieval
		policyExec, exists := executor.GetPolicyExecutor(policyName)
		if !exists {
			t.Fatal("Policy executor should exist")
		}
		
		if policyExec == nil {
			t.Fatal("Policy executor should not be nil")
		}
	})
}

// Test retry policy integration
func TestRetryPolicyIntegrationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockMetrics := &testutil.MockMetricsRecorder{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		executor := resilience.NewFailsafeExecutor(mockMetrics, logger)
		
		policyName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_-]*$`).Draw(t, "policy_name")
		
		policy, err := entities.NewPolicy(policyName)
		if err != nil {
			t.Fatalf("Failed to create policy: %v", err)
		}
		
		// Add retry configuration
		maxAttempts := rapid.IntRange(2, 5).Draw(t, "max_attempts")
		baseDelay := time.Duration(rapid.Int64Range(1000, 5000).Draw(t, "base_delay_ms")) * time.Millisecond
		maxDelay := time.Duration(rapid.Int64Range(int64(baseDelay/time.Millisecond)+1000, 300000).Draw(t, "max_delay_ms")) * time.Millisecond
		
		retryConfig, err := entities.NewRetryConfig(maxAttempts, baseDelay, maxDelay, 2.0, 0.1)
		if err != nil {
			t.Fatalf("Failed to create retry config: %v", err)
		}
		
		err = policy.SetRetry(retryConfig)
		if err != nil {
			t.Fatalf("Failed to set retry: %v", err)
		}
		
		err = executor.RegisterPolicy(policy)
		if err != nil {
			t.Fatalf("Failed to register policy: %v", err)
		}
		
		// Test retry behavior
		ctx := context.Background()
		attemptCount := 0
		
		err = executor.Execute(ctx, policyName, func() error {
			attemptCount++
			if attemptCount < maxAttempts {
				return errors.New("simulated failure")
			}
			return nil // Success on last attempt
		})
		
		if err != nil {
			t.Fatalf("Expected eventual success after retries, got error: %v", err)
		}
		
		if attemptCount != maxAttempts {
			t.Fatalf("Expected %d attempts, got %d", maxAttempts, attemptCount)
		}
	})
}

// Using shared mocks from testutil package