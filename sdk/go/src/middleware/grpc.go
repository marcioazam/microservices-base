package middleware

import (
	"context"

	"github.com/auth-platform/sdk-go/src/token"
	"google.golang.org/grpc"
)

// GRPCConfig holds gRPC interceptor configuration.
type GRPCConfig struct {
	SkipMethods    []string
	TokenExtractor token.Extractor
	Audience       string
	Issuer         string
	RequiredClaims []string
}

// GRPCOption configures gRPC interceptors.
type GRPCOption func(*GRPCConfig)

// WithGRPCSkipMethods sets methods to skip authentication.
func WithGRPCSkipMethods(methods ...string) GRPCOption {
	return func(c *GRPCConfig) { c.SkipMethods = methods }
}

// WithGRPCAudience sets the expected audience.
func WithGRPCAudience(audience string) GRPCOption {
	return func(c *GRPCConfig) { c.Audience = audience }
}

// WithGRPCIssuer sets the expected issuer.
func WithGRPCIssuer(issuer string) GRPCOption {
	return func(c *GRPCConfig) { c.Issuer = issuer }
}

// WithGRPCRequiredClaims sets required claims.
func WithGRPCRequiredClaims(claims ...string) GRPCOption {
	return func(c *GRPCConfig) { c.RequiredClaims = claims }
}

// NewGRPCConfig creates a new gRPC configuration.
func NewGRPCConfig(opts ...GRPCOption) *GRPCConfig {
	c := &GRPCConfig{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// GRPCInterceptor provides gRPC authentication interceptors.
type GRPCInterceptor struct {
	config    *GRPCConfig
	validator TokenValidator
}

// NewGRPCInterceptor creates a new gRPC interceptor.
func NewGRPCInterceptor(validator TokenValidator, opts ...GRPCOption) *GRPCInterceptor {
	return &GRPCInterceptor{
		config:    NewGRPCConfig(opts...),
		validator: validator,
	}
}

// UnaryServerInterceptor returns a unary server interceptor.
func (i *GRPCInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if i.shouldSkip(info.FullMethod) {
			return handler(ctx, req)
		}

		newCtx, err := i.authenticate(ctx)
		if err != nil {
			return nil, MapToGRPCError(err)
		}

		return handler(newCtx, req)
	}
}

// StreamServerInterceptor returns a stream server interceptor.
func (i *GRPCInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if i.shouldSkip(info.FullMethod) {
			return handler(srv, ss)
		}

		newCtx, err := i.authenticate(ss.Context())
		if err != nil {
			return MapToGRPCError(err)
		}

		wrapped := &wrappedServerStream{ServerStream: ss, ctx: newCtx}
		return handler(srv, wrapped)
	}
}

func (i *GRPCInterceptor) shouldSkip(method string) bool {
	for _, m := range i.config.SkipMethods {
		if m == method {
			return true
		}
	}
	return false
}

func (i *GRPCInterceptor) authenticate(ctx context.Context) (context.Context, error) {
	extractor := i.config.TokenExtractor
	if extractor == nil {
		extractor = token.NewGRPCExtractor()
	}

	tokenStr, _, err := extractor.Extract(ctx)
	if err != nil {
		return nil, err
	}

	result, err := i.validator.ValidateToken(tokenStr, i.config.Audience)
	if err != nil {
		return nil, err
	}

	ctx = ContextWithClaims(ctx, result.Claims)
	ctx = ContextWithToken(ctx, tokenStr)
	return ctx, nil
}

// wrappedServerStream wraps a grpc.ServerStream with a custom context.
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
