package grpc

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"pgregory.net/rapid"

	"github.com/authcorp/libs/go/src/fault"
)

// TestGRPCErrorMappingProperty validates that fault errors map to correct gRPC codes.
// Property 7: gRPC Error Code Mapping
// Validates: Requirements 5.6, 8.1
func TestGRPCErrorMappingProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate error type
		errorType := rapid.IntRange(0, 5).Draw(t, "errorType")

		var inputErr error
		var expectedCode codes.Code

		switch errorType {
		case 0:
			inputErr = fault.NewCircuitOpenError("test-service", "corr-123", 
				rapid.Just(fault.CircuitOpenError{}).Draw(t, "circuitErr").OpenedAt, 0, 0)
			expectedCode = codes.Unavailable
		case 1:
			inputErr = fault.NewRateLimitError("test-service", "corr-123", 100, 0, 0)
			expectedCode = codes.ResourceExhausted
		case 2:
			inputErr = fault.NewTimeoutError("test-service", "corr-123", 0, 0, nil)
			expectedCode = codes.DeadlineExceeded
		case 3:
			inputErr = fault.NewBulkheadFullError("test-service", "corr-123", 10, 100, 10)
			expectedCode = codes.ResourceExhausted
		case 4:
			inputErr = fault.NewRetryExhaustedError("test-service", "corr-123", 3, 0, nil)
			expectedCode = codes.Aborted
		case 5:
			inputErr = fault.NewInvalidPolicyError("field", nil, "expected")
			expectedCode = codes.InvalidArgument
		}

		// Convert to gRPC error
		grpcErr := ToGRPCError(inputErr)

		// Extract status
		st, ok := status.FromError(grpcErr)
		if !ok {
			t.Errorf("expected gRPC status error, got: %T", grpcErr)
			return
		}

		// Property: Error code should match expected
		if st.Code() != expectedCode {
			t.Errorf("expected code %v for error type %d, got: %v", expectedCode, errorType, st.Code())
		}
	})
}

// TestGRPCErrorNilProperty validates nil error handling.
func TestGRPCErrorNilProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Property: nil input should return nil output
		result := ToGRPCError(nil)
		if result != nil {
			t.Errorf("expected nil for nil input, got: %v", result)
		}
	})
}

// TestGRPCErrorUnknownProperty validates unknown error handling.
func TestGRPCErrorUnknownProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random error message
		msg := rapid.String().Draw(t, "errorMsg")
		if msg == "" {
			msg = "unknown error"
		}

		inputErr := errors.New(msg)
		grpcErr := ToGRPCError(inputErr)

		st, ok := status.FromError(grpcErr)
		if !ok {
			t.Errorf("expected gRPC status error, got: %T", grpcErr)
			return
		}

		// Property: Unknown errors should map to Internal
		if st.Code() != codes.Internal {
			t.Errorf("expected Internal code for unknown error, got: %v", st.Code())
		}
	})
}

// TestGRPCHelperFunctionsProperty validates helper function consistency.
func TestGRPCHelperFunctionsProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		msg := rapid.StringN(1, 100, 100).Draw(t, "message")

		testCases := []struct {
			name     string
			fn       func(string) error
			expected codes.Code
		}{
			{"InvalidArgument", InvalidArgumentError, codes.InvalidArgument},
			{"NotFound", NotFoundError, codes.NotFound},
			{"Unauthenticated", UnauthenticatedError, codes.Unauthenticated},
			{"PermissionDenied", PermissionDeniedError, codes.PermissionDenied},
			{"Unavailable", UnavailableError, codes.Unavailable},
			{"Internal", InternalError, codes.Internal},
		}

		for _, tc := range testCases {
			err := tc.fn(msg)
			st, ok := status.FromError(err)
			if !ok {
				t.Errorf("%s: expected gRPC status error", tc.name)
				continue
			}

			// Property: Helper functions should return correct codes
			if st.Code() != tc.expected {
				t.Errorf("%s: expected %v, got %v", tc.name, tc.expected, st.Code())
			}

			// Property: Message should be preserved
			if st.Message() != msg {
				t.Errorf("%s: expected message %q, got %q", tc.name, msg, st.Message())
			}
		}
	})
}
