package error

import (
	"errors"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func defaultTestParameters() *gopter.TestParameters {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	return params
}

// **Feature: resilience-lib-extraction, Property 4: Error Wrapping Preservation**
// **Validates: Requirements 3.2, 3.5**
func TestProperty_ErrorWrappingPreservation(t *testing.T) {
	params := defaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("unwrap_returns_original_cause", prop.ForAll(
		func(causeMsg string) bool {
			cause := errors.New(causeMsg)
			err := NewRetryExhaustedError("test-service", 3, cause)
			
			unwrapped := err.Unwrap()
			return unwrapped == cause
		},
		gen.AnyString(),
	))

	props.Property("errors_as_works_with_resilience_error", prop.ForAll(
		func(service string) bool {
			err := NewCircuitOpenError(service)
			
			var resErr *ResilienceError
			return As(err, &resErr) && resErr.Code == ErrCircuitOpen
		},
		gen.AnyString(),
	))

	props.Property("error_is_matches_same_code", prop.ForAll(
		func(_ int) bool {
			err1 := &ResilienceError{Code: ErrCircuitOpen}
			err2 := &ResilienceError{Code: ErrCircuitOpen}
			err3 := &ResilienceError{Code: ErrTimeout}
			
			return err1.Is(err2) && !err1.Is(err3)
		},
		gen.IntRange(1, 100),
	))

	props.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 5: gRPC Error Mapping Completeness**
// **Validates: Requirements 3.3**
func TestProperty_GRPCErrorMappingCompleteness(t *testing.T) {
	params := defaultTestParameters()
	props := gopter.NewProperties(params)

	allCodes := []ResilienceErrorCode{
		ErrCircuitOpen,
		ErrRateLimitExceeded,
		ErrTimeout,
		ErrBulkheadFull,
		ErrRetryExhausted,
		ErrInvalidPolicy,
		ErrServiceUnavailable,
		ErrValidation,
	}

	props.Property("all_error_codes_have_grpc_mapping", prop.ForAll(
		func(idx int) bool {
			code := allCodes[idx%len(allCodes)]
			grpcCode := ToGRPCCode(code)
			// Should not return Unknown (0) for known codes
			return grpcCode != 0
		},
		gen.IntRange(0, len(allCodes)-1),
	))

	props.Property("to_grpc_error_returns_valid_status", prop.ForAll(
		func(service string, retryAfter int64) bool {
			err := NewRateLimitError(service, time.Duration(retryAfter)*time.Millisecond)
			grpcErr := ToGRPCError(err)
			return grpcErr != nil
		},
		gen.AnyString(),
		gen.Int64Range(0, 10000),
	))

	props.Property("nil_error_returns_nil_grpc_error", prop.ForAll(
		func(_ int) bool {
			return ToGRPCError(nil) == nil
		},
		gen.IntRange(1, 100),
	))

	props.TestingRun(t)
}
