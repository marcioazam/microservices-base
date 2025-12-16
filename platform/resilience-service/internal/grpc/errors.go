// Package grpc implements the gRPC service layer.
package grpc

import (
	"errors"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorMapping maps internal error codes to gRPC status codes.
var ErrorMapping = map[domain.ErrorCode]codes.Code{
	domain.ErrCircuitOpen:        codes.Unavailable,
	domain.ErrRateLimitExceeded:  codes.ResourceExhausted,
	domain.ErrTimeout:            codes.DeadlineExceeded,
	domain.ErrBulkheadFull:       codes.ResourceExhausted,
	domain.ErrRetryExhausted:     codes.Unavailable,
	domain.ErrInvalidPolicy:      codes.InvalidArgument,
	domain.ErrServiceUnavailable: codes.Unavailable,
}

// ToGRPCError converts a domain error to a gRPC status error.
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	var resErr *domain.ResilienceError
	if errors.As(err, &resErr) {
		code, ok := ErrorMapping[resErr.Code]
		if !ok {
			code = codes.Internal
		}

		st := status.New(code, resErr.Message)

		// Add details if available
		if resErr.Service != "" || len(resErr.Metadata) > 0 {
			// Could add structured details here using status.WithDetails
			// For now, just include in message
		}

		return st.Err()
	}

	// Default to internal error
	return status.Error(codes.Internal, err.Error())
}

// ToGRPCCode returns the gRPC code for a domain error code.
func ToGRPCCode(code domain.ErrorCode) codes.Code {
	if grpcCode, ok := ErrorMapping[code]; ok {
		return grpcCode
	}
	return codes.Internal
}

// FromGRPCCode returns the domain error code for a gRPC code.
func FromGRPCCode(code codes.Code) domain.ErrorCode {
	for domainCode, grpcCode := range ErrorMapping {
		if grpcCode == code {
			return domainCode
		}
	}
	return domain.ErrServiceUnavailable
}
