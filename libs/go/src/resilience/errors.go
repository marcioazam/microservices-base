package resilience

import (
	"encoding/json"
	"fmt"
	"time"
)

// ErrorCode represents resilience error types.
type ErrorCode string

const (
	ErrCodeCircuitOpen   ErrorCode = "CIRCUIT_OPEN"
	ErrCodeRateLimited   ErrorCode = "RATE_LIMITED"
	ErrCodeTimeout       ErrorCode = "TIMEOUT"
	ErrCodeBulkheadFull  ErrorCode = "BULKHEAD_FULL"
	ErrCodeRetryExhausted ErrorCode = "RETRY_EXHAUSTED"
	ErrCodeInvalidPolicy ErrorCode = "INVALID_POLICY"
)

// ResilienceError is the base error type for all resilience errors.
type ResilienceError struct {
	Code          ErrorCode `json:"code"`
	Service       string    `json:"service"`
	Message       string    `json:"message"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	Cause         error     `json:"-"`
}

// Error implements the error interface.
func (e *ResilienceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)", e.Code, e.Service, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Service, e.Message)
}

// Unwrap returns the underlying cause.
func (e *ResilienceError) Unwrap() error {
	return e.Cause
}

// MarshalJSON implements json.Marshaler.
func (e *ResilienceError) MarshalJSON() ([]byte, error) {
	type Alias ResilienceError
	aux := &struct {
		*Alias
		CauseMsg string `json:"cause,omitempty"`
	}{
		Alias: (*Alias)(e),
	}
	if e.Cause != nil {
		aux.CauseMsg = e.Cause.Error()
	}
	return json.Marshal(aux)
}

// UnmarshalJSON implements json.Unmarshaler.
func (e *ResilienceError) UnmarshalJSON(data []byte) error {
	type Alias ResilienceError
	aux := &struct {
		*Alias
		CauseMsg string `json:"cause,omitempty"`
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.CauseMsg != "" {
		e.Cause = fmt.Errorf("%s", aux.CauseMsg)
	}
	return nil
}

// CircuitOpenError indicates the circuit breaker is open.
type CircuitOpenError struct {
	ResilienceError
	OpenedAt    time.Time     `json:"opened_at"`
	ResetAfter  time.Duration `json:"reset_after"`
	FailureRate float64       `json:"failure_rate"`
}

// NewCircuitOpenError creates a new CircuitOpenError.
func NewCircuitOpenError(service, correlationID string, openedAt time.Time, resetAfter time.Duration, failureRate float64) *CircuitOpenError {
	return &CircuitOpenError{
		ResilienceError: ResilienceError{
			Code:          ErrCodeCircuitOpen,
			Service:       service,
			Message:       "circuit breaker is open",
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		},
		OpenedAt:    openedAt,
		ResetAfter:  resetAfter,
		FailureRate: failureRate,
	}
}

// RateLimitError indicates rate limit exceeded.
type RateLimitError struct {
	ResilienceError
	Limit      int           `json:"limit"`
	Window     time.Duration `json:"window"`
	RetryAfter time.Duration `json:"retry_after"`
}

// NewRateLimitError creates a new RateLimitError.
func NewRateLimitError(service, correlationID string, limit int, window, retryAfter time.Duration) *RateLimitError {
	return &RateLimitError{
		ResilienceError: ResilienceError{
			Code:          ErrCodeRateLimited,
			Service:       service,
			Message:       "rate limit exceeded",
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		},
		Limit:      limit,
		Window:     window,
		RetryAfter: retryAfter,
	}
}

// MarshalJSON implements json.Marshaler for RateLimitError.
func (e *RateLimitError) MarshalJSON() ([]byte, error) {
	type Alias RateLimitError
	aux := &struct {
		Code          ErrorCode `json:"code"`
		Service       string    `json:"service"`
		Message       string    `json:"message"`
		CorrelationID string    `json:"correlation_id,omitempty"`
		Timestamp     time.Time `json:"timestamp"`
		Limit         int       `json:"limit"`
		Window        int64     `json:"window"`
		RetryAfter    int64     `json:"retry_after"`
	}{
		Code:          e.Code,
		Service:       e.Service,
		Message:       e.Message,
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
	ResilienceError
	Timeout time.Duration `json:"timeout"`
	Elapsed time.Duration `json:"elapsed"`
}

// NewTimeoutError creates a new TimeoutError.
func NewTimeoutError(service, correlationID string, timeout, elapsed time.Duration, cause error) *TimeoutError {
	return &TimeoutError{
		ResilienceError: ResilienceError{
			Code:          ErrCodeTimeout,
			Service:       service,
			Message:       "operation timed out",
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
			Cause:         cause,
		},
		Timeout: timeout,
		Elapsed: elapsed,
	}
}

// BulkheadFullError indicates bulkhead capacity exceeded.
type BulkheadFullError struct {
	ResilienceError
	MaxConcurrent int `json:"max_concurrent"`
	QueueSize     int `json:"queue_size"`
	CurrentLoad   int `json:"current_load"`
}

// NewBulkheadFullError creates a new BulkheadFullError.
func NewBulkheadFullError(service, correlationID string, maxConcurrent, queueSize, currentLoad int) *BulkheadFullError {
	return &BulkheadFullError{
		ResilienceError: ResilienceError{
			Code:          ErrCodeBulkheadFull,
			Service:       service,
			Message:       "bulkhead capacity exceeded",
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		},
		MaxConcurrent: maxConcurrent,
		QueueSize:     queueSize,
		CurrentLoad:   currentLoad,
	}
}

// RetryExhaustedError indicates all retries failed.
type RetryExhaustedError struct {
	ResilienceError
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
		ResilienceError: ResilienceError{
			Code:          ErrCodeRetryExhausted,
			Service:       service,
			Message:       "all retry attempts exhausted",
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
			Cause:         cause,
		},
		Attempts:   attempts,
		TotalTime:  totalTime,
		LastErrors: lastErrors,
	}
}

// InvalidPolicyError indicates invalid configuration.
type InvalidPolicyError struct {
	ResilienceError
	Field    string `json:"field"`
	Value    any    `json:"value"`
	Expected string `json:"expected"`
}

// NewInvalidPolicyError creates a new InvalidPolicyError.
func NewInvalidPolicyError(field string, value any, expected string) *InvalidPolicyError {
	return &InvalidPolicyError{
		ResilienceError: ResilienceError{
			Code:      ErrCodeInvalidPolicy,
			Message:   fmt.Sprintf("invalid policy: %s", field),
			Timestamp: time.Now(),
		},
		Field:    field,
		Value:    value,
		Expected: expected,
	}
}
