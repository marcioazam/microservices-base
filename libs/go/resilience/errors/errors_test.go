package errors

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 4: Error String Contains Required Fields**
// **Validates: Requirements 2.6**
func TestErrorStringContainsRequiredFields(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Error() contains code, service, and message", prop.ForAll(
		func(service, message string) bool {
			if service == "" {
				return true // Skip empty service
			}
			err := &ResilienceError{
				Code:    ErrCircuitOpen,
				Service: service,
				Message: message,
			}
			errStr := err.Error()
			return strings.Contains(errStr, string(ErrCircuitOpen)) &&
				strings.Contains(errStr, service) &&
				strings.Contains(errStr, message)
		},
		gen.AnyString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 5: Error Unwrap Returns Cause**
// **Validates: Requirements 2.7**
func TestErrorUnwrapReturnsCause(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Unwrap returns the exact cause error", prop.ForAll(
		func(msg string) bool {
			cause := errors.New(msg)
			err := &ResilienceError{
				Code:    ErrRetryExhausted,
				Service: "test-service",
				Message: "test message",
				Cause:   cause,
			}
			return err.Unwrap() == cause
		},
		gen.AnyString(),
	))

	properties.Property("Unwrap returns nil when no cause", prop.ForAll(
		func(service string) bool {
			err := &ResilienceError{
				Code:    ErrCircuitOpen,
				Service: service,
				Message: "test",
			}
			return err.Unwrap() == nil
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

func TestCircuitOpenError(t *testing.T) {
	t.Run("NewCircuitOpenError creates error", func(t *testing.T) {
		err := NewCircuitOpenError("my-service")
		if err.Code != ErrCircuitOpen {
			t.Errorf("expected %s, got %s", ErrCircuitOpen, err.Code)
		}
		if err.Service != "my-service" {
			t.Errorf("expected my-service, got %s", err.Service)
		}
		if !IsCircuitOpen(err) {
			t.Error("expected IsCircuitOpen to return true")
		}
	})

	t.Run("NewCircuitOpenErrorWithReset includes reset time", func(t *testing.T) {
		resetAt := time.Now().Add(time.Minute)
		err := NewCircuitOpenErrorWithReset("my-service", resetAt)
		if err.ResetAt != resetAt {
			t.Error("expected reset time to be set")
		}
	})
}

func TestRateLimitError(t *testing.T) {
	t.Run("NewRateLimitError creates error", func(t *testing.T) {
		err := NewRateLimitError("my-service", 5*time.Second)
		if err.Code != ErrRateLimitExceeded {
			t.Errorf("expected %s, got %s", ErrRateLimitExceeded, err.Code)
		}
		if err.RetryAfter != 5*time.Second {
			t.Errorf("expected 5s, got %s", err.RetryAfter)
		}
		if !IsRateLimitExceeded(err) {
			t.Error("expected IsRateLimitExceeded to return true")
		}
	})

	t.Run("NewRateLimitErrorWithDetails includes details", func(t *testing.T) {
		err := NewRateLimitErrorWithDetails("my-service", 5*time.Second, 100, 0)
		if err.Limit != 100 || err.Remaining != 0 {
			t.Error("expected limit and remaining to be set")
		}
	})
}

func TestTimeoutError(t *testing.T) {
	t.Run("NewTimeoutError creates error", func(t *testing.T) {
		err := NewTimeoutError("my-service", 30*time.Second)
		if err.Code != ErrTimeout {
			t.Errorf("expected %s, got %s", ErrTimeout, err.Code)
		}
		if err.Timeout != 30*time.Second {
			t.Errorf("expected 30s, got %s", err.Timeout)
		}
		if !IsTimeout(err) {
			t.Error("expected IsTimeout to return true")
		}
	})

	t.Run("NewTimeoutErrorWithOperation includes operation", func(t *testing.T) {
		err := NewTimeoutErrorWithOperation("my-service", "fetchData", 30*time.Second)
		if err.Operation != "fetchData" {
			t.Errorf("expected fetchData, got %s", err.Operation)
		}
	})
}

func TestBulkheadFullError(t *testing.T) {
	t.Run("NewBulkheadFullError creates error", func(t *testing.T) {
		err := NewBulkheadFullError("my-service", "api-calls")
		if err.Code != ErrBulkheadFull {
			t.Errorf("expected %s, got %s", ErrBulkheadFull, err.Code)
		}
		if err.Partition != "api-calls" {
			t.Errorf("expected api-calls, got %s", err.Partition)
		}
		if !IsBulkheadFull(err) {
			t.Error("expected IsBulkheadFull to return true")
		}
	})

	t.Run("NewBulkheadFullErrorWithDetails includes details", func(t *testing.T) {
		err := NewBulkheadFullErrorWithDetails("my-service", "api-calls", 10, 5)
		if err.MaxConcurrent != 10 || err.QueueSize != 5 {
			t.Error("expected max concurrent and queue size to be set")
		}
	})
}

func TestRetryExhaustedError(t *testing.T) {
	t.Run("NewRetryExhaustedError creates error with cause", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := NewRetryExhaustedError("my-service", 3, cause)
		if err.Code != ErrRetryExhausted {
			t.Errorf("expected %s, got %s", ErrRetryExhausted, err.Code)
		}
		if err.Attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", err.Attempts)
		}
		if err.Unwrap() != cause {
			t.Error("expected cause to be unwrapped")
		}
		if !IsRetryExhausted(err) {
			t.Error("expected IsRetryExhausted to return true")
		}
	})
}

func TestInvalidPolicyError(t *testing.T) {
	err := NewInvalidPolicyError("my-service", "default", "timeout", "must be positive")
	if err.Code != ErrInvalidPolicy {
		t.Errorf("expected %s, got %s", ErrInvalidPolicy, err.Code)
	}
	if err.PolicyName != "default" {
		t.Errorf("expected default, got %s", err.PolicyName)
	}
	if err.Field != "timeout" {
		t.Errorf("expected timeout, got %s", err.Field)
	}
}

func TestErrorIs(t *testing.T) {
	err1 := &ResilienceError{Code: ErrCircuitOpen}
	err2 := &ResilienceError{Code: ErrCircuitOpen}
	err3 := &ResilienceError{Code: ErrTimeout}

	if !err1.Is(err2) {
		t.Error("expected errors with same code to match")
	}
	if err1.Is(err3) {
		t.Error("expected errors with different codes to not match")
	}
}
