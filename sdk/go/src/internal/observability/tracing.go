package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/auth-platform/sdk-go"

// Tracer provides OpenTelemetry tracing for the SDK.
type Tracer struct {
	tracer trace.Tracer
}

// NewTracer creates a new tracer.
func NewTracer() *Tracer {
	return &Tracer{
		tracer: otel.Tracer(tracerName),
	}
}

// NewTracerWithName creates a tracer with a custom name.
func NewTracerWithName(name string) *Tracer {
	return &Tracer{
		tracer: otel.Tracer(name),
	}
}

// StartSpan starts a new span.
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// SetSpanError marks a span as errored.
func SetSpanError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanOK marks a span as successful.
func SetSpanOK(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}

// AddSpanAttributes adds attributes to a span.
func AddSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	span.SetAttributes(attrs...)
}

// TraceTokenValidation creates a span for token validation.
func (t *Tracer) TraceTokenValidation(ctx context.Context) (context.Context, trace.Span) {
	return t.StartSpan(ctx, "auth.validate_token",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
}

// TraceTokenExtraction creates a span for token extraction.
func (t *Tracer) TraceTokenExtraction(ctx context.Context) (context.Context, trace.Span) {
	return t.StartSpan(ctx, "auth.extract_token",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
}

// TraceJWKSFetch creates a span for JWKS fetching.
func (t *Tracer) TraceJWKSFetch(ctx context.Context, uri string) (context.Context, trace.Span) {
	ctx, span := t.StartSpan(ctx, "auth.fetch_jwks",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	span.SetAttributes(attribute.String("jwks.uri", uri))
	return ctx, span
}

// TraceDPoPGeneration creates a span for DPoP proof generation.
func (t *Tracer) TraceDPoPGeneration(ctx context.Context) (context.Context, trace.Span) {
	return t.StartSpan(ctx, "auth.generate_dpop",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
}

// TraceRetry creates a span for retry operations.
func (t *Tracer) TraceRetry(ctx context.Context, attempt int) (context.Context, trace.Span) {
	ctx, span := t.StartSpan(ctx, "auth.retry",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	span.SetAttributes(attribute.Int("retry.attempt", attempt))
	return ctx, span
}
