package authplatform

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorCode represents SDK error codes for programmatic handling.
type ErrorCode string

const (
	// ErrCodeInvalidConfig indicates invalid client configuration.
	ErrCodeInvalidConfig ErrorCode = "INVALID_CONFIG"
	// ErrCodeTokenExpired indicates the access token has expired.
	ErrCodeTokenExpired ErrorCode = "TOKEN_EXPIRED"
	// ErrCodeTokenInvalid indicates the token is invalid.
	ErrCodeTokenInvalid ErrorCode = "TOKEN_INVALID"
	// ErrCodeTokenRefresh indicates token refresh failed.
	ErrCodeTokenRefresh ErrorCode = "TOKEN_REFRESH_FAILED"
	// ErrCodeNetwork indicates a network error occurred.
	ErrCodeNetwork ErrorCode = "NETWORK_ERROR"
	// ErrCodeRateLimited indicates rate limit was exceeded.
	ErrCodeRateLimited ErrorCode = "RATE_LIMITED"
	// ErrCodeValidation indicates token validation failed.
	ErrCodeValidation ErrorCode = "VALIDATION_FAILED"
	// ErrCodeUnauthorized indicates the request was unauthorized.
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	// ErrCodeTokenMissing indicates the token is missing.
	ErrCodeTokenMissing ErrorCode = "TOKEN_MISSING"
	// ErrCodeDPoPRequired indicates DPoP proof is required.
	ErrCodeDPoPRequired ErrorCode = "DPOP_REQUIRED"
	// ErrCodeDPoPInvalid indicates DPoP proof is invalid.
	ErrCodeDPoPInvalid ErrorCode = "DPOP_INVALID"
	// ErrCodePKCEInvalid indicates PKCE parameters are invalid.
	ErrCodePKCEInvalid ErrorCode = "PKCE_INVALID"
)

// SDKError is the structured error type for all SDK errors.
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

// Sentinel errors for backward compatibility.
var (
	// ErrInvalidConfig indicates invalid client configuration.
	ErrInvalidConfig = errors.New("authplatform: invalid configuration")
	// ErrTokenExpired indicates the access token has expired.
	ErrTokenExpired = errors.New("authplatform: token expired")
	// ErrTokenRefresh indicates token refresh failed.
	ErrTokenRefresh = errors.New("authplatform: token refresh failed")
	// ErrNetwork indicates a network error occurred.
	ErrNetwork = errors.New("authplatform: network error")
	// ErrRateLimited indicates rate limit was exceeded.
	ErrRateLimited = errors.New("authplatform: rate limited")
	// ErrValidation indicates token validation failed.
	ErrValidation = errors.New("authplatform: validation failed")
	// ErrUnauthorized indicates the request was unauthorized.
	ErrUnauthorized = errors.New("authplatform: unauthorized")
	// ErrDPoPRequired indicates DPoP proof is required.
	ErrDPoPRequired = errors.New("authplatform: DPoP proof required")
	// ErrDPoPInvalid indicates DPoP proof is invalid.
	ErrDPoPInvalid = errors.New("authplatform: DPoP proof invalid")
	// ErrPKCEInvalid indicates PKCE parameters are invalid.
	ErrPKCEInvalid = errors.New("authplatform: PKCE invalid")
)

// IsTokenExpired checks if the error is a token expired error.
func IsTokenExpired(err error) bool {
	return errors.Is(err, ErrTokenExpired) || isSDKErrorCode(err, ErrCodeTokenExpired)
}

// IsRateLimited checks if the error is a rate limit error.
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited) || isSDKErrorCode(err, ErrCodeRateLimited)
}

// IsNetwork checks if the error is a network error.
func IsNetwork(err error) bool {
	return errors.Is(err, ErrNetwork) || isSDKErrorCode(err, ErrCodeNetwork)
}

// IsValidation checks if the error is a validation error.
func IsValidation(err error) bool {
	return errors.Is(err, ErrValidation) || isSDKErrorCode(err, ErrCodeValidation)
}

// IsInvalidConfig checks if the error is an invalid config error.
func IsInvalidConfig(err error) bool {
	return errors.Is(err, ErrInvalidConfig) || isSDKErrorCode(err, ErrCodeInvalidConfig)
}

// IsUnauthorized checks if the error is an unauthorized error.
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized) || isSDKErrorCode(err, ErrCodeUnauthorized)
}

// IsDPoPRequired checks if the error indicates DPoP is required.
func IsDPoPRequired(err error) bool {
	return errors.Is(err, ErrDPoPRequired) || isSDKErrorCode(err, ErrCodeDPoPRequired)
}

// IsDPoPInvalid checks if the error indicates DPoP is invalid.
func IsDPoPInvalid(err error) bool {
	return errors.Is(err, ErrDPoPInvalid) || isSDKErrorCode(err, ErrCodeDPoPInvalid)
}

// IsPKCEInvalid checks if the error indicates PKCE is invalid.
func IsPKCEInvalid(err error) bool {
	return errors.Is(err, ErrPKCEInvalid) || isSDKErrorCode(err, ErrCodePKCEInvalid)
}

// isSDKErrorCode checks if the error chain contains an SDKError with the given code.
func isSDKErrorCode(err error, code ErrorCode) bool {
	var sdkErr *SDKError
	if errors.As(err, &sdkErr) {
		return sdkErr.Code == code
	}
	return false
}

// sensitivePatterns contains patterns that should not appear in error messages.
var sensitivePatterns = []string{
	"Bearer ",
	"DPoP ",
	"secret",
	"password",
	"credential",
	"eyJ", // JWT header prefix (base64)
}

// ContainsSensitiveData checks if a string contains sensitive data patterns.
func ContainsSensitiveData(s string) bool {
	lower := strings.ToLower(s)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// SanitizeError removes sensitive data from error messages.
func SanitizeError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if ContainsSensitiveData(msg) {
		return errors.New("authplatform: error occurred (details redacted)")
	}
	return err
}
