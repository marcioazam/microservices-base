// Package grpc provides health check server implementation.
package grpc

import (
	"context"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/application/services"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// HealthServer implements gRPC health check service.
type HealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	healthService *services.HealthService
}

// NewHealthServer creates a new health check server.
func NewHealthServer(healthService *services.HealthService) *HealthServer {
	return &HealthServer{
		healthService: healthService,
	}
}

// Check performs a health check.
func (h *HealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	// Get aggregated health status
	healthStatus, err := h.healthService.GetAggregatedHealth(ctx)
	if err != nil {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}

	// Convert domain health status to gRPC status
	grpcStatus := h.convertHealthStatus(healthStatus.Status)

	return &grpc_health_v1.HealthCheckResponse{
		Status: grpcStatus,
	}, nil
}

// Watch performs a streaming health check.
func (h *HealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	// Send initial status
	resp, err := h.Check(stream.Context(), req)
	if err != nil {
		return err
	}

	if err := stream.Send(resp); err != nil {
		return err
	}

	// Continue sending periodic updates
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			resp, err := h.Check(stream.Context(), req)
			if err != nil {
				return status.Errorf(codes.Internal, "health check failed: %v", err)
			}

			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}

// convertHealthStatus converts domain health status to gRPC health status.
func (h *HealthServer) convertHealthStatus(status valueobjects.HealthState) grpc_health_v1.HealthCheckResponse_ServingStatus {
	switch status {
	case valueobjects.HealthHealthy:
		return grpc_health_v1.HealthCheckResponse_SERVING
	case valueobjects.HealthDegraded:
		return grpc_health_v1.HealthCheckResponse_SERVING // Still serving but degraded
	case valueobjects.HealthUnhealthy:
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING
	case valueobjects.HealthUnknown:
		return grpc_health_v1.HealthCheckResponse_UNKNOWN
	default:
		return grpc_health_v1.HealthCheckResponse_UNKNOWN
	}
}