package error

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ResilienceErrorMapping maps resilience error codes to gRPC status codes.
var ResilienceErrorMapping = map[ResilienceErrorCode]codes.Code{
	ErrCircuitOpen:        codes.Unavailable,
	ErrRateLimitExceeded:  codes.ResourceExhausted,
	ErrTimeout:            codes.DeadlineExceeded,
	ErrBulkheadFull:       codes.ResourceExhausted,
	ErrRetryExhausted:     codes.Unavailable,
	ErrInvalidPolicy:      codes.InvalidArgument,
	ErrServiceUnavailable: codes.Unavailable,
	ErrValidation:         codes.InvalidArgument,
}

// ToGRPCError converts a resilience error to a gRPC status error.
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	var resErr *ResilienceError
	if As(err, &resErr) {
		code, ok := ResilienceErrorMapping[resErr.Code]
		if !ok {
			code = codes.Internal
		}

		st := status.New(code, resErr.Message)
		return st.Err()
	}

	// Default to internal error
	return status.Error(codes.Internal, err.Error())
}

// ToGRPCCode returns the gRPC code for a resilience error code.
func ToGRPCCode(code ResilienceErrorCode) codes.Code {
	if grpcCode, ok := ResilienceErrorMapping[code]; ok {
		return grpcCode
	}
	return codes.Internal
}

// FromGRPCCode returns the resilience error code for a gRPC code.
func FromGRPCCode(code codes.Code) ResilienceErrorCode {
	for resCode, grpcCode := range ResilienceErrorMapping {
		if grpcCode == code {
			return resCode
		}
	}
	return ErrServiceUnavailable
}

// ToGRPCStatus converts a resilience error to a gRPC status.
func ToGRPCStatus(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}

	var resErr *ResilienceError
	if As(err, &resErr) {
		code, ok := ResilienceErrorMapping[resErr.Code]
		if !ok {
			code = codes.Internal
		}
		return status.New(code, resErr.Message)
	}

	return status.New(codes.Internal, err.Error())
}

// FromGRPCError converts a gRPC error to a ResilienceError.
func FromGRPCError(err error) *ResilienceError {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return &ResilienceError{
			Code:    ErrServiceUnavailable,
			Message: err.Error(),
		}
	}

	return &ResilienceError{
		Code:    FromGRPCCode(st.Code()),
		Message: st.Message(),
	}
}
