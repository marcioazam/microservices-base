// Package errors provides consistent error handling for IAM Policy Service.
package errors

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Error codes for IAM Policy Service.
const (
	CodeInvalidInput     = "INVALID_INPUT"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeForbidden        = "FORBIDDEN"
	CodeNotFound         = "NOT_FOUND"
	CodeConflict         = "CONFLICT"
	CodeInternal         = "INTERNAL"
	CodeUnavailable      = "UNAVAILABLE"
	CodeTimeout          = "TIMEOUT"
	CodeRateLimited      = "RATE_LIMITED"
	CodePolicyEvalFailed = "POLICY_EVAL_FAILED"
	CodeCacheError       = "CACHE_ERROR"
)

// ServiceError represents a service-level error.
type ServiceError struct {
	Code          string
	Message       string
	Details       map[string]interface{}
	CorrelationID string
	Cause         error
}

// Error implements the error interface.
func (e *ServiceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// WithCorrelationID adds a correlation ID to the error.
func (e *ServiceError) WithCorrelationID(id string) *ServiceError {
	e.CorrelationID = id
	return e
}

// WithDetail adds a detail to the error.
func (e *ServiceError) WithDetail(key string, value interface{}) *ServiceError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// New creates a new service error.
func New(code, message string) *ServiceError {
	return &ServiceError{Code: code, Message: message}
}

// Wrap wraps an error with a service error.
func Wrap(err error, code, message string) *ServiceError {
	return &ServiceError{Code: code, Message: message, Cause: err}
}

// InvalidInput creates an invalid input error.
func InvalidInput(message string) *ServiceError {
	return New(CodeInvalidInput, message)
}

// Unauthorized creates an unauthorized error.
func Unauthorized(message string) *ServiceError {
	return New(CodeUnauthorized, message)
}

// Forbidden creates a forbidden error.
func Forbidden(message string) *ServiceError {
	return New(CodeForbidden, message)
}

// NotFound creates a not found error.
func NotFound(message string) *ServiceError {
	return New(CodeNotFound, message)
}

// Internal creates an internal error.
func Internal(message string) *ServiceError {
	return New(CodeInternal, message)
}

// Unavailable creates an unavailable error.
func Unavailable(message string) *ServiceError {
	return New(CodeUnavailable, message)
}

// RateLimited creates a rate limited error.
func RateLimited(message string) *ServiceError {
	return New(CodeRateLimited, message)
}

// ToGRPCStatus converts a service error to gRPC status.
func ToGRPCStatus(err error) *status.Status {
	var svcErr *ServiceError
	if errors.As(err, &svcErr) {
		code := codeToGRPC(svcErr.Code)
		return status.New(code, svcErr.Message)
	}
	return status.New(codes.Internal, "internal error")
}

// ToGRPCError converts a service error to gRPC error.
func ToGRPCError(err error) error {
	return ToGRPCStatus(err).Err()
}

func codeToGRPC(code string) codes.Code {
	switch code {
	case CodeInvalidInput:
		return codes.InvalidArgument
	case CodeUnauthorized:
		return codes.Unauthenticated
	case CodeForbidden:
		return codes.PermissionDenied
	case CodeNotFound:
		return codes.NotFound
	case CodeConflict:
		return codes.AlreadyExists
	case CodeUnavailable:
		return codes.Unavailable
	case CodeTimeout:
		return codes.DeadlineExceeded
	case CodeRateLimited:
		return codes.ResourceExhausted
	default:
		return codes.Internal
	}
}

// IsCode checks if an error has a specific code.
func IsCode(err error, code string) bool {
	var svcErr *ServiceError
	if errors.As(err, &svcErr) {
		return svcErr.Code == code
	}
	return false
}

// GetCorrelationID extracts correlation ID from error.
func GetCorrelationID(err error) string {
	var svcErr *ServiceError
	if errors.As(err, &svcErr) {
		return svcErr.CorrelationID
	}
	return ""
}
