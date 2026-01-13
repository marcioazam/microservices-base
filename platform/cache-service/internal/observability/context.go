// Package observability provides unified logging, metrics, and tracing.
// This package wraps libs/go/src/observability for backward compatibility.
package observability

import (
	"context"

	libobs "github.com/authcorp/libs/go/src/observability"
	"go.opentelemetry.io/otel/trace"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey contextKey = "request_id"
)

// WithCorrelationID adds a correlation ID to the context.
// Delegates to lib observability.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return libobs.WithCorrelationID(ctx, id)
}

// GetCorrelationID retrieves the correlation ID from context.
// Delegates to lib observability.
func GetCorrelationID(ctx context.Context) string {
	return libobs.CorrelationIDFromContext(ctx)
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDKey, id)
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithUserID adds a user ID to the context.
// Delegates to lib observability using UserContext.
func WithUserID(ctx context.Context, id string) context.Context {
	return libobs.WithUserContext(ctx, libobs.UserContext{UserID: id})
}

// GetUserID retrieves the user ID from context.
// Delegates to lib observability.
func GetUserID(ctx context.Context) string {
	if user, ok := libobs.UserContextFromContext(ctx); ok {
		return user.UserID
	}
	return ""
}

// GetTraceID extracts the trace ID from context using OpenTelemetry.
func GetTraceID(ctx context.Context) string {
	// First check lib observability
	if traceID, _ := libobs.TraceContextFromContext(ctx); traceID != "" {
		return traceID
	}
	// Fall back to OpenTelemetry
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID extracts the span ID from context using OpenTelemetry.
func GetSpanID(ctx context.Context) string {
	// First check lib observability
	if _, spanID := libobs.TraceContextFromContext(ctx); spanID != "" {
		return spanID
	}
	// Fall back to OpenTelemetry
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// WithTraceID adds a trace ID to the context (for testing/manual override).
// Delegates to lib observability.
func WithTraceID(ctx context.Context, id string) context.Context {
	_, spanID := libobs.TraceContextFromContext(ctx)
	return libobs.WithTraceContext(ctx, id, spanID)
}

// WithSpanID adds a span ID to the context (for testing/manual override).
// Delegates to lib observability.
func WithSpanID(ctx context.Context, id string) context.Context {
	traceID, _ := libobs.TraceContextFromContext(ctx)
	return libobs.WithTraceContext(ctx, traceID, id)
}

// ExtractContext extracts all observability values from context.
func ExtractContext(ctx context.Context) map[string]string {
	m := make(map[string]string)
	if id := GetCorrelationID(ctx); id != "" {
		m["correlation_id"] = id
	}
	if id := GetRequestID(ctx); id != "" {
		m["request_id"] = id
	}
	if id := GetTraceID(ctx); id != "" {
		m["trace_id"] = id
	}
	if id := GetSpanID(ctx); id != "" {
		m["span_id"] = id
	}
	if id := GetUserID(ctx); id != "" {
		m["user_id"] = id
	}
	return m
}

// GenerateCorrelationID generates a new correlation ID.
// Delegates to lib observability.
func GenerateCorrelationID() string {
	return libobs.GenerateCorrelationID()
}

// EnsureCorrelationID ensures context has a correlation ID.
// Delegates to lib observability.
func EnsureCorrelationID(ctx context.Context) context.Context {
	return libobs.EnsureCorrelationID(ctx)
}
