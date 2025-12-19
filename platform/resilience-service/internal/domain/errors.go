// Package domain defines core interfaces and types for the resilience service.
// This package re-exports types from libs/go/resilience/errors for backward compatibility.
package domain

import (
	"fmt"
	"time"

	liberrors "github.com/auth-platform/libs/go/resilience/errors"
)

// ErrorCode represents the type of resilience error.
// Re-exported from libs/go/resilience/errors for backward compatibility.
type ErrorCode = liberrors.ErrorCode

// Error code constants - re-exported for backward compatibility.
const (
	ErrCircuitOpen        ErrorCode = liberrors.ErrCircuitOpen
	ErrRateLimitExceeded  ErrorCode = liberrors.ErrRateLimitExceeded
	ErrTimeout            ErrorCode = liberrors.ErrTimeout
	ErrBulkheadFull       ErrorCode = liberrors.ErrBulkheadFull
	ErrRetryExhausted     ErrorCode = liberrors.ErrRetryExhausted
	ErrInvalidPolicy      ErrorCode = liberrors.ErrInvalidPolicy
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// ResilienceError represents errors from resilience operations.
// This maintains backward compatibility with the original API.
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

// IsCircuitOpen checks if the error is a circuit open error.
var IsCircuitOpen = liberrors.IsCircuitOpen

// IsRateLimitExceeded checks if the error is a rate limit error.
var IsRateLimitExceeded = liberrors.IsRateLimitExceeded

// IsTimeout checks if the error is a timeout error.
var IsTimeout = liberrors.IsTimeout

// IsBulkheadFull checks if the error is a bulkhead full error.
var IsBulkheadFull = liberrors.IsBulkheadFull

// IsRetryExhausted checks if the error is a retry exhausted error.
var IsRetryExhausted = liberrors.IsRetryExhausted
