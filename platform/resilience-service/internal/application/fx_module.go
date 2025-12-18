// Package application provides fx module for dependency injection.
package application

import (
	"log/slog"

	"github.com/auth-platform/platform/resilience-service/internal/application/services"
	"github.com/auth-platform/platform/resilience-service/internal/domain/interfaces"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

// Module provides application services for dependency injection.
var Module = fx.Module("application",
	fx.Provide(
		NewResilienceService,
		NewPolicyService,
		NewHealthService,
	),
)

// NewResilienceService creates a resilience service with injected dependencies.
func NewResilienceService(
	executor interfaces.ResilienceExecutor,
	metrics interfaces.MetricsRecorder,
	logger *slog.Logger,
	tracer trace.Tracer,
) *services.ResilienceService {
	return services.NewResilienceService(executor, metrics, logger, tracer)
}

// NewPolicyService creates a policy service with injected dependencies.
func NewPolicyService(
	repository interfaces.PolicyRepository,
	validator interfaces.PolicyValidator,
	emitter interfaces.EventEmitter,
	logger *slog.Logger,
	tracer trace.Tracer,
) *services.PolicyService {
	return services.NewPolicyService(repository, validator, emitter, logger, tracer)
}

// NewHealthService creates a health service with injected dependencies.
func NewHealthService(
	checkers []interfaces.HealthChecker,
	logger *slog.Logger,
	tracer trace.Tracer,
) *services.HealthService {
	return services.NewHealthService(checkers, logger, tracer)
}