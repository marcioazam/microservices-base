package error

import (
	"errors"
	"testing"
	"time"
)

func TestResilienceErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ResilienceError
		expected string
	}{
		{
			name: "with service",
			err: &ResilienceError{
				Code:    ErrCircuitOpen,
				Message: "circuit breaker is open",
				Service: "payment-service",
			},
			expected: "[CIRCUIT_OPEN] payment-service: circuit breaker is open",
		},
		{
			name: "without service",
			err: &ResilienceError{
				Code:    ErrInvalidPolicy,
				Message: "invalid configuration",
			},
			expected: "[INVALID_POLICY] invalid configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestResilienceErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &ResilienceError{
		Code:    ErrRetryExhausted,
		Message: "retry exhausted",
		Cause:   cause,
	}

	if unwrapped := err.Unwrap(); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestResilienceErrorIs(t *testing.T) {
	err1 := &ResilienceError{Code: ErrCircuitOpen}
	err2 := &ResilienceError{Code: ErrCircuitOpen}
	err3 := &ResilienceError{Code: ErrTimeout}

	if !err1.Is(err2) {
		t.Error("expected err1.Is(err2) to be true")
	}
	if err1.Is(err3) {
		t.Error("expected err1.Is(err3) to be false")
	}
	if err1.Is(errors.New("other")) {
		t.Error("expected err1.Is(other) to be false")
	}
}

func TestResilienceErrorWithMetadata(t *testing.T) {
	err := NewCircuitOpenError("test-service").
		WithMetadata("attempts", 5).
		WithMetadata("last_error", "connection refused")

	if err.Metadata["attempts"] != 5 {
		t.Errorf("expected attempts=5, got %v", err.Metadata["attempts"])
	}
	if err.Metadata["last_error"] != "connection refused" {
		t.Errorf("expected last_error='connection refused', got %v", err.Metadata["last_error"])
	}
}

func TestResilienceErrorWithCause(t *testing.T) {
	cause := errors.New("root cause")
	err := NewCircuitOpenError("test-service").WithCause(cause)

	if err.Cause != cause {
		t.Errorf("expected cause=%v, got %v", cause, err.Cause)
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *ResilienceError
		code     ResilienceErrorCode
		hasRetry bool
	}{
		{
			name: "circuit open",
			err:  NewCircuitOpenError("svc"),
			code: ErrCircuitOpen,
		},
		{
			name:     "rate limit",
			err:      NewRateLimitError("svc", 5*time.Second),
			code:     ErrRateLimitExceeded,
			hasRetry: true,
		},
		{
			name: "timeout",
			err:  NewTimeoutError("svc", 30*time.Second),
			code: ErrTimeout,
		},
		{
			name: "bulkhead full",
			err:  NewBulkheadFullError("partition"),
			code: ErrBulkheadFull,
		},
		{
			name: "retry exhausted",
			err:  NewRetryExhaustedError("svc", 3, errors.New("cause")),
			code: ErrRetryExhausted,
		},
		{
			name: "invalid policy",
			err:  NewInvalidPolicyError("bad config"),
			code: ErrInvalidPolicy,
		},
		{
			name: "service unavailable",
			err:  NewServiceUnavailableError("svc", errors.New("cause")),
			code: ErrServiceUnavailable,
		},
		{
			name: "validation",
			err:  NewValidationError("field", "must be positive"),
			code: ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("expected code %s, got %s", tt.code, tt.err.Code)
			}
			if tt.hasRetry && tt.err.RetryAfter == 0 {
				t.Error("expected non-zero RetryAfter")
			}
		})
	}
}

func TestIsErrorHelpers(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		check  func(error) bool
		expect bool
	}{
		{"circuit open - true", NewCircuitOpenError("svc"), IsCircuitOpen, true},
		{"circuit open - false", NewTimeoutError("svc", time.Second), IsCircuitOpen, false},
		{"rate limit - true", NewRateLimitError("svc", time.Second), IsRateLimitExceeded, true},
		{"rate limit - false", NewCircuitOpenError("svc"), IsRateLimitExceeded, false},
		{"timeout - true", NewTimeoutError("svc", time.Second), IsTimeout, true},
		{"timeout - false", NewCircuitOpenError("svc"), IsTimeout, false},
		{"bulkhead - true", NewBulkheadFullError("p"), IsBulkheadFull, true},
		{"bulkhead - false", NewCircuitOpenError("svc"), IsBulkheadFull, false},
		{"retry - true", NewRetryExhaustedError("svc", 3, nil), IsRetryExhausted, true},
		{"retry - false", NewCircuitOpenError("svc"), IsRetryExhausted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.check(tt.err); got != tt.expect {
				t.Errorf("expected %v, got %v", tt.expect, got)
			}
		})
	}
}

func TestErrorsAsWithResilienceError(t *testing.T) {
	cause := errors.New("root cause")
	wrapped := NewRetryExhaustedError("svc", 3, cause)

	var resErr *ResilienceError
	if !As(wrapped, &resErr) {
		t.Error("expected errors.As to succeed")
	}
	if resErr.Code != ErrRetryExhausted {
		t.Errorf("expected code %s, got %s", ErrRetryExhausted, resErr.Code)
	}
}
