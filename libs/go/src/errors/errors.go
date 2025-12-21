// Package errors provides typed error handling with HTTP/gRPC mapping.
// Supports Go 1.25+ features including generic error type assertions.
package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// AsType is a generic error type assertion using Go 1.25+ errors package.
// Returns the error as type T and true if the error chain contains type T.
func AsType[T error](err error) (T, bool) {
	var target T
	if errors.As(err, &target) {
		return target, true
	}
	return target, false
}

// Must panics if err is not nil, otherwise returns value.
func Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

// ErrorCode represents application error categories.
type ErrorCode string

// Error codes for all application error categories.
const (
	// Client errors (4xx)
	ErrCodeValidation     ErrorCode = "VALIDATION_ERROR"
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden      ErrorCode = "FORBIDDEN"
	ErrCodeConflict       ErrorCode = "CONFLICT"
	ErrCodeBadRequest     ErrorCode = "BAD_REQUEST"
	ErrCodeTooManyReqs    ErrorCode = "TOO_MANY_REQUESTS"
	ErrCodePrecondition   ErrorCode = "PRECONDITION_FAILED"

	// Server errors (5xx)
	ErrCodeInternal       ErrorCode = "INTERNAL_ERROR"
	ErrCodeUnavailable    ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeTimeout        ErrorCode = "TIMEOUT"
	ErrCodeNotImplemented ErrorCode = "NOT_IMPLEMENTED"
	ErrCodeDependency     ErrorCode = "DEPENDENCY_ERROR"

	// Business errors
	ErrCodeBusinessRule   ErrorCode = "BUSINESS_RULE_VIOLATION"
	ErrCodeInvalidState   ErrorCode = "INVALID_STATE"
)

// AppError is the standard application error type.
type AppError struct {
	Code          ErrorCode         `json:"code"`
	Message       string            `json:"message"`
	Details       map[string]any    `json:"details,omitempty"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	Timestamp     time.Time         `json:"timestamp"`
	cause         error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/As.
func (e *AppError) Unwrap() error {
	return e.cause
}

// WithCause sets the underlying cause.
func (e *AppError) WithCause(cause error) *AppError {
	e.cause = cause
	return e
}

// WithDetail adds a detail to the error.
func (e *AppError) WithDetail(key string, value any) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// WithCorrelationID sets the correlation ID.
func (e *AppError) WithCorrelationID(id string) *AppError {
	e.CorrelationID = id
	return e
}

// HTTPStatus returns the HTTP status code for this error.
func (e *AppError) HTTPStatus() int {
	return httpStatusMap[e.Code]
}

// GRPCCode returns the gRPC status code for this error.
func (e *AppError) GRPCCode() int {
	return grpcCodeMap[e.Code]
}

// Is checks if the error matches a target error code.
func (e *AppError) Is(target error) bool {
	if t, ok := target.(*AppError); ok {
		return e.Code == t.Code
	}
	return errors.Is(e.cause, target)
}

// As implements errors.As for type assertion.
func (e *AppError) As(target any) bool {
	if t, ok := target.(**AppError); ok {
		*t = e
		return true
	}
	return false
}

// MarshalJSON implements json.Marshaler.
func (e *AppError) MarshalJSON() ([]byte, error) {
	type Alias AppError
	aux := &struct {
		*Alias
		Cause string `json:"cause,omitempty"`
	}{Alias: (*Alias)(e)}
	if e.cause != nil {
		aux.Cause = e.cause.Error()
	}
	return json.Marshal(aux)
}

// HTTP status code mapping.
var httpStatusMap = map[ErrorCode]int{
	ErrCodeValidation:     http.StatusBadRequest,
	ErrCodeNotFound:       http.StatusNotFound,
	ErrCodeUnauthorized:   http.StatusUnauthorized,
	ErrCodeForbidden:      http.StatusForbidden,
	ErrCodeConflict:       http.StatusConflict,
	ErrCodeBadRequest:     http.StatusBadRequest,
	ErrCodeTooManyReqs:    http.StatusTooManyRequests,
	ErrCodePrecondition:   http.StatusPreconditionFailed,
	ErrCodeInternal:       http.StatusInternalServerError,
	ErrCodeUnavailable:    http.StatusServiceUnavailable,
	ErrCodeTimeout:        http.StatusGatewayTimeout,
	ErrCodeNotImplemented: http.StatusNotImplemented,
	ErrCodeDependency:     http.StatusBadGateway,
	ErrCodeBusinessRule:   http.StatusUnprocessableEntity,
	ErrCodeInvalidState:   http.StatusConflict,
}

// gRPC status code mapping (codes from google.golang.org/grpc/codes).
var grpcCodeMap = map[ErrorCode]int{
	ErrCodeValidation:     3,  // InvalidArgument
	ErrCodeNotFound:       5,  // NotFound
	ErrCodeUnauthorized:   16, // Unauthenticated
	ErrCodeForbidden:      7,  // PermissionDenied
	ErrCodeConflict:       6,  // AlreadyExists
	ErrCodeBadRequest:     3,  // InvalidArgument
	ErrCodeTooManyReqs:    8,  // ResourceExhausted
	ErrCodePrecondition:   9,  // FailedPrecondition
	ErrCodeInternal:       13, // Internal
	ErrCodeUnavailable:    14, // Unavailable
	ErrCodeTimeout:        4,  // DeadlineExceeded
	ErrCodeNotImplemented: 12, // Unimplemented
	ErrCodeDependency:     14, // Unavailable
	ErrCodeBusinessRule:   9,  // FailedPrecondition
	ErrCodeInvalidState:   9,  // FailedPrecondition
}
