package fault

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	apperrors "github.com/authcorp/libs/go/src/errors"
)

// ErrorCode represents resilience error types.
type ErrorCode string

const (
	ErrCodeCircuitOpen    ErrorCode = "CIRCUIT_OPEN"
	ErrCodeRateLimited    ErrorCode = "RATE_LIMITED"
	ErrCodeTimeout        ErrorCode = "TIMEOUT"
	ErrCodeBulkheadFull   ErrorCode = "BULKHEAD_FULL"
	ErrCodeRetryExhausted ErrorCode = "RETRY_EXHAUSTED"
	ErrCodeInvalidPolicy  ErrorCode = "INVALID_POLICY"
)

// ResilienceError is the base error type for all resilience errors.
// It extends AppError to provide consistent error handling across the application.
type ResilienceError struct {
	*apperrors.AppError
	Code          ErrorCode `json:"code"`
	Service       string    `json:"service"`
	Pattern       string    `json:"pattern,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	Cause         error     `json:"-"`
}

// NewResilienceError creates a new ResilienceError with AppError base.
func NewResilienceError(code ErrorCode, service, message, pattern string) *ResilienceError {
	return &ResilienceError{
		AppError: &apperrors.AppError{
			Code:      apperrors.ErrCodeUnavailable,
			Message:   message,
			Timestamp: time.Now(),
		},
		Code:      code,
		Service:   service,
		Pattern:   pattern,
		Timestamp: time.Now(),
	}
}

// Error implements the error interface.
func (e *ResilienceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)", e.Code, e.Service, e.AppError.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Service, e.AppError.Message)
}

// Unwrap returns the underlying cause.
func (e *ResilienceError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return e.AppError
}

// Is checks if the error matches a target.
func (e *ResilienceError) Is(target error) bool {
	if t, ok := target.(*ResilienceError); ok {
		return e.Code == t.Code
	}
	return errors.Is(e.AppError, target) || errors.Is(e.Cause, target)
}

// HTTPStatus returns the HTTP status code for this error.
func (e *ResilienceError) HTTPStatus() int {
	switch e.Code {
	case ErrCodeCircuitOpen, ErrCodeBulkheadFull:
		return http.StatusServiceUnavailable
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case ErrCodeTimeout:
		return http.StatusGatewayTimeout
	case ErrCodeRetryExhausted:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// MarshalJSON implements json.Marshaler.
func (e *ResilienceError) MarshalJSON() ([]byte, error) {
	aux := struct {
		Code          ErrorCode `json:"code"`
		Service       string    `json:"service"`
		Message       string    `json:"message"`
		Pattern       string    `json:"pattern,omitempty"`
		CorrelationID string    `json:"correlation_id,omitempty"`
		Timestamp     time.Time `json:"timestamp"`
		Cause         string    `json:"cause,omitempty"`
	}{
		Code:          e.Code,
		Service:       e.Service,
		Message:       e.AppError.Message,
		Pattern:       e.Pattern,
		CorrelationID: e.CorrelationID,
		Timestamp:     e.Timestamp,
	}
	if e.Cause != nil {
		aux.Cause = e.Cause.Error()
	}
	return json.Marshal(aux)
}

// UnmarshalJSON implements json.Unmarshaler.
func (e *ResilienceError) UnmarshalJSON(data []byte) error {
	aux := struct {
		Code          ErrorCode `json:"code"`
		Service       string    `json:"service"`
		Message       string    `json:"message"`
		Pattern       string    `json:"pattern,omitempty"`
		CorrelationID string    `json:"correlation_id,omitempty"`
		Timestamp     time.Time `json:"timestamp"`
		Cause         string    `json:"cause,omitempty"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	e.Code = aux.Code
	e.Service = aux.Service
	e.Pattern = aux.Pattern
	e.CorrelationID = aux.CorrelationID
	e.Timestamp = aux.Timestamp
	e.AppError = &apperrors.AppError{
		Code:      apperrors.ErrCodeUnavailable,
		Message:   aux.Message,
		Timestamp: aux.Timestamp,
	}
	if aux.Cause != "" {
		e.Cause = fmt.Errorf("%s", aux.Cause)
	}
	return nil
}

// CircuitOpenError indicates the circuit breaker is open.
type CircuitOpenError struct {
	*ResilienceError
	OpenedAt    time.Time     `json:"opened_at"`
	ResetAfter  time.Duration `json:"reset_after"`
	FailureRate float64       `json:"failure_rate"`
}

// NewCircuitOpenError creates a new CircuitOpenError.
func NewCircuitOpenError(service, correlationID string, openedAt time.Time, resetAfter time.Duration, failureRate float64) *CircuitOpenError {
	return &CircuitOpenError{
		ResilienceError: &ResilienceError{
			AppError: &apperrors.AppError{
				Code:      apperrors.ErrCodeUnavailable,
				Message:   "circuit breaker is open",
				Timestamp: time.Now(),
			},
			Code:          ErrCodeCircuitOpen,
			Service:       service,
			Pattern:       "circuit_breaker",
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		},
		OpenedAt:    openedAt,
		ResetAfter:  resetAfter,
		FailureRate: failureRate,
	}
}

// Error implements the error interface.
func (e *CircuitOpenError) Error() string {
	return e.ResilienceError.Error()
}

// Unwrap returns the underlying ResilienceError.
func (e *CircuitOpenError) Unwrap() error {
	return e.ResilienceError
}

// RateLimitError indicates rate limit exceeded.
type RateLimitError struct {
	*ResilienceError
	Limit      int           `json:"limit"`
	Window     time.Duration `json:"window"`
	RetryAfter time.Duration `json:"retry_after"`
}

// NewRateLimitError creates a new RateLimitError.
func NewRateLimitError(service, correlationID string, limit int, window, retryAfter time.Duration) *RateLimitError {
	return &RateLimitError{
		ResilienceError: &ResilienceError{
			AppError: &apperrors.AppError{
				Code:      apperrors.ErrCodeTooManyReqs,
				Message:   "rate limit exceeded",
				Timestamp: time.Now(),
			},
			Code:          ErrCodeRateLimited,
			Service:       service,
			Pattern:       "rate_limiter",
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		},
		Limit:      limit,
		Window:     window,
		RetryAfter: retryAfter,
	}
}

// Error implements the error interface.
func (e *RateLimitError) Error() string {
	return e.ResilienceError.Error()
}

// Unwrap returns the underlying ResilienceError.
func (e *RateLimitError) Unwrap() error {
	return e.ResilienceError
}

// MarshalJSON implements json.Marshaler for RateLimitError.
func (e *RateLimitError) MarshalJSON() ([]byte, error) {
	aux := struct {
		Code          ErrorCode `json:"code"`
		Service       string    `json:"service"`
		Message       string    `json:"message"`
		Pattern       string    `json:"pattern"`
		CorrelationID string    `json:"correlation_id,omitempty"`
		Timestamp     time.Time `json:"timestamp"`
		Limit         int       `json:"limit"`
		Window        int64     `json:"window"`
		RetryAfter    int64     `json:"retry_after"`
	}{
		Code:          e.Code,
		Service:       e.Service,
		Message:       e.AppError.Message,
		Pattern:       e.Pattern,
		CorrelationID: e.CorrelationID,
		Timestamp:     e.Timestamp,
		Limit:         e.Limit,
		Window:        int64(e.Window),
		RetryAfter:    int64(e.RetryAfter),
	}
	return json.Marshal(aux)
}

// TimeoutError indicates operation timed out.
type TimeoutError struct {
	*ResilienceError
	Timeout time.Duration `json:"timeout"`
	Elapsed time.Duration `json:"elapsed"`
}

// NewTimeoutError creates a new TimeoutError.
func NewTimeoutError(service, correlationID string, timeout, elapsed time.Duration, cause error) *TimeoutError {
	return &TimeoutError{
		ResilienceError: &ResilienceError{
			AppError: &apperrors.AppError{
				Code:      apperrors.ErrCodeTimeout,
				Message:   "operation timed out",
				Timestamp: time.Now(),
			},
			Code:          ErrCodeTimeout,
			Service:       service,
			Pattern:       "timeout",
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
			Cause:         cause,
		},
		Timeout: timeout,
		Elapsed: elapsed,
	}
}

// Error implements the error interface.
func (e *TimeoutError) Error() string {
	return e.ResilienceError.Error()
}

// Unwrap returns the underlying ResilienceError.
func (e *TimeoutError) Unwrap() error {
	return e.ResilienceError
}

// BulkheadFullError indicates bulkhead capacity exceeded.
type BulkheadFullError struct {
	*ResilienceError
	MaxConcurrent int `json:"max_concurrent"`
	QueueSize     int `json:"queue_size"`
	CurrentLoad   int `json:"current_load"`
}

// NewBulkheadFullError creates a new BulkheadFullError.
func NewBulkheadFullError(service, correlationID string, maxConcurrent, queueSize, currentLoad int) *BulkheadFullError {
	return &BulkheadFullError{
		ResilienceError: &ResilienceError{
			AppError: &apperrors.AppError{
				Code:      apperrors.ErrCodeUnavailable,
				Message:   "bulkhead capacity exceeded",
				Timestamp: time.Now(),
			},
			Code:          ErrCodeBulkheadFull,
			Service:       service,
			Pattern:       "bulkhead",
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		},
		MaxConcurrent: maxConcurrent,
		QueueSize:     queueSize,
		CurrentLoad:   currentLoad,
	}
}

// Error implements the error interface.
func (e *BulkheadFullError) Error() string {
	return e.ResilienceError.Error()
}

// Unwrap returns the underlying ResilienceError.
func (e *BulkheadFullError) Unwrap() error {
	return e.ResilienceError
}

// RetryExhaustedError indicates all retries failed.
type RetryExhaustedError struct {
	*ResilienceError
	Attempts   int           `json:"attempts"`
	TotalTime  time.Duration `json:"total_time"`
	LastErrors []error       `json:"-"`
}

// NewRetryExhaustedError creates a new RetryExhaustedError.
func NewRetryExhaustedError(service, correlationID string, attempts int, totalTime time.Duration, lastErrors []error) *RetryExhaustedError {
	var cause error
	if len(lastErrors) > 0 {
		cause = lastErrors[len(lastErrors)-1]
	}
	return &RetryExhaustedError{
		ResilienceError: &ResilienceError{
			AppError: &apperrors.AppError{
				Code:      apperrors.ErrCodeDependency,
				Message:   "all retry attempts exhausted",
				Timestamp: time.Now(),
			},
			Code:          ErrCodeRetryExhausted,
			Service:       service,
			Pattern:       "retry",
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
			Cause:         cause,
		},
		Attempts:   attempts,
		TotalTime:  totalTime,
		LastErrors: lastErrors,
	}
}

// Error implements the error interface.
func (e *RetryExhaustedError) Error() string {
	return e.ResilienceError.Error()
}

// Unwrap returns the underlying ResilienceError.
func (e *RetryExhaustedError) Unwrap() error {
	return e.ResilienceError
}

// InvalidPolicyError indicates invalid configuration.
type InvalidPolicyError struct {
	*ResilienceError
	Field    string `json:"field"`
	Value    any    `json:"value"`
	Expected string `json:"expected"`
}

// NewInvalidPolicyError creates a new InvalidPolicyError.
func NewInvalidPolicyError(field string, value any, expected string) *InvalidPolicyError {
	return &InvalidPolicyError{
		ResilienceError: &ResilienceError{
			AppError: &apperrors.AppError{
				Code:      apperrors.ErrCodeValidation,
				Message:   fmt.Sprintf("invalid policy: %s", field),
				Timestamp: time.Now(),
			},
			Code:      ErrCodeInvalidPolicy,
			Pattern:   "configuration",
			Timestamp: time.Now(),
		},
		Field:    field,
		Value:    value,
		Expected: expected,
	}
}

// Error implements the error interface.
func (e *InvalidPolicyError) Error() string {
	return e.ResilienceError.Error()
}

// Unwrap returns the underlying ResilienceError.
func (e *InvalidPolicyError) Unwrap() error {
	return e.ResilienceError
}
