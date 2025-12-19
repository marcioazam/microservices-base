// Package otel provides OpenTelemetry integration.
package otel

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Provider manages OpenTelemetry configuration.
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
	serviceName    string
	// instrumentationAttrs holds concurrent-safe instrumentation attributes
	instrumentationAttrs []attribute.KeyValue
}

// Config holds OpenTelemetry configuration.
type Config struct {
	ServiceName string
	Endpoint    string
	Insecure    bool
}

// NewProvider creates a new OpenTelemetry provider.
func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
	// Create OTLP exporter
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	// Create resource with instrumentation attributes for concurrent-safe handling
	instrumentationAttrs := []attribute.KeyValue{
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion("1.0.0"),
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			instrumentationAttrs...,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	// Create tracer provider with optimized settings for Go 1.25+
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for W3C Trace Context
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{
		tracerProvider:       tp,
		tracer:               tp.Tracer(cfg.ServiceName),
		serviceName:          cfg.ServiceName,
		instrumentationAttrs: instrumentationAttrs,
	}, nil
}

// Shutdown gracefully shuts down the provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil || p.tracerProvider == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.tracerProvider.Shutdown(ctx)
}

// Tracer returns the tracer.
func (p *Provider) Tracer() trace.Tracer {
	if p == nil {
		return nil
	}
	return p.tracer
}

// StartSpan starts a new span with instrumentation attributes.
func (p *Provider) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if p == nil || p.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return p.tracer.Start(ctx, name, opts...)
}

// StartSpanWithAttributes starts a new span with additional attributes.
func (p *Provider) StartSpanWithAttributes(ctx context.Context, name string, attrs []attribute.KeyValue, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if p == nil || p.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	// Combine instrumentation attributes with provided attributes
	allAttrs := make([]attribute.KeyValue, 0, len(p.instrumentationAttrs)+len(attrs))
	allAttrs = append(allAttrs, p.instrumentationAttrs...)
	allAttrs = append(allAttrs, attrs...)

	opts = append(opts, trace.WithAttributes(allAttrs...))
	return p.tracer.Start(ctx, name, opts...)
}

// GetInstrumentationAttributes returns the instrumentation attributes for concurrent-safe access.
func (p *Provider) GetInstrumentationAttributes() []attribute.KeyValue {
	if p == nil {
		return nil
	}
	// Return a copy to ensure concurrent safety
	result := make([]attribute.KeyValue, len(p.instrumentationAttrs))
	copy(result, p.instrumentationAttrs)
	return result
}

// SpanFromContext returns the span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddEvent adds an event to the current span.
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetAttributes sets attributes on the current span.
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span.
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// GetTraceID returns the trace ID from context.
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from context.
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// ExtractTraceContext extracts trace context from the current span.
func ExtractTraceContext(ctx context.Context) (traceID, spanID string) {
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()

	if spanCtx.HasTraceID() {
		traceID = spanCtx.TraceID().String()
	}
	if spanCtx.HasSpanID() {
		spanID = spanCtx.SpanID().String()
	}

	return traceID, spanID
}
