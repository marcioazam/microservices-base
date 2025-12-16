package authplatform

import "errors"

// Sentinel errors for the Auth Platform SDK.
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
)

// IsTokenExpired checks if the error is a token expired error.
func IsTokenExpired(err error) bool {
	return errors.Is(err, ErrTokenExpired)
}

// IsRateLimited checks if the error is a rate limit error.
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}

// IsNetwork checks if the error is a network error.
func IsNetwork(err error) bool {
	return errors.Is(err, ErrNetwork)
}

// IsValidation checks if the error is a validation error.
func IsValidation(err error) bool {
	return errors.Is(err, ErrValidation)
}
