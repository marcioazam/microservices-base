package cache

import (
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorCode represents cache-specific error codes.
type ErrorCode int

const (
	// ErrUnknown represents an unknown error.
	ErrUnknown ErrorCode = iota
	// ErrKeyNotFound indicates the key was not found in cache.
	ErrKeyNotFound
	// ErrInvalidKey indicates the key is invalid.
	ErrInvalidKey
	// ErrInvalidValue indicates the value is invalid.
	ErrInvalidValue
	// ErrTTLInvalid indicates the TTL is invalid.
	ErrTTLInvalid
	// ErrRedisUnavailable indicates Redis is not available.
	ErrRedisUnavailable
	// ErrCircuitOpen indicates the circuit breaker is open.
	ErrCircuitOpen
	// ErrEncryptionFailed indicates encryption failed.
	ErrEncryptionFailed
	// ErrDecryptionFailed indicates decryption failed.
	ErrDecryptionFailed
	// ErrUnauthorized indicates the request is not authorized.
	ErrUnauthorized
	// ErrForbidden indicates the request is forbidden.
	ErrForbidden
	// ErrNamespaceInvalid indicates the namespace is invalid.
	ErrNamespaceInvalid
	// ErrBrokerUnavailable indicates the message broker is not available.
	ErrBrokerUnavailable
)

// String returns the string representation of ErrorCode.
func (c ErrorCode) String() string {
	switch c {
	case ErrUnknown:
		return "unknown"
	case ErrKeyNotFound:
		return "key_not_found"
	case ErrInvalidKey:
		return "invalid_key"
	case ErrInvalidValue:
		return "invalid_value"
	case ErrTTLInvalid:
		return "ttl_invalid"
	case ErrRedisUnavailable:
		return "redis_unavailable"
	case ErrCircuitOpen:
		return "circuit_open"
	case ErrEncryptionFailed:
		return "encryption_failed"
	case ErrDecryptionFailed:
		return "decryption_failed"
	case ErrUnauthorized:
		return "unauthorized"
	case ErrForbidden:
		return "forbidden"
	case ErrNamespaceInvalid:
		return "namespace_invalid"
	case ErrBrokerUnavailable:
		return "broker_unavailable"
	default:
		return "unknown"
	}
}

// Error represents a cache-specific error.
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target.
func (e *Error) Is(target error) bool {
	var cacheErr *Error
	if errors.As(target, &cacheErr) {
		return e.Code == cacheErr.Code
	}
	return false
}

// ToHTTPStatus maps error code to HTTP status.
func (e *Error) ToHTTPStatus() int {
	switch e.Code {
	case ErrKeyNotFound:
		return http.StatusNotFound
	case ErrInvalidKey, ErrInvalidValue, ErrTTLInvalid, ErrNamespaceInvalid:
		return http.StatusBadRequest
	case ErrUnauthorized:
		return http.StatusUnauthorized
	case ErrForbidden:
		return http.StatusForbidden
	case ErrRedisUnavailable, ErrCircuitOpen, ErrBrokerUnavailable:
		return http.StatusServiceUnavailable
	case ErrEncryptionFailed, ErrDecryptionFailed, ErrUnknown:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// ToGRPCStatus maps error code to gRPC status.
func (e *Error) ToGRPCStatus() *status.Status {
	switch e.Code {
	case ErrKeyNotFound:
		return status.New(codes.NotFound, e.Message)
	case ErrInvalidKey, ErrInvalidValue, ErrTTLInvalid, ErrNamespaceInvalid:
		return status.New(codes.InvalidArgument, e.Message)
	case ErrUnauthorized:
		return status.New(codes.Unauthenticated, e.Message)
	case ErrForbidden:
		return status.New(codes.PermissionDenied, e.Message)
	case ErrRedisUnavailable, ErrCircuitOpen, ErrBrokerUnavailable:
		return status.New(codes.Unavailable, e.Message)
	case ErrEncryptionFailed, ErrDecryptionFailed, ErrUnknown:
		return status.New(codes.Internal, e.Message)
	default:
		return status.New(codes.Internal, e.Message)
	}
}

// NewError creates a new cache error.
func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// WrapError wraps an existing error with a base error for error chain.
func WrapError(base *Error, message string, cause error) *Error {
	return &Error{
		Code:    base.Code,
		Message: message,
		Cause:   cause,
	}
}

// Predefined errors for common cases.
var (
	ErrNotFound           = NewError(ErrKeyNotFound, "key not found")
	ErrInvalidKeyError    = NewError(ErrInvalidKey, "key is invalid")
	ErrInvalidValueError  = NewError(ErrInvalidValue, "value is invalid")
	ErrInvalidTTL         = NewError(ErrTTLInvalid, "TTL must be positive")
	ErrRedisDown          = NewError(ErrRedisUnavailable, "redis is unavailable")
	ErrCircuitBreakerOpen = NewError(ErrCircuitOpen, "circuit breaker is open")
	ErrInvalidNamespace   = NewError(ErrNamespaceInvalid, "namespace is invalid")
	ErrBrokerDown         = NewError(ErrBrokerUnavailable, "message broker is unavailable")
	ErrEncryptFailed      = NewError(ErrEncryptionFailed, "encryption failed")
	ErrDecryptFailed      = NewError(ErrDecryptionFailed, "decryption failed")
)

// IsNotFound checks if the error is a key not found error.
func IsNotFound(err error) bool {
	var cacheErr *Error
	if errors.As(err, &cacheErr) {
		return cacheErr.Code == ErrKeyNotFound
	}
	return false
}

// IsRedisUnavailable checks if the error is a Redis unavailable error.
func IsRedisUnavailable(err error) bool {
	var cacheErr *Error
	if errors.As(err, &cacheErr) {
		return cacheErr.Code == ErrRedisUnavailable
	}
	return false
}

// IsCircuitOpen checks if the error is a circuit open error.
func IsCircuitOpen(err error) bool {
	var cacheErr *Error
	if errors.As(err, &cacheErr) {
		return cacheErr.Code == ErrCircuitOpen
	}
	// Also check for circuitbreaker.ErrOpen
	return errors.Is(err, ErrCircuitBreakerOpen)
}

// ToHTTPStatusFromError converts any error to HTTP status code.
func ToHTTPStatusFromError(err error) int {
	var cacheErr *Error
	if errors.As(err, &cacheErr) {
		return cacheErr.ToHTTPStatus()
	}
	return http.StatusInternalServerError
}

// ToGRPCStatusFromError converts any error to gRPC status.
func ToGRPCStatusFromError(err error) *status.Status {
	var cacheErr *Error
	if errors.As(err, &cacheErr) {
		return cacheErr.ToGRPCStatus()
	}
	return status.New(codes.Internal, err.Error())
}

// ToHTTPStatus is a convenience function for error to HTTP status mapping.
func ToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	return ToHTTPStatusFromError(err)
}

// ToGRPCCode is a convenience function for error to gRPC code mapping.
func ToGRPCCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	var cacheErr *Error
	if errors.As(err, &cacheErr) {
		return cacheErr.ToGRPCStatus().Code()
	}
	return codes.Internal
}

// Additional predefined errors for TTL validation.
var (
	ErrTTLTooShort = NewError(ErrTTLInvalid, "TTL is below minimum")
	ErrTTLTooLong  = NewError(ErrTTLInvalid, "TTL exceeds maximum")
)
