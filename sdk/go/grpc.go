package authplatform

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCInterceptorConfig holds configuration for gRPC interceptors.
type GRPCInterceptorConfig struct {
	SkipMethods    []string
	TokenExtractor TokenExtractor
	Audience       string
	Issuer         string
	RequiredClaims []string
}

// GRPCInterceptorOption configures gRPC interceptors.
type GRPCInterceptorOption func(*GRPCInterceptorConfig)

// WithGRPCSkipMethods sets methods to skip authentication.
func WithGRPCSkipMethods(methods ...string) GRPCInterceptorOption {
	return func(c *GRPCInterceptorConfig) {
		c.SkipMethods = methods
	}
}

// WithGRPCAudience sets the expected audience.
func WithGRPCAudience(audience string) GRPCInterceptorOption {
	return func(c *GRPCInterceptorConfig) {
		c.Audience = audience
	}
}

// WithGRPCIssuer sets the expected issuer.
func WithGRPCIssuer(issuer string) GRPCInterceptorOption {
	return func(c *GRPCInterceptorConfig) {
		c.Issuer = issuer
	}
}

// WithGRPCRequiredClaims sets required claims.
func WithGRPCRequiredClaims(claims ...string) GRPCInterceptorOption {
	return func(c *GRPCInterceptorConfig) {
		c.RequiredClaims = claims
	}
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for token validation.
func (c *Client) UnaryServerInterceptor(opts ...GRPCInterceptorOption) grpc.UnaryServerInterceptor {
	config := &GRPCInterceptorConfig{
		TokenExtractor: NewGRPCTokenExtractor(),
	}
	for _, opt := range opts {
		opt(config)
	}

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Check skip methods
		if ShouldSkipMethod(info.FullMethod, config.SkipMethods) {
			return handler(ctx, req)
		}

		claims, err := c.validateGRPCRequest(ctx, config)
		if err != nil {
			return nil, err
		}

		ctx = context.WithValue(ctx, ClaimsContextKey, claims)
		return handler(ctx, req)
	}
}


// StreamServerInterceptor returns a gRPC stream server interceptor for token validation.
func (c *Client) StreamServerInterceptor(opts ...GRPCInterceptorOption) grpc.StreamServerInterceptor {
	config := &GRPCInterceptorConfig{
		TokenExtractor: NewGRPCTokenExtractor(),
	}
	for _, opt := range opts {
		opt(config)
	}

	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Check skip methods
		if ShouldSkipMethod(info.FullMethod, config.SkipMethods) {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		claims, err := c.validateGRPCRequest(ctx, config)
		if err != nil {
			return err
		}

		wrapped := &wrappedServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(ctx, ClaimsContextKey, claims),
		}
		return handler(srv, wrapped)
	}
}

func (c *Client) validateGRPCRequest(ctx context.Context, config *GRPCInterceptorConfig) (*Claims, error) {
	token, scheme, err := extractGRPCToken(ctx)
	if err != nil {
		return nil, MapToGRPCError(err)
	}

	// DPoP tokens require additional validation
	if scheme == TokenSchemeDPoP {
		return nil, status.Error(codes.Unimplemented, "DPoP validation not yet supported for gRPC")
	}

	// Validate token
	if c.jwksCache != nil {
		opts := ValidationOptions{
			Audience:       config.Audience,
			Issuer:         config.Issuer,
			RequiredClaims: config.RequiredClaims,
		}
		claims, err := c.jwksCache.ValidateTokenWithOpts(ctx, token, opts)
		if err != nil {
			return nil, MapToGRPCError(err)
		}
		return claims, nil
	}

	claims, err := c.ValidateToken(ctx, token)
	if err != nil {
		return nil, MapToGRPCError(err)
	}
	return claims, nil
}

func extractGRPCToken(ctx context.Context) (string, TokenScheme, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", &SDKError{Code: ErrCodeUnauthorized, Message: "missing metadata"}
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", "", &SDKError{Code: ErrCodeUnauthorized, Message: "missing authorization header"}
	}

	auth := values[0]
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		return "", "", &SDKError{Code: ErrCodeTokenInvalid, Message: "invalid authorization format"}
	}

	scheme := strings.ToLower(parts[0])
	switch scheme {
	case "bearer":
		return parts[1], TokenSchemeBearer, nil
	case "dpop":
		return parts[1], TokenSchemeDPoP, nil
	default:
		return "", "", &SDKError{Code: ErrCodeTokenInvalid, Message: "unsupported token scheme"}
	}
}

// MapToGRPCError maps SDK errors to appropriate gRPC status codes.
func MapToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	sdkErr, ok := err.(*SDKError)
	if !ok {
		return status.Error(codes.Internal, err.Error())
	}

	switch sdkErr.Code {
	case ErrCodeUnauthorized, ErrCodeTokenMissing:
		return status.Error(codes.Unauthenticated, sdkErr.Message)
	case ErrCodeTokenInvalid, ErrCodeTokenExpired:
		return status.Error(codes.Unauthenticated, sdkErr.Message)
	case ErrCodeValidation:
		return status.Error(codes.InvalidArgument, sdkErr.Message)
	case ErrCodeNetwork:
		return status.Error(codes.Unavailable, sdkErr.Message)
	case ErrCodeRateLimited:
		return status.Error(codes.ResourceExhausted, sdkErr.Message)
	default:
		return status.Error(codes.Internal, sdkErr.Message)
	}
}

// ShouldSkipMethod checks if a method should skip authentication.
func ShouldSkipMethod(method string, skipMethods []string) bool {
	for _, skip := range skipMethods {
		if method == skip || strings.HasSuffix(method, skip) {
			return true
		}
	}
	return false
}


// UnaryClientInterceptor returns a gRPC unary client interceptor that adds auth token.
func (c *Client) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		token, err := c.GetAccessToken(ctx)
		if err != nil {
			return status.Error(codes.Unauthenticated, "failed to get token: "+err.Error())
		}

		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor returns a gRPC stream client interceptor that adds auth token.
func (c *Client) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		token, err := c.GetAccessToken(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "failed to get token: "+err.Error())
		}

		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// wrappedServerStream wraps a grpc.ServerStream with a custom context.
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
