package cache

import "errors"

var (
	// ErrConnectionFailed indicates connection to cache-service failed.
	ErrConnectionFailed = errors.New("cache: connection failed")

	// ErrCircuitOpen indicates the circuit breaker is open.
	ErrCircuitOpen = errors.New("cache: circuit breaker open")

	// ErrTimeout indicates the operation timed out.
	ErrTimeout = errors.New("cache: operation timeout")

	// ErrNotFound indicates the key was not found.
	ErrNotFound = errors.New("cache: key not found")

	// ErrInvalidKey indicates an invalid cache key.
	ErrInvalidKey = errors.New("cache: invalid key")

	// ErrInvalidNamespace indicates an invalid namespace.
	ErrInvalidNamespace = errors.New("cache: invalid namespace")

	// ErrInvalidValue indicates an invalid value.
	ErrInvalidValue = errors.New("cache: invalid value")

	// ErrInvalidConfig indicates invalid configuration.
	ErrInvalidConfig = errors.New("cache: invalid configuration")

	// ErrServiceUnavailable indicates the cache service is unavailable.
	ErrServiceUnavailable = errors.New("cache: service unavailable")
)

// IsRetryable returns true if the error is retryable.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrConnectionFailed) ||
		errors.Is(err, ErrServiceUnavailable)
}

// IsNotFound returns true if the error indicates key not found.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsCircuitOpen returns true if the circuit breaker is open.
func IsCircuitOpen(err error) bool {
	return errors.Is(err, ErrCircuitOpen)
}
