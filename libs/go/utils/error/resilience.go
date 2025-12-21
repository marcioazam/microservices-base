package error

import (
	"fmt"
	"time"
)

// ResilienceErrorCode represents the type of resilience error.
type ResilienceErrorCode string

const (
	ErrCircuitOpen        ResilienceErrorCode = "CIRCUIT_OPEN"
	ErrRateLimitExceeded  ResilienceErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrTimeout            ResilienceErrorCode = "TIMEOUT"
	ErrBulkheadFull       ResilienceErrorCode = "BULKHEAD_FULL"
	ErrRetryExhausted     ResilienceErrorCode = "RETRY_EXHAUSTED"
	ErrInvalidPolicy      ResilienceErrorCode = "INVALID_POLICY"
	ErrServiceUnavailable ResilienceErrorCode = "SERVICE_UNAVAILABLE"
	ErrValidation         ResilienceErrorCode = "VALIDATION_ERROR"
)

// ResilienceError represents errors from resilience operations.
type ResilienceError struct {
	Code       ResilienceErrorCode
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

// Is implements errors.Is for ResilienceError.
func (e *ResilienceError) Is(target error) bool {
	t, ok := target.(*ResilienceError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// WithMetadata adds metadata to the error.
func (e *ResilienceError) WithMetadata(key string, value any) *ResilienceError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
}

// WithCause adds a cause to the error.
func (e *ResilienceError) WithCause(cause error) *ResilienceError {
	e.Cause = cause
	return e
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

// NewServiceUnavailableError creates a service unavailable error.
func NewServiceUnavailableError(service string, cause error) *ResilienceError {
	return &ResilienceError{
		Code:    ErrServiceUnavailable,
		Message: "service is unavailable",
		Service: service,
		Cause:   cause,
	}
}

// NewValidationError creates a validation error.
func NewValidationError(field, message string) *ResilienceError {
	return &ResilienceError{
		Code:    ErrValidation,
		Message: message,
		Metadata: map[string]any{
			"field": field,
		},
	}
}

// IsCircuitOpen checks if the error is a circuit open error.
func IsCircuitOpen(err error) bool {
	var resErr *ResilienceError
	if As(err, &resErr) {
		return resErr.Code == ErrCircuitOpen
	}
	return false
}

// IsRateLimitExceeded checks if the error is a rate limit exceeded error.
func IsRateLimitExceeded(err error) bool {
	var resErr *ResilienceError
	if As(err, &resErr) {
		return resErr.Code == ErrRateLimitExceeded
	}
	return false
}

// IsTimeout checks if the error is a timeout error.
func IsTimeout(err error) bool {
	var resErr *ResilienceError
	if As(err, &resErr) {
		return resErr.Code == ErrTimeout
	}
	return false
}

// IsBulkheadFull checks if the error is a bulkhead full error.
func IsBulkheadFull(err error) bool {
	var resErr *ResilienceError
	if As(err, &resErr) {
		return resErr.Code == ErrBulkheadFull
	}
	return false
}

// IsRetryExhausted checks if the error is a retry exhausted error.
func IsRetryExhausted(err error) bool {
	var resErr *ResilienceError
	if As(err, &resErr) {
		return resErr.Code == ErrRetryExhausted
	}
	return false
}
