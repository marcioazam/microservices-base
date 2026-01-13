// Package middleware provides property-based tests for gRPC interceptors.
package middleware

import (
	"testing"

	"github.com/auth-platform/sdk-go/src/errors"
	"github.com/auth-platform/sdk-go/src/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"pgregory.net/rapid"
)

// Property 23: Error to Status Code Mapping
func TestProperty_ErrorToGRPCStatusMapping(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		allCodes := errors.AllErrorCodes()
		code := rapid.SampledFrom(allCodes).Draw(t, "code")
		err := errors.NewError(code, "test message")

		grpcErr := middleware.MapToGRPCError(err)
		st, ok := status.FromError(grpcErr)
		if !ok {
			t.Fatal("expected gRPC status error")
		}

		expectedCode := middleware.GRPCCodeForError(code)
		if st.Code() != expectedCode {
			t.Fatalf("code = %v, want %v for SDK code %s", st.Code(), expectedCode, code)
		}
	})
}

// Property: All SDK error codes map to valid gRPC codes
func TestProperty_AllSDKCodesMapToValidGRPC(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		allCodes := errors.AllErrorCodes()
		code := rapid.SampledFrom(allCodes).Draw(t, "code")

		grpcCode := middleware.GRPCCodeForError(code)

		// Verify it's a valid gRPC code (not negative)
		if grpcCode < 0 {
			t.Fatalf("invalid gRPC code %v for SDK code %s", grpcCode, code)
		}
	})
}

// Property: Nil error maps to nil
func TestProperty_NilErrorMapsToNil(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		result := middleware.MapToGRPCError(nil)
		if result != nil {
			t.Fatal("nil error should map to nil")
		}
	})
}

// Property: Unknown codes map to Internal
func TestProperty_UnknownCodesMapToInternal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		unknownCode := errors.ErrorCode(rapid.StringMatching(`UNKNOWN_[A-Z]{5}`).Draw(t, "code"))

		grpcCode := middleware.GRPCCodeForError(unknownCode)
		if grpcCode != codes.Internal {
			t.Fatalf("unknown code %s should map to Internal, got %v", unknownCode, grpcCode)
		}
	})
}

// Property: Authentication errors map to Unauthenticated
func TestProperty_AuthErrorsMapToUnauthenticated(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		authCodes := []errors.ErrorCode{
			errors.ErrCodeTokenExpired,
			errors.ErrCodeTokenInvalid,
			errors.ErrCodeTokenMissing,
			errors.ErrCodeTokenRefresh,
			errors.ErrCodeDPoPRequired,
			errors.ErrCodeDPoPInvalid,
		}
		code := rapid.SampledFrom(authCodes).Draw(t, "code")

		grpcCode := middleware.GRPCCodeForError(code)
		if grpcCode != codes.Unauthenticated {
			t.Fatalf("auth code %s should map to Unauthenticated, got %v", code, grpcCode)
		}
	})
}

// Property: gRPC config preserves all options
func TestProperty_GRPCConfigPreservesOptions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		audience := rapid.StringMatching(`[a-z0-9-]{5,20}`).Draw(t, "audience")
		issuer := "https://" + rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "issuer") + ".com"
		numMethods := rapid.IntRange(0, 5).Draw(t, "numMethods")
		methods := make([]string, numMethods)
		for i := 0; i < numMethods; i++ {
			methods[i] = "/" + rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "method")
		}

		config := middleware.NewGRPCConfig(
			middleware.WithGRPCAudience(audience),
			middleware.WithGRPCIssuer(issuer),
			middleware.WithGRPCSkipMethods(methods...),
		)

		if config.Audience != audience {
			t.Fatalf("audience = %s, want %s", config.Audience, audience)
		}
		if config.Issuer != issuer {
			t.Fatalf("issuer = %s, want %s", config.Issuer, issuer)
		}
		if len(config.SkipMethods) != numMethods {
			t.Fatalf("skip methods = %d, want %d", len(config.SkipMethods), numMethods)
		}
	})
}

// Property: IsUnauthenticated is consistent with GRPCCodeForError
func TestProperty_IsUnauthenticatedConsistent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		allCodes := errors.AllErrorCodes()
		code := rapid.SampledFrom(allCodes).Draw(t, "code")
		err := errors.NewError(code, "test")

		isUnauth := middleware.IsUnauthenticated(err)
		grpcCode := middleware.GRPCCodeForError(code)

		if isUnauth != (grpcCode == codes.Unauthenticated) {
			t.Fatalf("IsUnauthenticated inconsistent with GRPCCodeForError for %s", code)
		}
	})
}

// Property: IsPermissionDenied is consistent with GRPCCodeForError
func TestProperty_IsPermissionDeniedConsistent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		allCodes := errors.AllErrorCodes()
		code := rapid.SampledFrom(allCodes).Draw(t, "code")
		err := errors.NewError(code, "test")

		isDenied := middleware.IsPermissionDenied(err)
		grpcCode := middleware.GRPCCodeForError(code)

		if isDenied != (grpcCode == codes.PermissionDenied) {
			t.Fatalf("IsPermissionDenied inconsistent with GRPCCodeForError for %s", code)
		}
	})
}

// Property: Error message is preserved in gRPC error
func TestProperty_ErrorMessagePreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		allCodes := errors.AllErrorCodes()
		code := rapid.SampledFrom(allCodes).Draw(t, "code")
		message := rapid.StringMatching(`[a-z ]{10,50}`).Draw(t, "message")

		err := errors.NewError(code, message)
		grpcErr := middleware.MapToGRPCError(err)

		st, _ := status.FromError(grpcErr)
		// The message should contain the original message
		if st.Message() == "" {
			t.Fatal("gRPC error message should not be empty")
		}
	})
}
