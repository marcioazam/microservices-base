// Package grpc provides gRPC server configuration and interceptors.
package grpc

import (
	"context"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/logging"
	"github.com/authcorp/libs/go/src/observability"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// InterceptorConfig holds configuration for interceptors.
type InterceptorConfig struct {
	Logger         *logging.Logger
	EnableRecovery bool
	EnableLogging  bool
	EnableMetrics  bool
}

// LoggingInterceptor creates a unary logging interceptor.
func LoggingInterceptor(logger *logging.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Extract or generate correlation ID
		correlationID := extractCorrelationID(ctx)
		ctx = observability.WithCorrelationID(ctx, correlationID)

		// Log request
		if logger != nil {
			logger.Debug(ctx, "grpc request started",
				logging.String("method", info.FullMethod),
				logging.String("correlation_id", correlationID))
		}

		// Call handler
		resp, err := handler(ctx, req)

		// Log response
		duration := time.Since(start)
		if logger != nil {
			fields := []logging.Field{
				logging.String("method", info.FullMethod),
				logging.String("correlation_id", correlationID),
				logging.Duration("duration", duration),
			}

			if err != nil {
				fields = append(fields, logging.Error(err))
				logger.Error(ctx, "grpc request failed", fields...)
			} else {
				logger.Info(ctx, "grpc request completed", fields...)
			}
		}

		return resp, err
	}
}

// RecoveryInterceptor creates a unary recovery interceptor.
func RecoveryInterceptor(logger *logging.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				if logger != nil {
					logger.Error(ctx, "grpc panic recovered",
						logging.String("method", info.FullMethod),
						logging.Any("panic", r))
				}
				err = status.Error(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// MetricsInterceptor creates a unary metrics interceptor.
func MetricsInterceptor(metrics *Metrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}

		if metrics != nil {
			metrics.RecordRequest(info.FullMethod, code.String(), duration)
		}

		return resp, err
	}
}

// ChainUnaryInterceptors chains multiple unary interceptors.
func ChainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			next := chain
			chain = func(ctx context.Context, req interface{}) (interface{}, error) {
				return interceptor(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
					return next(ctx, req)
				})
			}
		}
		return chain(ctx, req)
	}
}

func extractCorrelationID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("x-correlation-id"); len(values) > 0 {
			return values[0]
		}
		if values := md.Get("x-request-id"); len(values) > 0 {
			return values[0]
		}
	}
	return uuid.New().String()
}

// Metrics holds gRPC metrics.
type Metrics struct {
	requestCount    map[string]int64
	requestDuration map[string]time.Duration
}

// NewMetrics creates a new metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		requestCount:    make(map[string]int64),
		requestDuration: make(map[string]time.Duration),
	}
}

// RecordRequest records a request metric.
func (m *Metrics) RecordRequest(method, code string, duration time.Duration) {
	key := method + ":" + code
	m.requestCount[key]++
	m.requestDuration[key] += duration
}
