package grpc

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/auth-platform/cache-service/internal/cache"
	libgrpc "github.com/authcorp/libs/go/src/grpc"
)

// ToGRPCError converts a cache error to a gRPC error.
// Uses lib grpc for fault tolerance errors, falls back to cache-specific handling.
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	// First try lib grpc conversion for fault errors
	if grpcErr := libgrpc.ToGRPCError(err); grpcErr != err {
		return grpcErr
	}

	var cacheErr *cache.Error
	if errors.As(err, &cacheErr) {
		return cacheErr.ToGRPCStatus().Err()
	}

	// Handle specific error checks for backward compatibility
	if cache.IsNotFound(err) {
		return status.Error(codes.NotFound, err.Error())
	}
	if cache.IsRedisUnavailable(err) {
		return status.Error(codes.Unavailable, err.Error())
	}
	if cache.IsCircuitOpen(err) {
		return status.Error(codes.Unavailable, err.Error())
	}

	return status.Error(codes.Internal, err.Error())
}

// InvalidArgumentError creates an InvalidArgument gRPC error.
func InvalidArgumentError(message string) error {
	return status.Error(codes.InvalidArgument, message)
}

// NotFoundError creates a NotFound gRPC error.
func NotFoundError(message string) error {
	return status.Error(codes.NotFound, message)
}

// UnauthenticatedError creates an Unauthenticated gRPC error.
func UnauthenticatedError(message string) error {
	return status.Error(codes.Unauthenticated, message)
}

// PermissionDeniedError creates a PermissionDenied gRPC error.
func PermissionDeniedError(message string) error {
	return status.Error(codes.PermissionDenied, message)
}

// UnavailableError creates an Unavailable gRPC error.
func UnavailableError(message string) error {
	return status.Error(codes.Unavailable, message)
}

// InternalError creates an Internal gRPC error.
func InternalError(message string) error {
	return status.Error(codes.Internal, message)
}
