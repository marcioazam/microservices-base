// Package main is the entry point for the resilience service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/config"
	"github.com/auth-platform/platform/resilience-service/internal/health"
	"github.com/auth-platform/platform/resilience-service/internal/policy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg := config.NewDefaultConfig()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize components
	healthAgg := health.NewAggregator(health.Config{})

	policyEngine := policy.NewEngine(policy.Config{
		ConfigPath:     cfg.Policy.ConfigPath,
		ReloadInterval: cfg.Policy.ReloadInterval,
	})

	// Start policy hot-reload if configured
	if cfg.Policy.ConfigPath != "" {
		if err := policyEngine.StartHotReload(ctx); err != nil {
			logger.Warn("failed to start policy hot-reload", slog.String("error", err.Error()))
		}
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register health check service
	healthServer := &healthCheckServer{aggregator: healthAgg}
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// Enable reflection for debugging
	reflection.Register(grpcServer)

	// Start listening
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting gRPC server", slog.String("address", addr))
		if err := grpcServer.Serve(listener); err != nil {
			logger.Error("gRPC server error", slog.String("error", err.Error()))
		}
	}()

	// Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	<-shutdown
	logger.Info("shutdown signal received")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Stop accepting new connections
	grpcServer.GracefulStop()

	// Stop policy engine
	policyEngine.Stop()

	// Wait for shutdown context or timeout
	select {
	case <-shutdownCtx.Done():
		logger.Warn("shutdown timeout exceeded, forcing shutdown")
		grpcServer.Stop()
	default:
	}

	logger.Info("server shutdown complete")
}

// healthCheckServer implements gRPC health check.
type healthCheckServer struct {
	grpc_health_v1.UnimplementedHealthServer
	aggregator *health.Aggregator
}

func (s *healthCheckServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	health, err := s.aggregator.GetAggregatedHealth(ctx)
	if err != nil {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}

	status := grpc_health_v1.HealthCheckResponse_SERVING
	if health.Status == "unhealthy" {
		status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: status,
	}, nil
}

func (s *healthCheckServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	// Send initial status
	resp, _ := s.Check(stream.Context(), req)
	if err := stream.Send(resp); err != nil {
		return err
	}

	// Keep connection open
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			resp, _ := s.Check(stream.Context(), req)
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}
