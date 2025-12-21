package grpc

import (
	"time"

	"github.com/authcorp/libs/go/src/fault"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToGRPCError converts a fault error to gRPC status.
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	// Check specific error types
	if fault.IsCircuitOpen(err) {
		circuitErr, _ := fault.AsCircuitOpenError(err)
		return status.Errorf(codes.Unavailable,
			"circuit breaker open for service %s: %s",
			circuitErr.Service, circuitErr.Message)
	}

	if fault.IsRateLimited(err) {
		rateErr, _ := fault.AsRateLimitError(err)
		return status.Errorf(codes.ResourceExhausted,
			"rate limit exceeded for service %s: limit %d per %v",
			rateErr.Service, rateErr.Limit, rateErr.Window)
	}

	if fault.IsTimeout(err) {
		timeoutErr, _ := fault.AsTimeoutError(err)
		return status.Errorf(codes.DeadlineExceeded,
			"timeout after %v for service %s",
			timeoutErr.Timeout, timeoutErr.Service)
	}

	if fault.IsBulkheadFull(err) {
		bulkheadErr, _ := fault.AsBulkheadFullError(err)
		return status.Errorf(codes.ResourceExhausted,
			"bulkhead full for service %s: max %d concurrent",
			bulkheadErr.Service, bulkheadErr.MaxConcurrent)
	}

	if fault.IsRetryExhausted(err) {
		retryErr, _ := fault.AsRetryExhaustedError(err)
		return status.Errorf(codes.Aborted,
			"retry exhausted after %d attempts for service %s",
			retryErr.Attempts, retryErr.Service)
	}

	if fault.IsInvalidPolicy(err) {
		policyErr, _ := fault.AsInvalidPolicyError(err)
		return status.Errorf(codes.InvalidArgument,
			"invalid policy: %s", policyErr.Field)
	}

	// Default to internal error
	return status.Errorf(codes.Internal, err.Error())
}

// FromGRPCError converts a gRPC status to fault error.
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
		return fault.NewCircuitOpenError("", "", time.Time{}, 0, 0)
	case codes.ResourceExhausted:
		return fault.NewRateLimitError("", "", 0, 0, 0)
	case codes.DeadlineExceeded:
		return fault.NewTimeoutError("", "", 0, 0, nil)
	case codes.Aborted:
		return fault.NewRetryExhaustedError("", "", 0, 0, nil)
	case codes.InvalidArgument:
		return fault.NewInvalidPolicyError("", nil, "")
	default:
		return err
	}
}

// ErrorCodeToGRPC maps fault error codes to gRPC codes.
func ErrorCodeToGRPC(code fault.ErrorCode) codes.Code {
	switch code {
	case fault.ErrCodeCircuitOpen:
		return codes.Unavailable
	case fault.ErrCodeRateLimited:
		return codes.ResourceExhausted
	case fault.ErrCodeTimeout:
		return codes.DeadlineExceeded
	case fault.ErrCodeBulkheadFull:
		return codes.ResourceExhausted
	case fault.ErrCodeRetryExhausted:
		return codes.Aborted
	case fault.ErrCodeInvalidPolicy:
		return codes.InvalidArgument
	default:
		return codes.Internal
	}
}

// GRPCToErrorCode maps gRPC codes to fault error codes.
func GRPCToErrorCode(code codes.Code) fault.ErrorCode {
	switch code {
	case codes.Unavailable:
		return fault.ErrCodeCircuitOpen
	case codes.ResourceExhausted:
		return fault.ErrCodeRateLimited
	case codes.DeadlineExceeded:
		return fault.ErrCodeTimeout
	case codes.Aborted:
		return fault.ErrCodeRetryExhausted
	case codes.InvalidArgument:
		return fault.ErrCodeInvalidPolicy
	default:
		return ""
	}
}

// ToGRPCStatus converts a fault error to a gRPC status.
func ToGRPCStatus(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}
	grpcErr := ToGRPCError(err)
	st, _ := status.FromError(grpcErr)
	return st
}

// IsUnavailable checks if the error represents an unavailable service.
func IsUnavailable(err error) bool {
	st, ok := status.FromError(ToGRPCError(err))
	return ok && st.Code() == codes.Unavailable
}

// IsResourceExhausted checks if the error represents resource exhaustion.
func IsResourceExhausted(err error) bool {
	st, ok := status.FromError(ToGRPCError(err))
	return ok && st.Code() == codes.ResourceExhausted
}

// IsDeadlineExceeded checks if the error represents a deadline exceeded.
func IsDeadlineExceeded(err error) bool {
	st, ok := status.FromError(ToGRPCError(err))
	return ok && st.Code() == codes.DeadlineExceeded
}

// IsInvalidArgument checks if the error represents an invalid argument.
func IsInvalidArgument(err error) bool {
	st, ok := status.FromError(ToGRPCError(err))
	return ok && st.Code() == codes.InvalidArgument
}
