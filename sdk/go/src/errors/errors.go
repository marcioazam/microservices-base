// Package errors provides unified error handling for the SDK.
package errors

import (
	"errors"
	"fmt"
)

// ErrorCode represents SDK error codes for programmatic handling.
type ErrorCode string

const (
	ErrCodeInvalidConfig ErrorCode = "INVALID_CONFIG"
	ErrCodeTokenExpired  ErrorCode = "TOKEN_EXPIRED"
	ErrCodeTokenInvalid  ErrorCode = "TOKEN_INVALID"
	ErrCodeTokenMissing  ErrorCode = "TOKEN_MISSING"
	ErrCodeTokenRefresh  ErrorCode = "TOKEN_REFRESH_FAILED"
	ErrCodeNetwork       ErrorCode = "NETWORK_ERROR"
	ErrCodeRateLimited   ErrorCode = "RATE_LIMITED"
	ErrCodeValidation    ErrorCode = "VALIDATION_FAILED"
	ErrCodeUnauthorized  ErrorCode = "UNAUTHORIZED"
	ErrCodeDPoPRequired  ErrorCode = "DPOP_REQUIRED"
	ErrCodeDPoPInvalid   ErrorCode = "DPOP_INVALID"
	ErrCodePKCEInvalid   ErrorCode = "PKCE_INVALID"
)

// AllErrorCodes returns all defined error codes for testing.
func AllErrorCodes() []ErrorCode {
	return []ErrorCode{
		ErrCodeInvalidConfig, ErrCodeTokenExpired, ErrCodeTokenInvalid,
		ErrCodeTokenMissing, ErrCodeTokenRefresh, ErrCodeNetwork,
		ErrCodeRateLimited, ErrCodeValidation, ErrCodeUnauthorized,
		ErrCodeDPoPRequired, ErrCodeDPoPInvalid, ErrCodePKCEInvalid,
	}
}

// SDKError is the unified error type for all SDK errors.
type SDKError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *SDKError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("authplatform: %s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("authplatform: %s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for error chain support.
func (e *SDKError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for errors.Is().
func (e *SDKError) Is(target error) bool {
	if t, ok := target.(*SDKError); ok {
		return e.Code == t.Code
	}
	return false
}

// NewError creates a new SDKError with the given code and message.
func NewError(code ErrorCode, message string) *SDKError {
	return &SDKError{Code: code, Message: message}
}

// WrapError creates a new SDKError wrapping an existing error.
func WrapError(code ErrorCode, message string, cause error) *SDKError {
	return &SDKError{Code: code, Message: message, Cause: cause}
}

// IsTokenExpired checks if the error is a token expired error.
func IsTokenExpired(err error) bool {
	return isSDKErrorCode(err, ErrCodeTokenExpired)
}

// IsTokenInvalid checks if the error is a token invalid error.
func IsTokenInvalid(err error) bool {
	return isSDKErrorCode(err, ErrCodeTokenInvalid)
}

// IsTokenMissing checks if the error is a token missing error.
func IsTokenMissing(err error) bool {
	return isSDKErrorCode(err, ErrCodeTokenMissing)
}

// IsRateLimited checks if the error is a rate limit error.
func IsRateLimited(err error) bool {
	return isSDKErrorCode(err, ErrCodeRateLimited)
}

// IsNetwork checks if the error is a network error.
func IsNetwork(err error) bool {
	return isSDKErrorCode(err, ErrCodeNetwork)
}

// IsValidation checks if the error is a validation error.
func IsValidation(err error) bool {
	return isSDKErrorCode(err, ErrCodeValidation)
}

// IsInvalidConfig checks if the error is an invalid config error.
func IsInvalidConfig(err error) bool {
	return isSDKErrorCode(err, ErrCodeInvalidConfig)
}

// IsUnauthorized checks if the error is an unauthorized error.
func IsUnauthorized(err error) bool {
	return isSDKErrorCode(err, ErrCodeUnauthorized)
}

// IsDPoPRequired checks if the error indicates DPoP is required.
func IsDPoPRequired(err error) bool {
	return isSDKErrorCode(err, ErrCodeDPoPRequired)
}

// IsDPoPInvalid checks if the error indicates DPoP is invalid.
func IsDPoPInvalid(err error) bool {
	return isSDKErrorCode(err, ErrCodeDPoPInvalid)
}

// IsPKCEInvalid checks if the error indicates PKCE is invalid.
func IsPKCEInvalid(err error) bool {
	return isSDKErrorCode(err, ErrCodePKCEInvalid)
}

// isSDKErrorCode checks if the error chain contains an SDKError with the given code.
func isSDKErrorCode(err error, code ErrorCode) bool {
	var sdkErr *SDKError
	if errors.As(err, &sdkErr) {
		return sdkErr.Code == code
	}
	return false
}

// GetCode extracts the error code from an SDKError, returns empty string if not SDKError.
func GetCode(err error) ErrorCode {
	var sdkErr *SDKError
	if errors.As(err, &sdkErr) {
		return sdkErr.Code
	}
	return ""
}
