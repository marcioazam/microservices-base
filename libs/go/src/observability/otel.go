// Package observability provides OpenTelemetry integration for tracing.
package observability

import (
	"context"
)

// Span represents a trace span for observability.
type Span interface {
	End()
	SetAttribute(key string, value any)
	SetStatus(code SpanStatusCode, description string)
	RecordError(err error)
	AddEvent(name string, attrs ...SpanAttribute)
}

// SpanStatusCode represents span status.
type SpanStatusCode int

const (
	SpanStatusUnset SpanStatusCode = iota
	SpanStatusOK
	SpanStatusError
)

// SpanAttribute represents a span attribute.
type SpanAttribute struct {
	Key   string
	Value any
}

// Attr creates a span attribute.
func Attr(key string, value any) SpanAttribute {
	return SpanAttribute{Key: key, Value: value}
}

// Tracer provides tracing capabilities.
type Tracer interface {
	Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
}

// SpanOption configures span creation.
type SpanOption func(*spanConfig)

type spanConfig struct {
	attributes []SpanAttribute
	kind       SpanKind
}

// SpanKind represents the type of span.
type SpanKind int

const (
	SpanKindInternal SpanKind = iota
	SpanKindServer
	SpanKindClient
	SpanKindProducer
	SpanKindConsumer
)

// WithSpanAttributes adds attributes to span.
func WithSpanAttributes(attrs ...SpanAttribute) SpanOption {
	return func(c *spanConfig) {
		c.attributes = append(c.attributes, attrs...)
	}
}

// WithSpanKind sets the span kind.
func WithSpanKind(kind SpanKind) SpanOption {
	return func(c *spanConfig) {
		c.kind = kind
	}
}

// noopSpan is a no-op span implementation.
type noopSpan struct{}

func (s *noopSpan) End()                                            {}
func (s *noopSpan) SetAttribute(key string, value any)              {}
func (s *noopSpan) SetStatus(code SpanStatusCode, description string) {}
func (s *noopSpan) RecordError(err error)                           {}
func (s *noopSpan) AddEvent(name string, attrs ...SpanAttribute)    {}

// noopTracer is a no-op tracer implementation.
type noopTracer struct{}

// NewNoopTracer creates a no-op tracer for testing.
func NewNoopTracer() Tracer {
	return &noopTracer{}
}

func (t *noopTracer) Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	return ctx, &noopSpan{}
}

// TracerProvider manages tracer instances.
type TracerProvider interface {
	Tracer(name string) Tracer
	Shutdown(ctx context.Context) error
}

// noopTracerProvider is a no-op tracer provider.
type noopTracerProvider struct{}

// NewNoopTracerProvider creates a no-op tracer provider.
func NewNoopTracerProvider() TracerProvider {
	return &noopTracerProvider{}
}

func (p *noopTracerProvider) Tracer(name string) Tracer {
	return &noopTracer{}
}

func (p *noopTracerProvider) Shutdown(ctx context.Context) error {
	return nil
}

// globalTracerProvider holds the global tracer provider.
var globalTracerProvider TracerProvider = &noopTracerProvider{}

// SetTracerProvider sets the global tracer provider.
func SetTracerProvider(tp TracerProvider) {
	globalTracerProvider = tp
}

// GetTracerProvider returns the global tracer provider.
func GetTracerProvider() TracerProvider {
	return globalTracerProvider
}

// GetTracer returns a tracer from the global provider.
func GetTracer(name string) Tracer {
	return globalTracerProvider.Tracer(name)
}

// StartSpan starts a new span using the global tracer.
func StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	return GetTracer("default").Start(ctx, name, opts...)
}

// SpanFromContext extracts span info from context for logging.
func SpanFromContext(ctx context.Context) (traceID, spanID string) {
	return TraceContextFromContext(ctx)
}
