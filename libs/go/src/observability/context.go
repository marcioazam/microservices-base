package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type contextKey string

const (
	correlationIDKey contextKey = "correlation_id"
	traceIDKey       contextKey = "trace_id"
	spanIDKey        contextKey = "span_id"
	userContextKey   contextKey = "user_context"
)

// UserContext holds user information for logging.
type UserContext struct {
	UserID   string
	TenantID string
	Roles    []string
}

// WithCorrelationID adds correlation ID to context.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// CorrelationIDFromContext extracts correlation ID from context.
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

// WithTraceContext adds trace context to context.
func WithTraceContext(ctx context.Context, traceID, spanID string) context.Context {
	ctx = context.WithValue(ctx, traceIDKey, traceID)
	ctx = context.WithValue(ctx, spanIDKey, spanID)
	return ctx
}

// TraceContextFromContext extracts trace context from context.
func TraceContextFromContext(ctx context.Context) (traceID, spanID string) {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		traceID = id
	}
	if id, ok := ctx.Value(spanIDKey).(string); ok {
		spanID = id
	}
	return
}

// WithUserContext adds user context to context.
func WithUserContext(ctx context.Context, user UserContext) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserContextFromContext extracts user context from context.
func UserContextFromContext(ctx context.Context) (UserContext, bool) {
	user, ok := ctx.Value(userContextKey).(UserContext)
	return user, ok
}

// GenerateCorrelationID generates a new correlation ID.
func GenerateCorrelationID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateTraceID generates a new trace ID (W3C format).
func GenerateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateSpanID generates a new span ID (W3C format).
func GenerateSpanID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// EnsureCorrelationID ensures context has a correlation ID.
func EnsureCorrelationID(ctx context.Context) context.Context {
	if CorrelationIDFromContext(ctx) == "" {
		return WithCorrelationID(ctx, GenerateCorrelationID())
	}
	return ctx
}

// PropagateContext creates a child context with propagated values.
func PropagateContext(parent context.Context) context.Context {
	ctx := context.Background()
	if id := CorrelationIDFromContext(parent); id != "" {
		ctx = WithCorrelationID(ctx, id)
	}
	if traceID, spanID := TraceContextFromContext(parent); traceID != "" {
		ctx = WithTraceContext(ctx, traceID, spanID)
	}
	if user, ok := UserContextFromContext(parent); ok {
		ctx = WithUserContext(ctx, user)
	}
	return ctx
}
