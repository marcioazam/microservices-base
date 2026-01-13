package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracer wraps OpenTelemetry tracer with cache-specific functionality.
type Tracer struct {
	tracer trace.Tracer
}

// NewTracer creates a new Tracer instance.
func NewTracer(name string) *Tracer {
	return &Tracer{
		tracer: otel.Tracer(name),
	}
}

// StartSpan starts a new span with the given name.
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// StartCacheOperation starts a span for a cache operation.
func (t *Tracer) StartCacheOperation(ctx context.Context, operation, key, namespace string) (context.Context, trace.Span) {
	ctx, span := t.tracer.Start(ctx, "cache."+operation,
		trace.WithAttributes(
			attribute.String("cache.operation", operation),
			attribute.String("cache.key", key),
			attribute.String("cache.namespace", namespace),
		),
	)

	// Add correlation ID if present
	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		span.SetAttributes(attribute.String("correlation_id", correlationID))
	}

	return ctx, span
}

// StartBatchOperation starts a span for a batch cache operation.
func (t *Tracer) StartBatchOperation(ctx context.Context, operation string, keyCount int, namespace string) (context.Context, trace.Span) {
	ctx, span := t.tracer.Start(ctx, "cache.batch."+operation,
		trace.WithAttributes(
			attribute.String("cache.operation", operation),
			attribute.Int("cache.key_count", keyCount),
			attribute.String("cache.namespace", namespace),
		),
	)

	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		span.SetAttributes(attribute.String("correlation_id", correlationID))
	}

	return ctx, span
}

// RecordCacheHit records a cache hit event on the span.
func RecordCacheHit(span trace.Span, source string) {
	span.SetAttributes(
		attribute.Bool("cache.hit", true),
		attribute.String("cache.source", source),
	)
}

// RecordCacheMiss records a cache miss event on the span.
func RecordCacheMiss(span trace.Span) {
	span.SetAttributes(attribute.Bool("cache.hit", false))
}

// RecordError records an error on the span.
func RecordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetSuccess marks the span as successful.
func SetSuccess(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}

// AddAttribute adds a custom attribute to the span.
func AddAttribute(span trace.Span, key string, value interface{}) {
	switch v := value.(type) {
	case string:
		span.SetAttributes(attribute.String(key, v))
	case int:
		span.SetAttributes(attribute.Int(key, v))
	case int64:
		span.SetAttributes(attribute.Int64(key, v))
	case float64:
		span.SetAttributes(attribute.Float64(key, v))
	case bool:
		span.SetAttributes(attribute.Bool(key, v))
	}
}
