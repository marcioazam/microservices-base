// Package property contains property-based tests for IAM Policy Service.
package property

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/auth-platform/iam-policy-service/tests/testutil"
	"github.com/authcorp/libs/go/src/fault"
	"pgregory.net/rapid"
)

// TestCircuitBreakerStateTransitions validates Property 12.
// Circuit breaker state transitions must be correct.
// Note: Uses shared fault library - Service Mesh handles resilience in production.
func TestCircuitBreakerStateTransitions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		threshold := rapid.IntRange(1, 10).Draw(t, "threshold")

		config := fault.CircuitBreakerConfig{
			Name:             "test-cb",
			FailureThreshold: threshold,
			SuccessThreshold: 2,
			Timeout:          100 * time.Millisecond,
		}

		cb, err := fault.NewCircuitBreaker(config)
		if err != nil {
			t.Fatalf("failed to create circuit breaker: %v", err)
		}

		// Property: initial state must be closed
		if cb.State() != fault.StateClosed {
			t.Errorf("initial state should be closed, got %s", cb.State())
		}

		// Cause failures to open circuit
		failErr := errors.New("test failure")
		for i := 0; i < threshold; i++ {
			_ = cb.Execute(context.Background(), func(ctx context.Context) error {
				return failErr
			})
		}

		// Property: circuit must be open after threshold failures
		if cb.State() != fault.StateOpen {
			t.Errorf("state should be open after %d failures, got %s", threshold, cb.State())
		}

		// Property: requests should be rejected when open
		err = cb.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
		var circuitOpenErr *fault.CircuitOpenError
		if !errors.As(err, &circuitOpenErr) {
			t.Error("should return CircuitOpenError when circuit is open")
		}
	})
}

// TestCircuitBreakerRecovery validates circuit breaker recovery.
func TestCircuitBreakerRecovery(t *testing.T) {
	config := fault.CircuitBreakerConfig{
		Name:             "test-recovery",
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}

	cb, err := fault.NewCircuitBreaker(config)
	if err != nil {
		t.Fatalf("failed to create circuit breaker: %v", err)
	}

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("fail")
		})
	}

	if cb.State() != fault.StateOpen {
		t.Fatalf("expected open state, got %s", cb.State())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Execute successful requests
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
	}

	// Property: circuit should be closed after successful requests
	if cb.State() != fault.StateClosed {
		t.Errorf("expected closed state after recovery, got %s", cb.State())
	}
}

// TestCircuitBreakerReset validates reset functionality.
func TestCircuitBreakerReset(t *testing.T) {
	config := fault.CircuitBreakerConfig{
		Name:             "test-reset",
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}

	cb, err := fault.NewCircuitBreaker(config)
	if err != nil {
		t.Fatalf("failed to create circuit breaker: %v", err)
	}

	// Open the circuit
	for i := 0; i < config.FailureThreshold; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("fail")
		})
	}

	// Reset
	cb.Reset()

	// Property: state should be closed after reset
	if cb.State() != fault.StateClosed {
		t.Errorf("expected closed state after reset, got %s", cb.State())
	}

	// Property: requests should be allowed after reset
	err = cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("request should succeed after reset, got %v", err)
	}
}

// TestErrorResponseConsistency validates Property 13.
// Error responses must be consistent.
func TestErrorResponseConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.SampledFrom([]string{
			"INVALID_INPUT",
			"UNAUTHORIZED",
			"FORBIDDEN",
			"NOT_FOUND",
			"INTERNAL",
		}).Draw(t, "code")
		message := testutil.NonEmptyStringGen().Draw(t, "message")
		correlationID := testutil.CorrelationIDGen().Draw(t, "correlationID")

		err := testutil.NewMockServiceError(code, message, correlationID)

		// Property: error must contain code
		if err.Code != code {
			t.Errorf("code mismatch: expected %s, got %s", code, err.Code)
		}

		// Property: error must contain message
		if err.Message != message {
			t.Errorf("message mismatch: expected %s, got %s", message, err.Message)
		}

		// Property: error must contain correlation ID
		if err.CorrelationID != correlationID {
			t.Errorf("correlation ID mismatch: expected %s, got %s", correlationID, err.CorrelationID)
		}

		// Property: error string must be non-empty
		if err.Error() == "" {
			t.Error("error string should not be empty")
		}
	})
}

// TestErrorCodeMapping validates error code to gRPC mapping.
func TestErrorCodeMapping(t *testing.T) {
	testCases := []struct {
		code     string
		expected string
	}{
		{"INVALID_INPUT", "InvalidArgument"},
		{"UNAUTHORIZED", "Unauthenticated"},
		{"FORBIDDEN", "PermissionDenied"},
		{"NOT_FOUND", "NotFound"},
		{"INTERNAL", "Internal"},
	}

	for _, tc := range testCases {
		err := testutil.NewMockServiceError(tc.code, "test", "")
		grpcCode := testutil.MapToGRPCCode(err.Code)

		if grpcCode != tc.expected {
			t.Errorf("code %s: expected gRPC code %s, got %s", tc.code, tc.expected, grpcCode)
		}
	}
}
