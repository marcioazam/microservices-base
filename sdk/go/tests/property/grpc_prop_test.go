package property

import (
	"testing"

	authplatform "github.com/auth-platform/sdk-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"pgregory.net/rapid"
)

// TestProperty14_GRPCStatusCodesForValidationFailures tests proper gRPC status code mapping.
func TestProperty14_GRPCStatusCodesForValidationFailures(t *testing.T) {
	testCases := []struct {
		name         string
		errorCode    authplatform.ErrorCode
		expectedCode codes.Code
	}{
		{"TokenMissing", authplatform.ErrCodeTokenMissing, codes.Unauthenticated},
		{"TokenInvalid", authplatform.ErrCodeTokenInvalid, codes.Unauthenticated},
		{"TokenExpired", authplatform.ErrCodeTokenExpired, codes.Unauthenticated},
		{"Validation", authplatform.ErrCodeValidation, codes.InvalidArgument},
		{"Network", authplatform.ErrCodeNetwork, codes.Unavailable},
		{"RateLimited", authplatform.ErrCodeRateLimited, codes.ResourceExhausted},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sdkErr := &authplatform.SDKError{
				Code:    tc.errorCode,
				Message: "test error",
			}

			grpcErr := authplatform.MapToGRPCError(sdkErr)
			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatal("expected gRPC status error")
			}

			if st.Code() != tc.expectedCode {
				t.Errorf("expected code %v, got %v", tc.expectedCode, st.Code())
			}
		})
	}
}

func TestGRPCInterceptorConfigOptions(t *testing.T) {
	t.Run("WithGRPCSkipMethods", func(t *testing.T) {
		config := &authplatform.GRPCInterceptorConfig{}
		opt := authplatform.WithGRPCSkipMethods("/health.Health/Check", "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo")
		opt(config)

		if len(config.SkipMethods) != 2 {
			t.Errorf("expected 2 skip methods, got %d", len(config.SkipMethods))
		}
	})

	t.Run("WithGRPCAudience", func(t *testing.T) {
		config := &authplatform.GRPCInterceptorConfig{}
		opt := authplatform.WithGRPCAudience("grpc-service")
		opt(config)

		if config.Audience != "grpc-service" {
			t.Errorf("expected audience grpc-service, got %s", config.Audience)
		}
	})

	t.Run("WithGRPCIssuer", func(t *testing.T) {
		config := &authplatform.GRPCInterceptorConfig{}
		opt := authplatform.WithGRPCIssuer("https://auth.example.com")
		opt(config)

		if config.Issuer != "https://auth.example.com" {
			t.Errorf("expected issuer https://auth.example.com, got %s", config.Issuer)
		}
	})

	t.Run("WithGRPCRequiredClaims", func(t *testing.T) {
		config := &authplatform.GRPCInterceptorConfig{}
		opt := authplatform.WithGRPCRequiredClaims("sub", "scope")
		opt(config)

		if len(config.RequiredClaims) != 2 {
			t.Errorf("expected 2 required claims, got %d", len(config.RequiredClaims))
		}
	})
}

func TestSkipMethodMatching(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		method := rapid.StringMatching(`^/[a-z]+\.[A-Z][a-z]+/[A-Z][a-z]+$`).Draw(t, "method")
		skipMethods := []string{method}

		// Method should be skipped when in skip list
		if !authplatform.ShouldSkipMethod(method, skipMethods) {
			t.Errorf("method %s should be skipped", method)
		}
	})
}

func TestSkipMethodSuffixMatching(t *testing.T) {
	testCases := []struct {
		method      string
		skipMethods []string
		shouldSkip  bool
	}{
		{"/health.Health/Check", []string{"/Check"}, true},
		{"/api.Service/GetUser", []string{"/GetUser"}, true},
		{"/api.Service/GetUser", []string{"/DeleteUser"}, false},
		{"/health.Health/Check", []string{"/health.Health/Check"}, true},
		{"/api.Service/Method", []string{}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			result := authplatform.ShouldSkipMethod(tc.method, tc.skipMethods)
			if result != tc.shouldSkip {
				t.Errorf("method %s with skip %v: expected %v, got %v",
					tc.method, tc.skipMethods, tc.shouldSkip, result)
			}
		})
	}
}

func TestNilErrorMapping(t *testing.T) {
	result := authplatform.MapToGRPCError(nil)
	if result != nil {
		t.Error("expected nil error to map to nil")
	}
}

func TestNonSDKErrorMapping(t *testing.T) {
	err := status.Error(codes.Unknown, "some error")
	result := authplatform.MapToGRPCError(err)

	st, ok := status.FromError(result)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.Internal {
		t.Errorf("expected Internal code for non-SDK error, got %v", st.Code())
	}
}
