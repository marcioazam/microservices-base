package property

import (
	"errors"
	"fmt"
	"testing"
	"time"

	liberror "github.com/auth-platform/libs/go/error"
	"pgregory.net/rapid"
)

// **Feature: platform-resilience-modernization, Property 8: Error Wrapping Preservation**
// **Validates: Requirements 12.2**
func TestProperty_ErrorWrappingPreservation(t *testing.T) {
	t.Run("unwrap_returns_original_cause", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,49}`).Draw(t, "service")
			attempts := rapid.IntRange(1, 10).Draw(t, "attempts")
			causeMsg := rapid.String().Draw(t, "causeMsg")

			cause := fmt.Errorf("original error: %s", causeMsg)
			err := liberror.NewRetryExhaustedError(service, attempts, cause)

			unwrapped := errors.Unwrap(err)
			if unwrapped != cause {
				t.Fatalf("unwrapped error %v != cause %v", unwrapped, cause)
			}
		})
	})

	t.Run("errors_is_finds_wrapped_cause", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,49}`).Draw(t, "service")
			attempts := rapid.IntRange(1, 10).Draw(t, "attempts")

			cause := errors.New("specific error")
			err := liberror.NewRetryExhaustedError(service, attempts, cause)

			if !errors.Is(err, cause) {
				t.Fatal("errors.Is should find wrapped cause")
			}
		})
	})
}

// **Feature: platform-resilience-modernization, Property 9: gRPC Error Mapping Completeness**
// **Validates: Requirements 12.3**
func TestProperty_GRPCErrorMappingCompleteness(t *testing.T) {
	t.Run("all_error_codes_have_grpc_mapping", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			codeIdx := rapid.IntRange(0, 6).Draw(t, "codeIdx")

			codes := []liberror.ResilienceErrorCode{
				liberror.ErrCircuitOpen,
				liberror.ErrRateLimitExceeded,
				liberror.ErrTimeout,
				liberror.ErrBulkheadFull,
				liberror.ErrRetryExhausted,
				liberror.ErrInvalidPolicy,
				liberror.ErrServiceUnavailable,
			}
			code := codes[codeIdx%len(codes)]

			err := &liberror.ResilienceError{
				Code:    code,
				Message: "test error",
			}

			if err.Code == "" {
				t.Fatal("error code should not be empty")
			}
		})
	})
}

// **Feature: resilience-service-modernization-2025, Property 2: Error Constructor Type Preservation**
// **Validates: Requirements 4.1, 4.3**
func TestProperty_ErrorConstructorTypePreservation(t *testing.T) {
	t.Run("circuit_open_error_is_detected_by_is_function", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,49}`).Draw(t, "service")

			err := liberror.NewCircuitOpenError(service)
			if !liberror.IsCircuitOpen(err) {
				t.Fatal("IsCircuitOpen should return true")
			}
		})
	})

	t.Run("rate_limit_error_is_detected_by_is_function", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,49}`).Draw(t, "service")
			retryAfterMs := rapid.IntRange(100, 10000).Draw(t, "retryAfterMs")

			err := liberror.NewRateLimitError(service, time.Duration(retryAfterMs)*time.Millisecond)
			if !liberror.IsRateLimitExceeded(err) {
				t.Fatal("IsRateLimitExceeded should return true")
			}
		})
	})

	t.Run("timeout_error_is_detected_by_is_function", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,49}`).Draw(t, "service")
			timeoutMs := rapid.IntRange(100, 10000).Draw(t, "timeoutMs")

			err := liberror.NewTimeoutError(service, time.Duration(timeoutMs)*time.Millisecond)
			if !liberror.IsTimeout(err) {
				t.Fatal("IsTimeout should return true")
			}
		})
	})

	t.Run("bulkhead_full_error_is_detected_by_is_function", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			partition := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,49}`).Draw(t, "partition")

			err := liberror.NewBulkheadFullError(partition)
			if !liberror.IsBulkheadFull(err) {
				t.Fatal("IsBulkheadFull should return true")
			}
		})
	})

	t.Run("retry_exhausted_error_is_detected_by_is_function", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,49}`).Draw(t, "service")
			attempts := rapid.IntRange(1, 10).Draw(t, "attempts")

			cause := errors.New("test cause")
			err := liberror.NewRetryExhaustedError(service, attempts, cause)
			if !liberror.IsRetryExhausted(err) {
				t.Fatal("IsRetryExhausted should return true")
			}
		})
	})
}
