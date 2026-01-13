package cache_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/auth-platform/cache-service/internal/cache"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestErrorToHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"not found", cache.ErrNotFound, http.StatusNotFound},
		{"invalid key", cache.ErrInvalidKeyError, http.StatusBadRequest},
		{"invalid value", cache.ErrInvalidValueError, http.StatusBadRequest},
		{"invalid namespace", cache.ErrInvalidNamespace, http.StatusBadRequest},
		{"ttl invalid", cache.ErrInvalidTTL, http.StatusBadRequest},
		{"ttl too short", cache.ErrTTLTooShort, http.StatusBadRequest},
		{"ttl too long", cache.ErrTTLTooLong, http.StatusBadRequest},
		{"redis unavailable", cache.ErrRedisDown, http.StatusServiceUnavailable},
		{"circuit breaker open", cache.ErrCircuitBreakerOpen, http.StatusServiceUnavailable},
		{"encryption failed", cache.ErrEncryptFailed, http.StatusInternalServerError},
		{"decryption failed", cache.ErrDecryptFailed, http.StatusInternalServerError},
		{"unknown error", errors.New("unknown"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := cache.ToHTTPStatus(tt.err)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestErrorToGRPCStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected codes.Code
	}{
		{"not found", cache.ErrNotFound, codes.NotFound},
		{"invalid key", cache.ErrInvalidKeyError, codes.InvalidArgument},
		{"invalid value", cache.ErrInvalidValueError, codes.InvalidArgument},
		{"invalid namespace", cache.ErrInvalidNamespace, codes.InvalidArgument},
		{"redis unavailable", cache.ErrRedisDown, codes.Unavailable},
		{"circuit breaker open", cache.ErrCircuitBreakerOpen, codes.Unavailable},
		{"encryption failed", cache.ErrEncryptFailed, codes.Internal},
		{"unknown error", errors.New("unknown"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := cache.ToGRPCCode(tt.err)
			assert.Equal(t, tt.expected, code)
		})
	}
}

func TestWrapError(t *testing.T) {
	cause := errors.New("connection refused")
	wrapped := cache.WrapError(cache.ErrRedisDown, "failed to connect", cause)

	assert.True(t, errors.Is(wrapped, cache.ErrRedisDown))
	assert.Contains(t, wrapped.Error(), "failed to connect")
	assert.Contains(t, wrapped.Error(), "connection refused")
}

func TestIsCircuitOpen(t *testing.T) {
	assert.True(t, cache.IsCircuitOpen(cache.ErrCircuitBreakerOpen))
	assert.False(t, cache.IsCircuitOpen(cache.ErrNotFound))
	assert.False(t, cache.IsCircuitOpen(errors.New("other")))
}

func TestWrappedErrorHTTPStatus(t *testing.T) {
	cause := errors.New("timeout")
	wrapped := cache.WrapError(cache.ErrRedisDown, "redis timeout", cause)

	status := cache.ToHTTPStatus(wrapped)
	assert.Equal(t, http.StatusServiceUnavailable, status)
}

func TestWrappedErrorGRPCCode(t *testing.T) {
	cause := errors.New("timeout")
	wrapped := cache.WrapError(cache.ErrNotFound, "key not found", cause)

	code := cache.ToGRPCCode(wrapped)
	assert.Equal(t, codes.NotFound, code)
}

func TestNilError(t *testing.T) {
	assert.Equal(t, http.StatusOK, cache.ToHTTPStatus(nil))
	assert.Equal(t, codes.OK, cache.ToGRPCCode(nil))
}
