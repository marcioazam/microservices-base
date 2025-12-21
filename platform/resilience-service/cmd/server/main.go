// Package main is the entry point for the modernized resilience service.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/auth-platform/platform/resilience-service/internal/application"
	"github.com/auth-platform/platform/resilience-service/internal/domain/interfaces"
	"github.com/auth-platform/platform/resilience-service/internal/domain/validators"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/config"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/observability"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/repositories"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/resilience"
	"github.com/auth-platform/platform/resilience-service/internal/presentation/grpc"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

func main() {
	// Create fx application with all modules
	app := fx.New(
		// Configuration
		fx.Provide(config.Load),
		
		// Logging
		fx.Provide(NewLogger),
		
		// Observability
		fx.Provide(
			observability.GetTracer,
			observability.GetMeter,
			NewOTelEmitter,
		),
		
		// Infrastructure
		fx.Provide(
			NewRedisRepository,
			NewFailsafeExecutor,
			NewHealthCheckers,
			NewPolicyValidator,
		),
		
		// Application services
		application.Module,
		
		// Presentation layer
		fx.Provide(grpc.NewServer),
		
		// Lifecycle management
		fx.Invoke(grpc.RegisterWithFx),
		fx.Invoke(SetupObservability),
	)

	// Run the application
	app.Run()
}

// Provider functions for fx dependency injection

// NewLogger creates a structured logger based on configuration.
func NewLogger(cfg *config.Config) *slog.Logger {
	var handler slog.Handler
	
	opts := &slog.HandlerOptions{
		Level: parseLogLevel(cfg.Logging.Level),
	}
	
	switch cfg.Logging.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	
	return slog.New(handler)
}

// NewOTelEmitter creates an OpenTelemetry event emitter.
func NewOTelEmitter(tracer trace.Tracer, logger *slog.Logger) (interfaces.EventEmitter, error) {
	meter := observability.GetMeter()
	return observability.NewOTelEmitter(tracer, meter, logger)
}

// NewRedisRepository creates a Redis-based policy repository.
func NewRedisRepository(cfg *config.Config, logger *slog.Logger) (interfaces.PolicyRepository, error) {
	metrics := observability.NewMetricsRecorder()
	return repositories.NewRedisRepository(&cfg.Redis, logger, metrics)
}

// NewFailsafeExecutor creates a failsafe-go based resilience executor.
func NewFailsafeExecutor(logger *slog.Logger) interfaces.ResilienceExecutor {
	metrics := observability.NewMetricsRecorder()
	return resilience.NewFailsafeExecutor(metrics, logger)
}

// NewHealthCheckers creates health checkers for the service.
func NewHealthCheckers() []interfaces.HealthChecker {
	// Return empty slice for now - would be populated with actual health checkers
	return []interfaces.HealthChecker{}
}

// NewPolicyValidator creates a policy validator.
func NewPolicyValidator() interfaces.PolicyValidator {
	return validators.NewPolicyValidator()
}

// SetupObservability configures OpenTelemetry.
func SetupObservability(lc fx.Lifecycle, cfg *config.Config, logger *slog.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			cleanup, err := observability.Setup(ctx, &cfg.OpenTelemetry, logger)
			if err != nil {
				return err
			}
			
			// Store cleanup function for shutdown
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					cleanup()
					return nil
				},
			})
			
			return nil
		},
	})
}

// parseLogLevel converts string log level to slog.Level.
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
