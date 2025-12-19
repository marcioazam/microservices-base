// Package errors provides resilience-specific error types.
package errors

import (
	"fmt"
	"time"
)

// ErrorCode represents a resilience error code.
type ErrorCode string

const (
	ErrCircuitOpen       ErrorCode = "CIRCUIT_OPEN"
	ErrRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrTimeout           ErrorCode = "TIMEOUT"
	ErrBulkheadFull      ErrorCode = "BULKHEAD_FULL"
	ErrRetryExhausted    ErrorCode = "RETRY_EXHAUSTED"
	ErrInvalidPolicy     ErrorCode = "INVALID_POLICY"
)

// ResilienceError is the base error type for all resilience errors.
type ResilienceError struct {
	Code    ErrorCode
	Service string
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *ResilienceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (cause: %v)", e.Code, e.Service, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Service, e.Message)
}

// Unwrap returns the underlying cause error.
func (e *ResilienceError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target.
func (e *ResilienceError) Is(target error) bool {
	if t, ok := target.(*ResilienceError); ok {
		return e.Code == t.Code
	}
	return false
}

// CircuitOpenError represents a circuit breaker open error.
type CircuitOpenError struct {
	ResilienceError
	ResetAt time.Time
}

// NewCircuitOpenError creates a new circuit open error.
func NewCircuitOpenError(service string) *CircuitOpenError {
	return &CircuitOpenError{
		ResilienceError: ResilienceError{
			Code:    ErrCircuitOpen,
			Service: service,
			Message: "circuit breaker is open",
		},
	}
}

// NewCircuitOpenErrorWithReset creates a circuit open error with reset time.
func NewCircuitOpenErrorWithReset(service string, resetAt time.Time) *CircuitOpenError {
	return &CircuitOpenError{
		ResilienceError: ResilienceError{
			Code:    ErrCircuitOpen,
			Service: service,
			Message: fmt.Sprintf("circuit breaker is open, resets at %s", resetAt.Format(time.RFC3339)),
		},
		ResetAt: resetAt,
	}
}

// RateLimitError represents a rate limit exceeded error.
type RateLimitError struct {
	ResilienceError
	RetryAfter time.Duration
	Limit      int
	Remaining  int
}

// NewRateLimitError creates a new rate limit error.
func NewRateLimitError(service string, retryAfter time.Duration) *RateLimitError {
	return &RateLimitError{
		ResilienceError: ResilienceError{
			Code:    ErrRateLimitExceeded,
			Service: service,
			Message: fmt.Sprintf("rate limit exceeded, retry after %s", retryAfter),
		},
		RetryAfter: retryAfter,
	}
}

// NewRateLimitErrorWithDetails creates a rate limit error with details.
func NewRateLimitErrorWithDetails(service string, retryAfter time.Duration, limit, remaining int) *RateLimitError {
	return &RateLimitError{
		ResilienceError: ResilienceError{
			Code:    ErrRateLimitExceeded,
			Service: service,
			Message: fmt.Sprintf("rate limit exceeded (%d/%d), retry after %s", remaining, limit, retryAfter),
		},
		RetryAfter: retryAfter,
		Limit:      limit,
		Remaining:  remaining,
	}
}

// TimeoutError represents a timeout error.
type TimeoutError struct {
	ResilienceError
	Timeout   time.Duration
	Operation string
}

// NewTimeoutError creates a new timeout error.
func NewTimeoutError(service string, timeout time.Duration) *TimeoutError {
	return &TimeoutError{
		ResilienceError: ResilienceError{
			Code:    ErrTimeout,
			Service: service,
			Message: fmt.Sprintf("operation timed out after %s", timeout),
		},
		Timeout: timeout,
	}
}

// NewTimeoutErrorWithOperation creates a timeout error with operation name.
func NewTimeoutErrorWithOperation(service, operation string, timeout time.Duration) *TimeoutError {
	return &TimeoutError{
		ResilienceError: ResilienceError{
			Code:    ErrTimeout,
			Service: service,
			Message: fmt.Sprintf("operation '%s' timed out after %s", operation, timeout),
		},
		Timeout:   timeout,
		Operation: operation,
	}
}

// BulkheadFullError represents a bulkhead full error.
type BulkheadFullError struct {
	ResilienceError
	Partition     string
	MaxConcurrent int
	QueueSize     int
}

// NewBulkheadFullError creates a new bulkhead full error.
func NewBulkheadFullError(service, partition string) *BulkheadFullError {
	return &BulkheadFullError{
		ResilienceError: ResilienceError{
			Code:    ErrBulkheadFull,
			Service: service,
			Message: fmt.Sprintf("bulkhead '%s' is full", partition),
		},
		Partition: partition,
	}
}

// NewBulkheadFullErrorWithDetails creates a bulkhead full error with details.
func NewBulkheadFullErrorWithDetails(service, partition string, maxConcurrent, queueSize int) *BulkheadFullError {
	return &BulkheadFullError{
		ResilienceError: ResilienceError{
			Code:    ErrBulkheadFull,
			Service: service,
			Message: fmt.Sprintf("bulkhead '%s' is full (max: %d, queue: %d)", partition, maxConcurrent, queueSize),
		},
		Partition:     partition,
		MaxConcurrent: maxConcurrent,
		QueueSize:     queueSize,
	}
}

// RetryExhaustedError represents a retry exhausted error.
type RetryExhaustedError struct {
	ResilienceError
	Attempts int
}

// NewRetryExhaustedError creates a new retry exhausted error.
func NewRetryExhaustedError(service string, attempts int, cause error) *RetryExhaustedError {
	return &RetryExhaustedError{
		ResilienceError: ResilienceError{
			Code:    ErrRetryExhausted,
			Service: service,
			Message: fmt.Sprintf("all %d retry attempts exhausted", attempts),
			Cause:   cause,
		},
		Attempts: attempts,
	}
}

// InvalidPolicyError represents an invalid policy error.
type InvalidPolicyError struct {
	ResilienceError
	PolicyName string
	Field      string
}

// NewInvalidPolicyError creates a new invalid policy error.
func NewInvalidPolicyError(service, policyName, field, reason string) *InvalidPolicyError {
	return &InvalidPolicyError{
		ResilienceError: ResilienceError{
			Code:    ErrInvalidPolicy,
			Service: service,
			Message: fmt.Sprintf("invalid policy '%s': field '%s' %s", policyName, field, reason),
		},
		PolicyName: policyName,
		Field:      field,
	}
}

// IsCircuitOpen checks if the error is a circuit open error.
func IsCircuitOpen(err error) bool {
	if e, ok := err.(*CircuitOpenError); ok {
		return e.Code == ErrCircuitOpen
	}
	if e, ok := err.(*ResilienceError); ok {
		return e.Code == ErrCircuitOpen
	}
	return false
}

// IsRateLimitExceeded checks if the error is a rate limit error.
func IsRateLimitExceeded(err error) bool {
	if e, ok := err.(*RateLimitError); ok {
		return e.Code == ErrRateLimitExceeded
	}
	if e, ok := err.(*ResilienceError); ok {
		return e.Code == ErrRateLimitExceeded
	}
	return false
}

// IsTimeout checks if the error is a timeout error.
func IsTimeout(err error) bool {
	if e, ok := err.(*TimeoutError); ok {
		return e.Code == ErrTimeout
	}
	if e, ok := err.(*ResilienceError); ok {
		return e.Code == ErrTimeout
	}
	return false
}

// IsBulkheadFull checks if the error is a bulkhead full error.
func IsBulkheadFull(err error) bool {
	if e, ok := err.(*BulkheadFullError); ok {
		return e.Code == ErrBulkheadFull
	}
	if e, ok := err.(*ResilienceError); ok {
		return e.Code == ErrBulkheadFull
	}
	return false
}

// IsRetryExhausted checks if the error is a retry exhausted error.
func IsRetryExhausted(err error) bool {
	if e, ok := err.(*RetryExhaustedError); ok {
		return e.Code == ErrRetryExhausted
	}
	if e, ok := err.(*ResilienceError); ok {
		return e.Code == ErrRetryExhausted
	}
	return false
}
