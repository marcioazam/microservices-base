package property_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/auth-platform/cache-service/internal/cache"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"pgregory.net/rapid"
)

// Property 4: Error code mapping consistency
func TestProperty_ErrorCodeMappingConsistency(t *testing.T) {
	knownErrors := []error{
		cache.ErrNotFound,
		cache.ErrInvalidKeyError,
		cache.ErrInvalidValueError,
		cache.ErrInvalidNamespace,
		cache.ErrInvalidTTL,
		cache.ErrTTLTooShort,
		cache.ErrTTLTooLong,
		cache.ErrRedisDown,
		cache.ErrCircuitBreakerOpen,
		cache.ErrEncryptFailed,
		cache.ErrDecryptFailed,
	}

	rapid.Check(t, func(t *rapid.T) {
		errIdx := rapid.IntRange(0, len(knownErrors)-1).Draw(t, "errIdx")
		err := knownErrors[errIdx]

		httpStatus := cache.ToHTTPStatus(err)
		grpcCode := cache.ToGRPCCode(err)

		// HTTP status should be valid
		assert.GreaterOrEqual(t, httpStatus, 100)
		assert.LessOrEqual(t, httpStatus, 599)

		// gRPC code should be valid
		assert.GreaterOrEqual(t, int(grpcCode), 0)
		assert.LessOrEqual(t, int(grpcCode), 16)

		// Mapping should be deterministic (same error = same codes)
		assert.Equal(t, httpStatus, cache.ToHTTPStatus(err))
		assert.Equal(t, grpcCode, cache.ToGRPCCode(err))
	})
}

// Property: HTTP and gRPC codes are semantically aligned
func TestProperty_HTTPGRPCSemanticAlignment(t *testing.T) {
	// 4xx HTTP errors should map to client-side gRPC codes
	clientErrors := []error{
		cache.ErrNotFound,
		cache.ErrInvalidKeyError,
		cache.ErrInvalidValueError,
		cache.ErrInvalidNamespace,
	}

	for _, err := range clientErrors {
		httpStatus := cache.ToHTTPStatus(err)
		grpcCode := cache.ToGRPCCode(err)

		// 4xx HTTP should map to InvalidArgument, NotFound, etc.
		if httpStatus >= 400 && httpStatus < 500 {
			validClientCodes := []codes.Code{
				codes.InvalidArgument,
				codes.NotFound,
				codes.PermissionDenied,
				codes.Unauthenticated,
				codes.FailedPrecondition,
			}
			assert.Contains(t, validClientCodes, grpcCode,
				"4xx HTTP should map to client-side gRPC code for %v", err)
		}
	}

	// 5xx HTTP errors should map to server-side gRPC codes
	serverErrors := []error{
		cache.ErrRedisDown,
		cache.ErrCircuitBreakerOpen,
		cache.ErrEncryptFailed,
	}

	for _, err := range serverErrors {
		httpStatus := cache.ToHTTPStatus(err)
		grpcCode := cache.ToGRPCCode(err)

		if httpStatus >= 500 {
			validServerCodes := []codes.Code{
				codes.Internal,
				codes.Unavailable,
				codes.Unknown,
				codes.DataLoss,
			}
			assert.Contains(t, validServerCodes, grpcCode,
				"5xx HTTP should map to server-side gRPC code for %v", err)
		}
	}
}

// Property: Wrapped errors preserve base error mapping
func TestProperty_WrappedErrorsPreserveMapping(t *testing.T) {
	baseErrors := []*cache.Error{
		cache.ErrNotFound,
		cache.ErrRedisDown,
		cache.ErrInvalidKeyError,
	}

	rapid.Check(t, func(t *rapid.T) {
		errIdx := rapid.IntRange(0, len(baseErrors)-1).Draw(t, "errIdx")
		baseErr := baseErrors[errIdx]
		message := rapid.String().Draw(t, "message")

		wrapped := cache.WrapError(baseErr, message, errors.New("cause"))

		// Wrapped error should have same HTTP status as base
		assert.Equal(t, cache.ToHTTPStatus(baseErr), cache.ToHTTPStatus(wrapped))

		// Wrapped error should have same gRPC code as base
		assert.Equal(t, cache.ToGRPCCode(baseErr), cache.ToGRPCCode(wrapped))
	})
}

// Property: Unknown errors map to 500/Internal
func TestProperty_UnknownErrorsMapToInternal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		message := rapid.String().Draw(t, "message")
		unknownErr := errors.New(message)

		httpStatus := cache.ToHTTPStatus(unknownErr)
		grpcCode := cache.ToGRPCCode(unknownErr)

		assert.Equal(t, http.StatusInternalServerError, httpStatus)
		assert.Equal(t, codes.Internal, grpcCode)
	})
}

// Property: Nil error maps to OK
func TestProperty_NilErrorMapsToOK(t *testing.T) {
	httpStatus := cache.ToHTTPStatus(nil)
	grpcCode := cache.ToGRPCCode(nil)

	assert.Equal(t, http.StatusOK, httpStatus)
	assert.Equal(t, codes.OK, grpcCode)
}

// Property: Error categories are consistent
func TestProperty_ErrorCategoriesConsistent(t *testing.T) {
	// Validation errors should all be 400
	validationErrors := []error{
		cache.ErrInvalidKeyError,
		cache.ErrInvalidValueError,
		cache.ErrInvalidNamespace,
		cache.ErrInvalidTTL,
		cache.ErrTTLTooShort,
		cache.ErrTTLTooLong,
	}

	for _, err := range validationErrors {
		assert.Equal(t, http.StatusBadRequest, cache.ToHTTPStatus(err),
			"validation error %v should be 400", err)
		assert.Equal(t, codes.InvalidArgument, cache.ToGRPCCode(err),
			"validation error %v should be InvalidArgument", err)
	}

	// Availability errors should all be 503
	availabilityErrors := []error{
		cache.ErrRedisDown,
		cache.ErrCircuitBreakerOpen,
	}

	for _, err := range availabilityErrors {
		assert.Equal(t, http.StatusServiceUnavailable, cache.ToHTTPStatus(err),
			"availability error %v should be 503", err)
		assert.Equal(t, codes.Unavailable, cache.ToGRPCCode(err),
			"availability error %v should be Unavailable", err)
	}
}
