// Package services provides health aggregation services.
package services

import (
	"context"
	"log/slog"
	"sync"

	"github.com/auth-platform/platform/resilience-service/internal/domain/interfaces"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
	"go.opentelemetry.io/otel/trace"
)

// HealthService aggregates health status from multiple components.
type HealthService struct {
	checkers []interfaces.HealthChecker
	logger   *slog.Logger
	tracer   trace.Tracer
	mu       sync.RWMutex
}

// NewHealthService creates a new health service.
func NewHealthService(
	checkers []interfaces.HealthChecker,
	logger *slog.Logger,
	tracer trace.Tracer,
) *HealthService {
	return &HealthService{
		checkers: checkers,
		logger:   logger,
		tracer:   tracer,
	}
}

// AddChecker adds a health checker to the service.
func (s *HealthService) AddChecker(checker interfaces.HealthChecker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.checkers = append(s.checkers, checker)
	s.logger.InfoContext(context.Background(), "health checker added",
		slog.String("checker_name", checker.Name()))
}

// RemoveChecker removes a health checker by name.
func (s *HealthService) RemoveChecker(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for i, checker := range s.checkers {
		if checker.Name() == name {
			s.checkers = append(s.checkers[:i], s.checkers[i+1:]...)
			s.logger.InfoContext(context.Background(), "health checker removed",
				slog.String("checker_name", name))
			return
		}
	}
}

// GetAggregatedHealth returns the aggregated health status of all components.
func (s *HealthService) GetAggregatedHealth(ctx context.Context) (valueobjects.HealthStatus, error) {
	ctx, span := s.tracer.Start(ctx, "health.aggregate")
	defer span.End()

	s.mu.RLock()
	checkers := make([]interfaces.HealthChecker, len(s.checkers))
	copy(checkers, s.checkers)
	s.mu.RUnlock()

	if len(checkers) == 0 {
		return valueobjects.NewHealthStatus(valueobjects.HealthUnknown, "no health checkers configured"), nil
	}

	// Check all components concurrently
	type checkResult struct {
		name   string
		status valueobjects.HealthStatus
	}

	results := make(chan checkResult, len(checkers))
	var wg sync.WaitGroup

	for _, checker := range checkers {
		wg.Add(1)
		go func(c interfaces.HealthChecker) {
			defer wg.Done()
			status := c.Check(ctx)
			results <- checkResult{name: c.Name(), status: status}
		}(checker)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Aggregate results
	overallState := valueobjects.HealthHealthy
	componentStatuses := make(map[string]valueobjects.HealthStatus)
	unhealthyComponents := []string{}

	for result := range results {
		componentStatuses[result.name] = result.status
		
		if !result.status.IsHealthy() {
			unhealthyComponents = append(unhealthyComponents, result.name)
			
			// If any component is unhealthy, overall status is unhealthy
			if result.status.Status == valueobjects.HealthUnhealthy {
				overallState = valueobjects.HealthUnhealthy
			} else if overallState == valueobjects.HealthHealthy && result.status.Status == valueobjects.HealthDegraded {
				overallState = valueobjects.HealthDegraded
			}
		}
	}

	// Create aggregated health status
	var message string
	switch overallState {
	case valueobjects.HealthHealthy:
		message = "all components healthy"
	case valueobjects.HealthDegraded:
		message = "some components degraded"
	case valueobjects.HealthUnhealthy:
		message = "some components unhealthy"
	default:
		message = "unknown health state"
	}

	aggregatedHealth := valueobjects.NewHealthStatus(overallState, message).
		WithDetail("component_count", len(checkers)).
		WithDetail("healthy_count", len(checkers)-len(unhealthyComponents)).
		WithDetail("unhealthy_count", len(unhealthyComponents))

	if len(unhealthyComponents) > 0 {
		aggregatedHealth = aggregatedHealth.WithDetail("unhealthy_components", unhealthyComponents)
	}

	// Add individual component statuses
	for name, status := range componentStatuses {
		aggregatedHealth = aggregatedHealth.WithDetail("component_"+name, map[string]any{
			"status":  status.Status,
			"message": status.Message,
		})
	}

	s.logger.InfoContext(ctx, "health check completed",
		slog.String("overall_status", string(overallState)),
		slog.Int("total_components", len(checkers)),
		slog.Int("unhealthy_components", len(unhealthyComponents)))

	return aggregatedHealth, nil
}

// GetComponentHealth returns the health status of a specific component.
func (s *HealthService) GetComponentHealth(ctx context.Context, componentName string) (valueobjects.HealthStatus, error) {
	ctx, span := s.tracer.Start(ctx, "health.component")
	defer span.End()

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, checker := range s.checkers {
		if checker.Name() == componentName {
			status := checker.Check(ctx)
			
			s.logger.DebugContext(ctx, "component health checked",
				slog.String("component", componentName),
				slog.String("status", string(status.Status)))
			
			return status, nil
		}
	}

	return valueobjects.NewHealthStatus(valueobjects.HealthUnknown, "component not found"), nil
}

// ListComponents returns the names of all registered health checkers.
func (s *HealthService) ListComponents() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, len(s.checkers))
	for i, checker := range s.checkers {
		names[i] = checker.Name()
	}

	return names
}