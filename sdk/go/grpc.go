package authplatform

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor for token validation.
func (c *Client) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		token, err := extractTokenFromMetadata(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		claims, err := c.ValidateToken(ctx, token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token: "+err.Error())
		}

		ctx = context.WithValue(ctx, ClaimsContextKey, claims)
		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for token validation.
func (c *Client) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		token, err := extractTokenFromMetadata(ctx)
		if err != nil {
			return status.Error(codes.Unauthenticated, err.Error())
		}

		claims, err := c.ValidateToken(ctx, token)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid token: "+err.Error())
		}

		// Wrap the stream with authenticated context
		wrapped := &wrappedServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(ctx, ClaimsContextKey, claims),
		}

		return handler(srv, wrapped)
	}
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

func extractTokenFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	auth := values[0]
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", status.Error(codes.Unauthenticated, "invalid authorization format")
	}

	return parts[1], nil
}

// wrappedServerStream wraps a grpc.ServerStream with a custom context.
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
