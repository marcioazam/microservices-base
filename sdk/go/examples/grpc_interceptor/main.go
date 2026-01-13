// Package main demonstrates gRPC interceptor usage.
package main

import (
	"context"
	"fmt"

	sdk "github.com/auth-platform/sdk-go/src"
	"github.com/auth-platform/sdk-go/src/middleware"
	"github.com/auth-platform/sdk-go/src/token"
	"github.com/auth-platform/sdk-go/src/types"
	"google.golang.org/grpc"
)

// mockValidator implements middleware.TokenValidator for demonstration.
type mockValidator struct{}

func (m *mockValidator) ValidateToken(tokenStr string, audience string) (*token.ValidationResult, error) {
	// In production, this would validate against JWKS
	return token.NewValidationResult(
		&types.Claims{Subject: "user123", Scope: "read write"},
		tokenStr,
		token.SchemeBearer,
	), nil
}

func main() {
	fmt.Println("gRPC Interceptor Example")
	fmt.Println("========================")

	// Create gRPC interceptor
	validator := &mockValidator{}
	interceptor := middleware.NewGRPCInterceptor(
		validator,
		middleware.WithGRPCSkipMethods("/health.Health/Check"),
		middleware.WithGRPCAudience("my-grpc-service"),
	)

	// Get the interceptors
	unaryInterceptor := interceptor.UnaryServerInterceptor()
	streamInterceptor := interceptor.StreamServerInterceptor()

	fmt.Println("Interceptors created:")
	fmt.Printf("  Unary: %T\n", unaryInterceptor)
	fmt.Printf("  Stream: %T\n", streamInterceptor)

	// Example: Create gRPC server with interceptors
	_ = grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)

	fmt.Println("\ngRPC server would be configured with auth interceptors")

	// Demonstrate error mapping
	fmt.Println("\nError Mapping Examples:")
	testErrors := []struct {
		code sdk.ErrorCode
		name string
	}{
		{"TOKEN_EXPIRED", "TokenExpired"},
		{"TOKEN_INVALID", "TokenInvalid"},
		{"RATE_LIMITED", "RateLimited"},
		{"UNAUTHORIZED", "Unauthorized"},
	}

	for _, te := range testErrors {
		err := sdk.NewError(te.code, "test error")
		grpcErr := sdk.MapToGRPCError(err)
		fmt.Printf("  %s -> %v\n", te.name, grpcErr)
	}

	// Demonstrate context helpers
	fmt.Println("\nContext Helpers:")
	claims := &types.Claims{Subject: "user123", Scope: "read write admin"}
	ctx := middleware.ContextWithClaims(context.Background(), claims)

	if subject, ok := sdk.GetSubject(ctx); ok {
		fmt.Printf("  Subject: %s\n", subject)
	}

	fmt.Printf("  Has 'read' scope: %v\n", sdk.HasScope(ctx, "read"))
	fmt.Printf("  Has 'delete' scope: %v\n", sdk.HasScope(ctx, "delete"))
}
