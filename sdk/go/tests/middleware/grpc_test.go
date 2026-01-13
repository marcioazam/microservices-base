// Package middleware provides unit tests for gRPC interceptors.
package middleware

import (
	"testing"

	"github.com/auth-platform/sdk-go/src/errors"
	"github.com/auth-platform/sdk-go/src/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapToGRPCError_Nil(t *testing.T) {
	err := middleware.MapToGRPCError(nil)
	if err != nil {
		t.Errorf("MapToGRPCError(nil) = %v, want nil", err)
	}
}

func TestMapToGRPCError_TokenExpired(t *testing.T) {
	sdkErr := errors.NewError(errors.ErrCodeTokenExpired, "token expired")
	grpcErr := middleware.MapToGRPCError(sdkErr)

	st, ok := status.FromError(grpcErr)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("code = %v, want Unauthenticated", st.Code())
	}
}

func TestMapToGRPCError_TokenInvalid(t *testing.T) {
	sdkErr := errors.NewError(errors.ErrCodeTokenInvalid, "invalid token")
	grpcErr := middleware.MapToGRPCError(sdkErr)

	st, _ := status.FromError(grpcErr)
	if st.Code() != codes.Unauthenticated {
		t.Errorf("code = %v, want Unauthenticated", st.Code())
	}
}

func TestMapToGRPCError_RateLimited(t *testing.T) {
	sdkErr := errors.NewError(errors.ErrCodeRateLimited, "rate limited")
	grpcErr := middleware.MapToGRPCError(sdkErr)

	st, _ := status.FromError(grpcErr)
	if st.Code() != codes.ResourceExhausted {
		t.Errorf("code = %v, want ResourceExhausted", st.Code())
	}
}

func TestMapToGRPCError_Network(t *testing.T) {
	sdkErr := errors.NewError(errors.ErrCodeNetwork, "network error")
	grpcErr := middleware.MapToGRPCError(sdkErr)

	st, _ := status.FromError(grpcErr)
	if st.Code() != codes.Unavailable {
		t.Errorf("code = %v, want Unavailable", st.Code())
	}
}

func TestMapToGRPCError_Validation(t *testing.T) {
	sdkErr := errors.NewError(errors.ErrCodeValidation, "validation failed")
	grpcErr := middleware.MapToGRPCError(sdkErr)

	st, _ := status.FromError(grpcErr)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", st.Code())
	}
}

func TestMapToGRPCError_Unauthorized(t *testing.T) {
	sdkErr := errors.NewError(errors.ErrCodeUnauthorized, "unauthorized")
	grpcErr := middleware.MapToGRPCError(sdkErr)

	st, _ := status.FromError(grpcErr)
	if st.Code() != codes.PermissionDenied {
		t.Errorf("code = %v, want PermissionDenied", st.Code())
	}
}

func TestMapToGRPCError_UnknownError(t *testing.T) {
	// Regular error without SDK error code
	grpcErr := middleware.MapToGRPCError(errors.NewError("", "unknown"))

	st, _ := status.FromError(grpcErr)
	if st.Code() != codes.Internal {
		t.Errorf("code = %v, want Internal", st.Code())
	}
}

func TestGRPCCodeForError(t *testing.T) {
	tests := []struct {
		code     errors.ErrorCode
		expected codes.Code
	}{
		{errors.ErrCodeTokenExpired, codes.Unauthenticated},
		{errors.ErrCodeTokenInvalid, codes.Unauthenticated},
		{errors.ErrCodeTokenMissing, codes.Unauthenticated},
		{errors.ErrCodeRateLimited, codes.ResourceExhausted},
		{errors.ErrCodeNetwork, codes.Unavailable},
		{errors.ErrCodeValidation, codes.InvalidArgument},
		{errors.ErrCodeUnauthorized, codes.PermissionDenied},
		{errors.ErrCodeDPoPRequired, codes.Unauthenticated},
		{errors.ErrCodeDPoPInvalid, codes.Unauthenticated},
		{errors.ErrCodePKCEInvalid, codes.InvalidArgument},
		{"UNKNOWN_CODE", codes.Internal},
	}

	for _, tt := range tests {
		got := middleware.GRPCCodeForError(tt.code)
		if got != tt.expected {
			t.Errorf("GRPCCodeForError(%s) = %v, want %v", tt.code, got, tt.expected)
		}
	}
}

func TestIsUnauthenticated(t *testing.T) {
	tests := []struct {
		code     errors.ErrorCode
		expected bool
	}{
		{errors.ErrCodeTokenExpired, true},
		{errors.ErrCodeTokenInvalid, true},
		{errors.ErrCodeTokenMissing, true},
		{errors.ErrCodeDPoPRequired, true},
		{errors.ErrCodeRateLimited, false},
		{errors.ErrCodeUnauthorized, false},
	}

	for _, tt := range tests {
		err := errors.NewError(tt.code, "test")
		got := middleware.IsUnauthenticated(err)
		if got != tt.expected {
			t.Errorf("IsUnauthenticated(%s) = %v, want %v", tt.code, got, tt.expected)
		}
	}
}

func TestIsPermissionDenied(t *testing.T) {
	tests := []struct {
		code     errors.ErrorCode
		expected bool
	}{
		{errors.ErrCodeUnauthorized, true},
		{errors.ErrCodeTokenExpired, false},
		{errors.ErrCodeRateLimited, false},
	}

	for _, tt := range tests {
		err := errors.NewError(tt.code, "test")
		got := middleware.IsPermissionDenied(err)
		if got != tt.expected {
			t.Errorf("IsPermissionDenied(%s) = %v, want %v", tt.code, got, tt.expected)
		}
	}
}

func TestGRPCConfig_Options(t *testing.T) {
	config := middleware.NewGRPCConfig(
		middleware.WithGRPCSkipMethods("/health.Health/Check"),
		middleware.WithGRPCAudience("my-api"),
		middleware.WithGRPCIssuer("https://issuer.com"),
		middleware.WithGRPCRequiredClaims("sub", "scope"),
	)

	if len(config.SkipMethods) != 1 {
		t.Errorf("SkipMethods length = %d, want 1", len(config.SkipMethods))
	}
	if config.Audience != "my-api" {
		t.Errorf("Audience = %s, want my-api", config.Audience)
	}
	if config.Issuer != "https://issuer.com" {
		t.Errorf("Issuer = %s, want https://issuer.com", config.Issuer)
	}
	if len(config.RequiredClaims) != 2 {
		t.Errorf("RequiredClaims length = %d, want 2", len(config.RequiredClaims))
	}
}
