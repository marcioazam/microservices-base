// Package observability provides tracing and metrics for IAM Policy Service.
package observability

import (
	"context"

	"github.com/authcorp/libs/go/src/observability"
)

// Tracer wraps the observability tracer for IAM Policy Service.
type Tracer struct {
	tracer observability.Tracer
	name   string
}

// NewTracer creates a new tracer.
func NewTracer(name string) *Tracer {
	return &Tracer{
		tracer: observability.GetTracer(name),
		name:   name,
	}
}

// StartSpan starts a new span.
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...observability.SpanOption) (context.Context, observability.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// StartAuthorizationSpan starts a span for authorization operations.
func (t *Tracer) StartAuthorizationSpan(ctx context.Context, subjectID, action, resourceType string) (context.Context, observability.Span) {
	return t.tracer.Start(ctx, "authorize",
		observability.WithSpanAttributes(
			observability.Attr("subject_id", subjectID),
			observability.Attr("action", action),
			observability.Attr("resource_type", resourceType),
		),
		observability.WithSpanKind(observability.SpanKindServer),
	)
}

// StartPolicyEvalSpan starts a span for policy evaluation.
func (t *Tracer) StartPolicyEvalSpan(ctx context.Context, policyCount int) (context.Context, observability.Span) {
	return t.tracer.Start(ctx, "policy_evaluation",
		observability.WithSpanAttributes(
			observability.Attr("policy_count", policyCount),
		),
		observability.WithSpanKind(observability.SpanKindInternal),
	)
}

// StartCacheSpan starts a span for cache operations.
func (t *Tracer) StartCacheSpan(ctx context.Context, operation string) (context.Context, observability.Span) {
	return t.tracer.Start(ctx, "cache_"+operation,
		observability.WithSpanAttributes(
			observability.Attr("cache_operation", operation),
		),
		observability.WithSpanKind(observability.SpanKindClient),
	)
}

// RecordError records an error on the current span.
func RecordError(span observability.Span, err error) {
	if span != nil && err != nil {
		span.RecordError(err)
		span.SetStatus(observability.SpanStatusError, err.Error())
	}
}

// SetSuccess marks the span as successful.
func SetSuccess(span observability.Span) {
	if span != nil {
		span.SetStatus(observability.SpanStatusOK, "")
	}
}

// AddEvent adds an event to the span.
func AddEvent(span observability.Span, name string, attrs ...observability.SpanAttribute) {
	if span != nil {
		span.AddEvent(name, attrs...)
	}
}

// TraceContextFromContext extracts trace context from context.
func TraceContextFromContext(ctx context.Context) (traceID, spanID string) {
	return observability.TraceContextFromContext(ctx)
}
