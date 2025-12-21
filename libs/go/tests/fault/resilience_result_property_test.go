// Feature: go-libs-state-of-art-2025, Property 6: Resilience Result Integration
// Validates: Requirements 7.3, 7.4, 7.5
package fault_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/fault"
	"pgregory.net/rapid"
)

// TestCancelledContextReturnsErr verifies cancelled context returns Err.
func TestCancelledContextReturnsErr(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		config := fault.RetryConfig{
			MaxAttempts:     3,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
			RetryIf:         func(err error) bool { return true },
		}

		result := fault.RetryWithResult(ctx, config, func(ctx context.Context) (int, error) {
			return 0, errors.New("should not reach here")
		})

		if result.IsOk() {
			t.Fatal("cancelled context should return Err")
		}

		if !errors.Is(result.UnwrapErr(), context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", result.UnwrapErr())
		}
	})
}

// TestTimeoutContextReturnsErr verifies timeout returns Err with DeadlineExceeded.
func TestTimeoutContextReturnsErr(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Wait for timeout
		time.Sleep(5 * time.Millisecond)

		config := fault.RetryConfig{
			MaxAttempts:     3,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
			RetryIf:         func(err error) bool { return true },
		}

		result := fault.RetryWithResult(ctx, config, func(ctx context.Context) (int, error) {
			return 0, errors.New("operation error")
		})

		if result.IsOk() {
			t.Fatal("timed out context should return Err")
		}

		if !errors.Is(result.UnwrapErr(), context.DeadlineExceeded) {
			t.Fatalf("expected context.DeadlineExceeded, got %v", result.UnwrapErr())
		}
	})
}

// TestCircuitOpenReturnsErr verifies open circuit returns Err with CircuitOpenError.
func TestCircuitOpenReturnsErr(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		config := fault.CircuitBreakerConfig{
			Name:             "test-circuit",
			FailureThreshold: 2,
			SuccessThreshold: 1,
			Timeout:          1 * time.Second,
			HalfOpenRequests: 1,
		}

		cb, err := fault.NewCircuitBreaker(config)
		if err != nil {
			t.Fatalf("failed to create circuit breaker: %v", err)
		}

		ctx := context.Background()

		// Trip the circuit by causing failures
		for i := 0; i < config.FailureThreshold; i++ {
			cb.Execute(ctx, func(ctx context.Context) error {
				return errors.New("failure")
			})
		}

		// Circuit should be open now
		if cb.State() != fault.StateOpen {
			t.Fatalf("circuit should be open, got %s", cb.State())
		}

		// Execute should return CircuitOpenError
		result := fault.ExecuteWithResult(cb, ctx, func(ctx context.Context) (int, error) {
			return 42, nil
		})

		if result.IsOk() {
			t.Fatal("open circuit should return Err")
		}

		var circuitErr *fault.CircuitOpenError
		if !errors.As(result.UnwrapErr(), &circuitErr) {
			t.Fatalf("expected CircuitOpenError, got %T: %v", result.UnwrapErr(), result.UnwrapErr())
		}
	})
}

// TestRetryExhaustedReturnsErr verifies exhausted retries return Err.
func TestRetryExhaustedReturnsErr(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxAttempts := rapid.IntRange(1, 5).Draw(t, "maxAttempts")

		config := fault.RetryConfig{
			MaxAttempts:     maxAttempts,
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     10 * time.Millisecond,
			Multiplier:      1.5,
			RetryIf:         func(err error) bool { return true },
		}

		attempts := 0
		result := fault.RetryWithResult(context.Background(), config, func(ctx context.Context) (int, error) {
			attempts++
			return 0, errors.New("always fails")
		})

		if result.IsOk() {
			t.Fatal("exhausted retries should return Err")
		}

		if attempts != maxAttempts {
			t.Fatalf("expected %d attempts, got %d", maxAttempts, attempts)
		}

		var retryErr *fault.RetryExhaustedError
		if !errors.As(result.UnwrapErr(), &retryErr) {
			t.Fatalf("expected RetryExhaustedError, got %T: %v", result.UnwrapErr(), result.UnwrapErr())
		}

		if retryErr.Attempts != maxAttempts {
			t.Fatalf("RetryExhaustedError.Attempts: expected %d, got %d", maxAttempts, retryErr.Attempts)
		}
	})
}

// TestSuccessfulOperationReturnsOk verifies successful operation returns Ok.
func TestSuccessfulOperationReturnsOk(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		expectedValue := rapid.Int().Draw(t, "expectedValue")

		config := fault.RetryConfig{
			MaxAttempts:     3,
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     10 * time.Millisecond,
			Multiplier:      2.0,
			RetryIf:         func(err error) bool { return true },
		}

		result := fault.RetryWithResult(context.Background(), config, func(ctx context.Context) (int, error) {
			return expectedValue, nil
		})

		if result.IsErr() {
			t.Fatalf("successful operation should return Ok, got Err: %v", result.UnwrapErr())
		}

		if result.Unwrap() != expectedValue {
			t.Fatalf("expected %d, got %d", expectedValue, result.Unwrap())
		}
	})
}

// TestCircuitBreakerSuccessReturnsOk verifies successful circuit breaker returns Ok.
func TestCircuitBreakerSuccessReturnsOk(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		expectedValue := rapid.Int().Draw(t, "expectedValue")

		config := fault.CircuitBreakerConfig{
			Name:             "test-circuit",
			FailureThreshold: 5,
			SuccessThreshold: 1,
			Timeout:          1 * time.Second,
			HalfOpenRequests: 1,
		}

		cb, err := fault.NewCircuitBreaker(config)
		if err != nil {
			t.Fatalf("failed to create circuit breaker: %v", err)
		}

		result := fault.ExecuteWithResult(cb, context.Background(), func(ctx context.Context) (int, error) {
			return expectedValue, nil
		})

		if result.IsErr() {
			t.Fatalf("successful operation should return Ok, got Err: %v", result.UnwrapErr())
		}

		if result.Unwrap() != expectedValue {
			t.Fatalf("expected %d, got %d", expectedValue, result.Unwrap())
		}
	})
}

// TestRetryEventualSuccess verifies retry succeeds after initial failures.
func TestRetryEventualSuccess(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		failuresBeforeSuccess := rapid.IntRange(1, 3).Draw(t, "failuresBeforeSuccess")
		expectedValue := rapid.Int().Draw(t, "expectedValue")

		config := fault.RetryConfig{
			MaxAttempts:     failuresBeforeSuccess + 1,
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     10 * time.Millisecond,
			Multiplier:      1.5,
			RetryIf:         func(err error) bool { return true },
		}

		attempts := 0
		result := fault.RetryWithResult(context.Background(), config, func(ctx context.Context) (int, error) {
			attempts++
			if attempts <= failuresBeforeSuccess {
				return 0, errors.New("temporary failure")
			}
			return expectedValue, nil
		})

		if result.IsErr() {
			t.Fatalf("should succeed after %d failures, got Err: %v", failuresBeforeSuccess, result.UnwrapErr())
		}

		if result.Unwrap() != expectedValue {
			t.Fatalf("expected %d, got %d", expectedValue, result.Unwrap())
		}

		if attempts != failuresBeforeSuccess+1 {
			t.Fatalf("expected %d attempts, got %d", failuresBeforeSuccess+1, attempts)
		}
	})
}

// TestResilienceErrorsExtendBase verifies all errors extend ResilienceError.
func TestResilienceErrorsExtendBase(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "service")
		correlationID := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "correlationID")

		// Test CircuitOpenError
		circuitErr := fault.NewCircuitOpenError(service, correlationID, time.Now(), time.Second, 0.5)
		if circuitErr.Service != service {
			t.Fatalf("CircuitOpenError.Service: expected %s, got %s", service, circuitErr.Service)
		}
		if circuitErr.CorrelationID != correlationID {
			t.Fatalf("CircuitOpenError.CorrelationID: expected %s, got %s", correlationID, circuitErr.CorrelationID)
		}
		if circuitErr.Code != fault.ErrCodeCircuitOpen {
			t.Fatalf("CircuitOpenError.Code: expected %s, got %s", fault.ErrCodeCircuitOpen, circuitErr.Code)
		}

		// Test RateLimitError
		rateErr := fault.NewRateLimitError(service, correlationID, 100, time.Minute, time.Second)
		if rateErr.Service != service {
			t.Fatalf("RateLimitError.Service: expected %s, got %s", service, rateErr.Service)
		}
		if rateErr.Code != fault.ErrCodeRateLimited {
			t.Fatalf("RateLimitError.Code: expected %s, got %s", fault.ErrCodeRateLimited, rateErr.Code)
		}

		// Test TimeoutError
		timeoutErr := fault.NewTimeoutError(service, correlationID, time.Second, 2*time.Second, nil)
		if timeoutErr.Service != service {
			t.Fatalf("TimeoutError.Service: expected %s, got %s", service, timeoutErr.Service)
		}
		if timeoutErr.Code != fault.ErrCodeTimeout {
			t.Fatalf("TimeoutError.Code: expected %s, got %s", fault.ErrCodeTimeout, timeoutErr.Code)
		}

		// Test BulkheadFullError
		bulkheadErr := fault.NewBulkheadFullError(service, correlationID, 10, 5, 15)
		if bulkheadErr.Service != service {
			t.Fatalf("BulkheadFullError.Service: expected %s, got %s", service, bulkheadErr.Service)
		}
		if bulkheadErr.Code != fault.ErrCodeBulkheadFull {
			t.Fatalf("BulkheadFullError.Code: expected %s, got %s", fault.ErrCodeBulkheadFull, bulkheadErr.Code)
		}

		// Test RetryExhaustedError
		retryErr := fault.NewRetryExhaustedError(service, correlationID, 3, time.Second, nil)
		if retryErr.Service != service {
			t.Fatalf("RetryExhaustedError.Service: expected %s, got %s", service, retryErr.Service)
		}
		if retryErr.Code != fault.ErrCodeRetryExhausted {
			t.Fatalf("RetryExhaustedError.Code: expected %s, got %s", fault.ErrCodeRetryExhausted, retryErr.Code)
		}
	})
}

// TestContextRespected verifies all operations respect context.Done().
func TestContextRespected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		config := fault.RetryConfig{
			MaxAttempts:     10,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			Multiplier:      2.0,
			RetryIf:         func(err error) bool { return true },
		}

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel after first attempt
		attempts := 0
		result := fault.RetryWithResult(ctx, config, func(ctx context.Context) (int, error) {
			attempts++
			if attempts == 1 {
				cancel()
			}
			return 0, errors.New("failure")
		})

		if result.IsOk() {
			t.Fatal("should return Err when context cancelled")
		}

		// Should have stopped early due to cancellation
		if attempts > 2 {
			t.Fatalf("should have stopped early, got %d attempts", attempts)
		}
	})
}
