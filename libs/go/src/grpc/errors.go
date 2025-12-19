package grpc

import (
	"time"

	"github.com/authcorp/libs/go/src/resilience"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToGRPCError converts a resilience error to gRPC status.
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	// Check specific error types
	if resilience.IsCircuitOpen(err) {
		circuitErr, _ := resilience.AsCircuitOpenError(err)
		return status.Errorf(codes.Unavailable,
			"circuit breaker open for service %s: %s",
			circuitErr.Service, circuitErr.Message)
	}

	if resilience.IsRateLimited(err) {
		rateErr, _ := resilience.AsRateLimitError(err)
		return status.Errorf(codes.ResourceExhausted,
			"rate limit exceeded for service %s: limit %d per %v",
			rateErr.Service, rateErr.Limit, rateErr.Window)
	}

	if resilience.IsTimeout(err) {
		timeoutErr, _ := resilience.AsTimeoutError(err)
		return status.Errorf(codes.DeadlineExceeded,
			"timeout after %v for service %s",
			timeoutErr.Timeout, timeoutErr.Service)
	}

	if resilience.IsBulkheadFull(err) {
		bulkheadErr, _ := resilience.AsBulkheadFullError(err)
		return status.Errorf(codes.ResourceExhausted,
			"bulkhead full for service %s: max %d concurrent",
			bulkheadErr.Service, bulkheadErr.MaxConcurrent)
	}

	if resilience.IsRetryExhausted(err) {
		retryErr, _ := resilience.AsRetryExhaustedError(err)
		return status.Errorf(codes.Aborted,
			"retry exhausted after %d attempts for service %s",
			retryErr.Attempts, retryErr.Service)
	}

	if resilience.IsInvalidPolicy(err) {
		policyErr, _ := resilience.AsInvalidPolicyError(err)
		return status.Errorf(codes.InvalidArgument,
			"invalid policy: %s", policyErr.Field)
	}

	// Default to internal error
	return status.Errorf(codes.Internal, err.Error())
}

// FromGRPCError converts a gRPC status to resilience error.
func FromGRPCError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	switch st.Code() {
	case codes.Unavailable:
		return resilience.NewCircuitOpenError("", "", time.Time{}, 0, 0)
	case codes.ResourceExhausted:
		return resilience.NewRateLimitError("", "", 0, 0, 0)
	case codes.DeadlineExceeded:
		return resilience.NewTimeoutError("", "", 0, 0, nil)
	case codes.Aborted:
		return resilience.NewRetryExhaustedError("", "", 0, 0, nil)
	case codes.InvalidArgument:
		return resilience.NewInvalidPolicyError("", nil, "")
	default:
		return err
	}
}

// ErrorCodeToGRPC maps resilience error codes to gRPC codes.
func ErrorCodeToGRPC(code resilience.ErrorCode) codes.Code {
	switch code {
	case resilience.ErrCodeCircuitOpen:
		return codes.Unavailable
	case resilience.ErrCodeRateLimited:
		return codes.ResourceExhausted
	case resilience.ErrCodeTimeout:
		return codes.DeadlineExceeded
	case resilience.ErrCodeBulkheadFull:
		return codes.ResourceExhausted
	case resilience.ErrCodeRetryExhausted:
		return codes.Aborted
	case resilience.ErrCodeInvalidPolicy:
		return codes.InvalidArgument
	default:
		return codes.Internal
	}
}

// GRPCToErrorCode maps gRPC codes to resilience error codes.
func GRPCToErrorCode(code codes.Code) resilience.ErrorCode {
	switch code {
	case codes.Unavailable:
		return resilience.ErrCodeCircuitOpen
	case codes.ResourceExhausted:
		return resilience.ErrCodeRateLimited
	case codes.DeadlineExceeded:
		return resilience.ErrCodeTimeout
	case codes.Aborted:
		return resilience.ErrCodeRetryExhausted
	case codes.InvalidArgument:
		return resilience.ErrCodeInvalidPolicy
	default:
		return ""
	}
}
