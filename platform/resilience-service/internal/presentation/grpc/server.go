// Package grpc provides gRPC server implementation with modern middleware.
package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/application/services"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/config"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Server represents the gRPC server with modern middleware.
type Server struct {
	server           *grpc.Server
	listener         net.Listener
	config           *config.ServerConfig
	logger           *slog.Logger
	tracer           trace.Tracer
	resilienceService *services.ResilienceService
	policyService    *services.PolicyService
	healthService    *services.HealthService
}

// NewServer creates a new gRPC server with middleware chain.
func NewServer(
	cfg *config.ServerConfig,
	logger *slog.Logger,
	tracer trace.Tracer,
	resilienceService *services.ResilienceService,
	policyService *services.PolicyService,
	healthService *services.HealthService,
) (*Server, error) {
	// Create listener
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Setup logging interceptor
	loggingOpts := []grpc_logging.Option{
		grpc_logging.WithLogOnEvents(grpc_logging.StartCall, grpc_logging.FinishCall),
	}

	// Setup recovery interceptor
	recoveryOpts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(func(p any) (err error) {
			logger.Error("gRPC panic recovered", slog.Any("panic", p))
			return status.Errorf(codes.Internal, "internal server error")
		}),
	}

	// Auth interceptor will be added later when needed

	// Create interceptor chain
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		grpc_recovery.UnaryServerInterceptor(recoveryOpts...),
		grpc_logging.UnaryServerInterceptor(InterceptorLogger(logger), loggingOpts...),
		// Skip auth interceptor for now - will be added later
		// grpc_auth.UnaryServerInterceptor(authFunc),
		TracingUnaryInterceptor(tracer),
		MetricsUnaryInterceptor(logger),
	}

	streamInterceptors := []grpc.StreamServerInterceptor{
		grpc_recovery.StreamServerInterceptor(recoveryOpts...),
		grpc_logging.StreamServerInterceptor(InterceptorLogger(logger), loggingOpts...),
		// Skip auth interceptor for now - will be added later
		// grpc_auth.StreamServerInterceptor(authFunc),
		TracingStreamInterceptor(tracer),
		MetricsStreamInterceptor(logger),
	}

	// Create gRPC server with options
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.MaxSendMsgSize),
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
	}

	server := grpc.NewServer(opts...)

	// Register health service
	healthServer := NewHealthServer(healthService)
	grpc_health_v1.RegisterHealthServer(server, healthServer)

	// Register resilience service
	// Note: This would require the actual protobuf service definition
	// For now, we'll skip the service registration

	// Enable reflection in non-production environments
	if cfg.Host != "0.0.0.0" || cfg.Port != 50056 { // Simple heuristic for non-prod
		reflection.Register(server)
		logger.Info("gRPC reflection enabled")
	}

	return &Server{
		server:           server,
		listener:         listener,
		config:           cfg,
		logger:           logger,
		tracer:           tracer,
		resilienceService: resilienceService,
		policyService:    policyService,
		healthService:    healthService,
	}, nil
}

// Start starts the gRPC server.
func (s *Server) Start() error {
	s.logger.Info("starting gRPC server",
		slog.String("address", s.listener.Addr().String()),
		slog.Int("max_recv_msg_size", s.config.MaxRecvMsgSize),
		slog.Int("max_send_msg_size", s.config.MaxSendMsgSize))

	return s.server.Serve(s.listener)
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping gRPC server")

	// Create a channel to signal when graceful stop is complete
	stopped := make(chan struct{})
	
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful stop or timeout
	select {
	case <-stopped:
		s.logger.Info("gRPC server stopped gracefully")
		return nil
	case <-ctx.Done():
		s.logger.Warn("gRPC server graceful stop timeout, forcing stop")
		s.server.Stop()
		return ctx.Err()
	}
}

// RegisterWithFx registers the server with fx lifecycle.
func RegisterWithFx(lc fx.Lifecycle, server *Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := server.Start(); err != nil {
					server.logger.Error("gRPC server error", slog.String("error", err.Error()))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Stop(ctx)
		},
	})
}

// InterceptorLogger adapts slog.Logger to grpc_logging.Logger interface.
func InterceptorLogger(l *slog.Logger) grpc_logging.Logger {
	return grpc_logging.LoggerFunc(func(ctx context.Context, lvl grpc_logging.Level, msg string, fields ...any) {
		switch lvl {
		case grpc_logging.LevelDebug:
			l.DebugContext(ctx, msg, fields...)
		case grpc_logging.LevelInfo:
			l.InfoContext(ctx, msg, fields...)
		case grpc_logging.LevelWarn:
			l.WarnContext(ctx, msg, fields...)
		case grpc_logging.LevelError:
			l.ErrorContext(ctx, msg, fields...)
		default:
			l.InfoContext(ctx, msg, fields...)
		}
	})
}

// TracingUnaryInterceptor adds distributed tracing to unary RPCs.
func TracingUnaryInterceptor(tracer trace.Tracer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, span := tracer.Start(ctx, info.FullMethod)
		defer span.End()

		resp, err := handler(ctx, req)
		if err != nil {
			span.RecordError(err)
		}

		return resp, err
	}
}

// TracingStreamInterceptor adds distributed tracing to streaming RPCs.
func TracingStreamInterceptor(tracer trace.Tracer) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, span := tracer.Start(ss.Context(), info.FullMethod)
		defer span.End()

		// Wrap the server stream to use the traced context
		wrapped := &tracedServerStream{ServerStream: ss, ctx: ctx}
		
		err := handler(srv, wrapped)
		if err != nil {
			span.RecordError(err)
		}

		return err
	}
}

// MetricsUnaryInterceptor adds metrics collection to unary RPCs.
func MetricsUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		
		resp, err := handler(ctx, req)
		
		duration := time.Since(start)
		
		// Log metrics
		logger.DebugContext(ctx, "gRPC unary call completed",
			slog.String("method", info.FullMethod),
			slog.Duration("duration", duration),
			slog.Bool("success", err == nil))

		return resp, err
	}
}

// MetricsStreamInterceptor adds metrics collection to streaming RPCs.
func MetricsStreamInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		
		err := handler(srv, ss)
		
		duration := time.Since(start)
		
		// Log metrics
		logger.DebugContext(ss.Context(), "gRPC stream call completed",
			slog.String("method", info.FullMethod),
			slog.Duration("duration", duration),
			slog.Bool("success", err == nil))

		return err
	}
}

// tracedServerStream wraps grpc.ServerStream to provide traced context.
type tracedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *tracedServerStream) Context() context.Context {
	return s.ctx
}