// Package services provides application services that orchestrate domain operations.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/authcorp/libs/go/src/fault"
	"github.com/auth-platform/platform/resilience-service/internal/domain/interfaces"
	"go.opentelemetry.io/otel/trace"
)

// ResilienceService orchestrates resilience operations.
type ResilienceService struct {
	executor interfaces.ResilienceExecutor
	metrics  interfaces.MetricsRecorder
	logger   *slog.Logger
	tracer   trace.Tracer
}

// NewResilienceService creates a new resilience service.
func NewResilienceService(
	executor interfaces.ResilienceExecutor,
	metrics interfaces.MetricsRecorder,
	logger *slog.Logger,
	tracer trace.Tracer,
) *ResilienceService {
	return &ResilienceService{
		executor: executor,
		metrics:  metrics,
		logger:   logger,
		tracer:   tracer,
	}
}

// Execute executes an operation with resilience patterns applied.
func (s *ResilienceService) Execute(ctx context.Context, policyName string, operation func() error) error {
	ctx, span := s.tracer.Start(ctx, "resilience.execute")
	defer span.End()

	start := time.Now()
	
	s.logger.InfoContext(ctx, "executing operation with resilience policy",
		slog.String("policy_name", policyName))

	err := s.executor.Execute(ctx, policyName, operation)
	
	duration := time.Since(start)
	success := err == nil

	// Record metrics using shared type
	metrics := fault.NewExecutionMetrics(policyName, duration, success)
	s.metrics.RecordExecution(ctx, metrics)

	if err != nil {
		s.logger.ErrorContext(ctx, "operation failed",
			slog.String("policy_name", policyName),
			slog.Duration("duration", duration),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return fmt.Errorf("resilience execution failed: %w", err)
	}

	s.logger.InfoContext(ctx, "operation completed successfully",
		slog.String("policy_name", policyName),
		slog.Duration("duration", duration))

	return nil
}

// ExecuteWithResult executes an operation with resilience patterns and returns a result.
func (s *ResilienceService) ExecuteWithResult(ctx context.Context, policyName string, operation func() (any, error)) (any, error) {
	ctx, span := s.tracer.Start(ctx, "resilience.execute_with_result")
	defer span.End()

	start := time.Now()
	
	s.logger.InfoContext(ctx, "executing operation with result and resilience policy",
		slog.String("policy_name", policyName))

	result := s.executor.ExecuteWithResult(ctx, policyName, operation)
	
	duration := time.Since(start)
	success := result.IsOk()

	// Record metrics using shared type
	metrics := fault.NewExecutionMetrics(policyName, duration, success)
	s.metrics.RecordExecution(ctx, metrics)

	if result.IsErr() {
		err := result.UnwrapErr()
		s.logger.ErrorContext(ctx, "operation with result failed",
			slog.String("policy_name", policyName),
			slog.Duration("duration", duration),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return nil, fmt.Errorf("resilience execution with result failed: %w", err)
	}

	s.logger.InfoContext(ctx, "operation with result completed successfully",
		slog.String("policy_name", policyName),
		slog.Duration("duration", duration))

	return result.Unwrap(), nil
}

// GetExecutionMetrics returns execution metrics for monitoring using shared type.
func (s *ResilienceService) GetExecutionMetrics(ctx context.Context, policyName string) (fault.ExecutionMetrics, error) {
	ctx, span := s.tracer.Start(ctx, "resilience.get_metrics")
	defer span.End()

	return fault.NewExecutionMetrics(policyName, 0, true), nil
}