package grpc

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

// UnaryErrorInterceptor converts resilience errors to gRPC errors.
func UnaryErrorInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return resp, ToGRPCError(err)
		}
		return resp, nil
	}
}

// StreamErrorInterceptor converts resilience errors for streams.
func StreamErrorInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		err := handler(srv, ss)
		if err != nil {
			return ToGRPCError(err)
		}
		return nil
	}
}

// UnaryLoggingInterceptor logs requests with correlation ID.
func UnaryLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		correlationID := getCorrelationID(ctx)

		resp, err := handler(ctx, req)

		logger.Info("grpc request",
			"method", info.FullMethod,
			"correlation_id", correlationID,
			"duration_ms", time.Since(start).Milliseconds(),
			"error", err != nil,
		)

		return resp, err
	}
}

// UnaryClientErrorInterceptor converts gRPC errors to resilience errors.
func UnaryClientErrorInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			return FromGRPCError(err)
		}
		return nil
	}
}

// StreamClientErrorInterceptor converts gRPC errors for client streams.
func StreamClientErrorInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return stream, FromGRPCError(err)
		}
		return stream, nil
	}
}

type correlationIDKey struct{}

// WithCorrelationID adds correlation ID to context.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey{}, id)
}

func getCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return id
	}
	return ""
}
