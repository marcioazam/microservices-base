package middleware

import (
	"github.com/auth-platform/sdk-go/src/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// errorCodeToGRPCStatus maps SDK error codes to gRPC status codes.
var errorCodeToGRPCStatus = map[errors.ErrorCode]codes.Code{
	errors.ErrCodeInvalidConfig: codes.InvalidArgument,
	errors.ErrCodeTokenExpired:  codes.Unauthenticated,
	errors.ErrCodeTokenInvalid:  codes.Unauthenticated,
	errors.ErrCodeTokenMissing:  codes.Unauthenticated,
	errors.ErrCodeTokenRefresh:  codes.Unauthenticated,
	errors.ErrCodeNetwork:       codes.Unavailable,
	errors.ErrCodeRateLimited:   codes.ResourceExhausted,
	errors.ErrCodeValidation:    codes.InvalidArgument,
	errors.ErrCodeUnauthorized:  codes.PermissionDenied,
	errors.ErrCodeDPoPRequired:  codes.Unauthenticated,
	errors.ErrCodeDPoPInvalid:   codes.Unauthenticated,
	errors.ErrCodePKCEInvalid:   codes.InvalidArgument,
}

// MapToGRPCError converts an SDK error to a gRPC status error.
func MapToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	code := errors.GetCode(err)
	if code == "" {
		return status.Error(codes.Internal, err.Error())
	}

	grpcCode, ok := errorCodeToGRPCStatus[code]
	if !ok {
		grpcCode = codes.Internal
	}

	return status.Error(grpcCode, err.Error())
}

// GRPCCodeForError returns the gRPC status code for an SDK error code.
func GRPCCodeForError(code errors.ErrorCode) codes.Code {
	if grpcCode, ok := errorCodeToGRPCStatus[code]; ok {
		return grpcCode
	}
	return codes.Internal
}

// IsUnauthenticated checks if the error maps to Unauthenticated.
func IsUnauthenticated(err error) bool {
	code := errors.GetCode(err)
	grpcCode := GRPCCodeForError(code)
	return grpcCode == codes.Unauthenticated
}

// IsPermissionDenied checks if the error maps to PermissionDenied.
func IsPermissionDenied(err error) bool {
	code := errors.GetCode(err)
	grpcCode := GRPCCodeForError(code)
	return grpcCode == codes.PermissionDenied
}
