// Package observability provides OpenTelemetry setup and configuration.
package observability

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Setup configures OpenTelemetry with the provided configuration.
func Setup(ctx context.Context, cfg *config.OTelConfig, logger *slog.Logger) (func(), error) {
	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Setup tracing
	tracerProvider, err := setupTracing(ctx, cfg, res, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to setup tracing: %w", err)
	}

	// Set global tracer provider
	otel.SetTracerProvider(tracerProvider)

	// Setup text map propagator for W3C trace context
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("OpenTelemetry configured successfully",
		slog.String("service_name", cfg.ServiceName),
		slog.String("service_version", cfg.ServiceVersion),
		slog.String("environment", cfg.Environment),
		slog.String("endpoint", cfg.Endpoint))

	// Return cleanup function
	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shutdown tracer provider", slog.String("error", err.Error()))
		}
		
		logger.Info("OpenTelemetry shutdown completed")
	}, nil
}

// setupTracing configures the tracing pipeline.
func setupTracing(ctx context.Context, cfg *config.OTelConfig, res *resource.Resource, logger *slog.Logger) (*sdktrace.TracerProvider, error) {
	// Create gRPC connection options
	var opts []grpc.DialOption
	if cfg.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Add custom headers if configured
	if len(cfg.Headers) > 0 {
		// Convert headers to metadata
		// This would require additional implementation for header injection
	}

	// Create OTLP trace exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithDialOption(opts...),
		otlptracegrpc.WithTimeout(cfg.Timeout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	return tp, nil
}

// GetTracer returns a tracer for the resilience service.
func GetTracer() trace.Tracer {
	return otel.Tracer("resilience-service")
}

// GetMeter returns a meter for the resilience service.
func GetMeter() metric.Meter {
	return otel.Meter("resilience-service")
}

// StructuredLogger provides structured logging with correlation IDs.
type StructuredLogger struct {
	logger *slog.Logger
}

// NewStructuredLogger creates a new structured logger.
func NewStructuredLogger(logger *slog.Logger) *StructuredLogger {
	return &StructuredLogger{logger: logger}
}

// Debug logs a debug message with context.
func (s *StructuredLogger) Debug(ctx context.Context, msg string, fields map[string]any) {
	attrs := s.contextAttributes(ctx)
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	// Convert []slog.Attr to []any for variadic function
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	s.logger.DebugContext(ctx, msg, args...)
}

// Info logs an info message with context.
func (s *StructuredLogger) Info(ctx context.Context, msg string, fields map[string]any) {
	attrs := s.contextAttributes(ctx)
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	// Convert []slog.Attr to []any for variadic function
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	s.logger.InfoContext(ctx, msg, args...)
}

// Warn logs a warning message with context.
func (s *StructuredLogger) Warn(ctx context.Context, msg string, fields map[string]any) {
	attrs := s.contextAttributes(ctx)
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	// Convert []slog.Attr to []any for variadic function
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	s.logger.WarnContext(ctx, msg, args...)
}

// Error logs an error message with context.
func (s *StructuredLogger) Error(ctx context.Context, msg string, err error, fields map[string]any) {
	attrs := s.contextAttributes(ctx)
	attrs = append(attrs, slog.String("error", err.Error()))
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	// Convert []slog.Attr to []any for variadic function
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	s.logger.ErrorContext(ctx, msg, args...)
}

// contextAttributes extracts correlation ID and trace information from context.
func (s *StructuredLogger) contextAttributes(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr

	// Extract trace information
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}

	// Extract correlation ID from context (if available)
	if correlationID := ctx.Value("correlation_id"); correlationID != nil {
		if id, ok := correlationID.(string); ok {
			attrs = append(attrs, slog.String("correlation_id", id))
		}
	}

	return attrs
}

// DistributedTracer provides distributed tracing capabilities.
type DistributedTracer struct {
	tracer trace.Tracer
}

// NewDistributedTracer creates a new distributed tracer.
func NewDistributedTracer(tracer trace.Tracer) *DistributedTracer {
	return &DistributedTracer{tracer: tracer}
}

// StartSpan starts a new span with the given name.
func (d *DistributedTracer) StartSpan(ctx context.Context, name string) (context.Context, func()) {
	ctx, span := d.tracer.Start(ctx, name)
	return ctx, func() { span.End() }
}

// AddEvent adds an event to the current span.
func (d *DistributedTracer) AddEvent(ctx context.Context, name string, attributes map[string]any) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	// Add event without attributes for now
	// TODO: Implement proper attribute conversion
	span.AddEvent(name)
}

// RecordError records an error in the current span.
func (d *DistributedTracer) RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	span.RecordError(err)
}