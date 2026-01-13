package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/auth-platform/file-upload"

// Tracer wraps OpenTelemetry tracer
type Tracer struct {
	tracer trace.Tracer
}

// NewTracer creates a new tracer
func NewTracer() *Tracer {
	return &Tracer{
		tracer: otel.Tracer(tracerName),
	}
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// StartSpanWithAttributes starts a new span with attributes
func (t *Tracer) StartSpanWithAttributes(ctx context.Context, name string, attrs map[string]string) (context.Context, trace.Span) {
	kvs := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		kvs = append(kvs, attribute.String(k, v))
	}
	return t.tracer.Start(ctx, name, trace.WithAttributes(kvs...))
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs map[string]string) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}
	
	kvs := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		kvs = append(kvs, attribute.String(k, v))
	}
	span.AddEvent(name, trace.WithAttributes(kvs...))
}

// SetError marks the span as error
func SetError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}
	span.RecordError(err)
}

// SetAttribute sets an attribute on the current span
func SetAttribute(ctx context.Context, key string, value string) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}
	span.SetAttributes(attribute.String(key, value))
}

// GetTraceID returns the trace ID from context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// GetSpanID returns the span ID from context
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	return span.SpanContext().SpanID().String()
}
