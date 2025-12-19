// Package errors provides gRPC error mapping utilities.
package errors

import (
	"errors"

	resilienceerrors "github.com/auth-platform/libs/go/resilience/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToGRPCCode converts a resilience error to a gRPC status code.
func ToGRPCCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}

	var resErr *resilienceerrors.ResilienceError
	if errors.As(err, &resErr) {
		return codeFromResilienceError(resErr.Code)
	}

	// Check specific error types
	var circuitErr *resilienceerrors.CircuitOpenError
	if errors.As(err, &circuitErr) {
		return codes.Unavailable
	}

	var rateLimitErr *resilienceerrors.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return codes.ResourceExhausted
	}

	var timeoutErr *resilienceerrors.TimeoutError
	if errors.As(err, &timeoutErr) {
		return codes.DeadlineExceeded
	}

	var bulkheadErr *resilienceerrors.BulkheadFullError
	if errors.As(err, &bulkheadErr) {
		return codes.ResourceExhausted
	}

	var retryErr *resilienceerrors.RetryExhaustedError
	if errors.As(err, &retryErr) {
		return codes.Unavailable
	}

	var invalidPolicyErr *resilienceerrors.InvalidPolicyError
	if errors.As(err, &invalidPolicyErr) {
		return codes.InvalidArgument
	}

	return codes.Internal
}

// ToGRPCError converts a resilience error to a gRPC status error.
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	code := ToGRPCCode(err)
	return status.Error(code, err.Error())
}

// ToGRPCStatus converts a resilience error to a gRPC status.
func ToGRPCStatus(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}

	code := ToGRPCCode(err)
	return status.New(code, err.Error())
}

// FromGRPCCode converts a gRPC status code to a resilience error code.
func FromGRPCCode(code codes.Code) resilienceerrors.ErrorCode {
	switch code {
	case codes.Unavailable:
		return resilienceerrors.ErrCircuitOpen
	case codes.ResourceExhausted:
		return resilienceerrors.ErrRateLimitExceeded
	case codes.DeadlineExceeded:
		return resilienceerrors.ErrTimeout
	case codes.InvalidArgument:
		return resilienceerrors.ErrInvalidPolicy
	default:
		return ""
	}
}

// FromGRPCError converts a gRPC error to a resilience error.
func FromGRPCError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	code := FromGRPCCode(st.Code())
	if code == "" {
		return err
	}

	return &resilienceerrors.ResilienceError{
		Code:    code,
		Service: "grpc",
		Message: st.Message(),
	}
}

func codeFromResilienceError(code resilienceerrors.ErrorCode) codes.Code {
	switch code {
	case resilienceerrors.ErrCircuitOpen:
		return codes.Unavailable
	case resilienceerrors.ErrRateLimitExceeded:
		return codes.ResourceExhausted
	case resilienceerrors.ErrTimeout:
		return codes.DeadlineExceeded
	case resilienceerrors.ErrBulkheadFull:
		return codes.ResourceExhausted
	case resilienceerrors.ErrRetryExhausted:
		return codes.Unavailable
	case resilienceerrors.ErrInvalidPolicy:
		return codes.InvalidArgument
	default:
		return codes.Internal
	}
}

// IsUnavailable checks if the error represents an unavailable service.
func IsUnavailable(err error) bool {
	return ToGRPCCode(err) == codes.Unavailable
}

// IsResourceExhausted checks if the error represents resource exhaustion.
func IsResourceExhausted(err error) bool {
	return ToGRPCCode(err) == codes.ResourceExhausted
}

// IsDeadlineExceeded checks if the error represents a deadline exceeded.
func IsDeadlineExceeded(err error) bool {
	return ToGRPCCode(err) == codes.DeadlineExceeded
}

// IsInvalidArgument checks if the error represents an invalid argument.
func IsInvalidArgument(err error) bool {
	return ToGRPCCode(err) == codes.InvalidArgument
}
