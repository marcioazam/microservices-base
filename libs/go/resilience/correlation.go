package resilience

import "context"

// CorrelationFunc returns a correlation ID for tracing.
// This type is used to inject correlation ID generation into resilience components.
type CorrelationFunc func() string

// DefaultCorrelationFunc returns an empty correlation ID.
// Use this as a fallback when no correlation function is provided.
func DefaultCorrelationFunc() string {
	return ""
}

// EnsureCorrelationFunc returns the provided function or default if nil.
// This helper ensures that a valid correlation function is always available.
func EnsureCorrelationFunc(fn CorrelationFunc) CorrelationFunc {
	if fn == nil {
		return DefaultCorrelationFunc
	}
	return fn
}

// correlationKey is the context key for correlation IDs.
type correlationKey struct{}

// ContextWithCorrelationID adds a correlation ID to the context.
func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationKey{}, correlationID)
}

// CorrelationIDFromContext extracts the correlation ID from the context.
// Returns an empty string if no correlation ID is present.
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationKey{}).(string); ok {
		return id
	}
	return ""
}

// CorrelationFuncFromContext returns a CorrelationFunc that extracts the ID from context.
func CorrelationFuncFromContext(ctx context.Context) CorrelationFunc {
	return func() string {
		return CorrelationIDFromContext(ctx)
	}
}
