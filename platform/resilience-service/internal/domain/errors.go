// Package domain defines core interfaces and types for the resilience service.
package domain

import (
	"fmt"
	"time"
)

// ErrorCode represents the type of resilience error.
type ErrorCode string

const (
	ErrCircuitOpen        ErrorCode = "CIRCUIT_OPEN"
	ErrRateLimitExceeded  ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrTimeout            ErrorCode = "TIMEOUT"
	ErrBulkheadFull       ErrorCode = "BULKHEAD_FULL"
	ErrRetryExhausted     ErrorCode = "RETRY_EXHAUSTED"
	ErrInvalidPolicy      ErrorCode = "INVALID_POLICY"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// ResilienceError represents errors from resilience operations.
type ResilienceError struct {
	Code       ErrorCode
	Message    string
	Service    string
	RetryAfter time.Duration
	Metadata   map[string]any
	Cause      error
}

func (e *ResilienceError) Error() string {
	if e.Service != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Service, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *ResilienceError) Unwrap() error {
	return e.Cause
}

// NewCircuitOpenError creates a circuit open error.
func NewCircuitOpenError(service string) *ResilienceError {
	return &ResilienceError{
		Code:    ErrCircuitOpen,
		Message: "circuit breaker is open",
		Service: service,
	}
}

// NewRateLimitError creates a rate limit exceeded error.
func NewRateLimitError(service string, retryAfter time.Duration) *ResilienceError {
	return &ResilienceError{
		Code:       ErrRateLimitExceeded,
		Message:    "rate limit exceeded",
		Service:    service,
		RetryAfter: retryAfter,
	}
}

// NewTimeoutError creates a timeout error.
func NewTimeoutError(service string, timeout time.Duration) *ResilienceError {
	return &ResilienceError{
		Code:    ErrTimeout,
		Message: fmt.Sprintf("operation timed out after %v", timeout),
		Service: service,
		Metadata: map[string]any{
			"timeout": timeout,
		},
	}
}

// NewBulkheadFullError creates a bulkhead full error.
func NewBulkheadFullError(partition string) *ResilienceError {
	return &ResilienceError{
		Code:    ErrBulkheadFull,
		Message: "bulkhead capacity exceeded",
		Service: partition,
	}
}

// NewRetryExhaustedError creates a retry exhausted error.
func NewRetryExhaustedError(service string, attempts int, cause error) *ResilienceError {
	return &ResilienceError{
		Code:    ErrRetryExhausted,
		Message: fmt.Sprintf("retry exhausted after %d attempts", attempts),
		Service: service,
		Cause:   cause,
		Metadata: map[string]any{
			"attempts": attempts,
		},
	}
}

// NewInvalidPolicyError creates an invalid policy error.
func NewInvalidPolicyError(message string) *ResilienceError {
	return &ResilienceError{
		Code:    ErrInvalidPolicy,
		Message: message,
	}
}
