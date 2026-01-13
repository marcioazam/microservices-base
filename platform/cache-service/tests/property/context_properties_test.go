package property_test

import (
	"context"
	"testing"

	"github.com/auth-platform/cache-service/internal/observability"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// Property 7: Correlation ID propagation
func TestProperty_CorrelationIDPropagation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		correlationID := rapid.StringMatching(`[a-zA-Z0-9\-]{1,64}`).Draw(t, "correlationID")

		ctx := context.Background()
		ctx = observability.WithCorrelationID(ctx, correlationID)

		// Correlation ID should be retrievable
		retrieved := observability.GetCorrelationID(ctx)
		assert.Equal(t, correlationID, retrieved, "correlation ID should propagate correctly")
	})
}

// Property: Request ID propagation
func TestProperty_RequestIDPropagation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		requestID := rapid.StringMatching(`[a-zA-Z0-9\-]{1,64}`).Draw(t, "requestID")

		ctx := context.Background()
		ctx = observability.WithRequestID(ctx, requestID)

		retrieved := observability.GetRequestID(ctx)
		assert.Equal(t, requestID, retrieved, "request ID should propagate correctly")
	})
}

// Property: Trace ID propagation
func TestProperty_TraceIDPropagation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		traceID := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "traceID")

		ctx := context.Background()
		ctx = observability.WithTraceID(ctx, traceID)

		retrieved := observability.GetTraceID(ctx)
		assert.Equal(t, traceID, retrieved, "trace ID should propagate correctly")
	})
}

// Property: Span ID propagation
func TestProperty_SpanIDPropagation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		spanID := rapid.StringMatching(`[a-f0-9]{16}`).Draw(t, "spanID")

		ctx := context.Background()
		ctx = observability.WithSpanID(ctx, spanID)

		retrieved := observability.GetSpanID(ctx)
		assert.Equal(t, spanID, retrieved, "span ID should propagate correctly")
	})
}

// Property: Context values are independent
func TestProperty_ContextValuesIndependent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		corrID := rapid.StringMatching(`corr-[a-z0-9]{8}`).Draw(t, "corrID")
		reqID := rapid.StringMatching(`req-[a-z0-9]{8}`).Draw(t, "reqID")
		traceID := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "traceID")
		spanID := rapid.StringMatching(`[a-f0-9]{16}`).Draw(t, "spanID")

		ctx := context.Background()
		ctx = observability.WithCorrelationID(ctx, corrID)
		ctx = observability.WithRequestID(ctx, reqID)
		ctx = observability.WithTraceID(ctx, traceID)
		ctx = observability.WithSpanID(ctx, spanID)

		// All values should be independently retrievable
		assert.Equal(t, corrID, observability.GetCorrelationID(ctx))
		assert.Equal(t, reqID, observability.GetRequestID(ctx))
		assert.Equal(t, traceID, observability.GetTraceID(ctx))
		assert.Equal(t, spanID, observability.GetSpanID(ctx))
	})
}

// Property: Child context inherits values
func TestProperty_ChildContextInheritance(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		correlationID := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "correlationID")

		parentCtx := context.Background()
		parentCtx = observability.WithCorrelationID(parentCtx, correlationID)

		childCtx, cancel := context.WithCancel(parentCtx)
		defer cancel()

		// Child should inherit parent's correlation ID
		assert.Equal(t, correlationID, observability.GetCorrelationID(childCtx))
	})
}

// Property: Context value override
func TestProperty_ContextValueOverride(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id1 := rapid.StringMatching(`[a-z]{8}`).Draw(t, "id1")
		id2 := rapid.StringMatching(`[a-z]{8}`).Draw(t, "id2")

		ctx := context.Background()
		ctx = observability.WithCorrelationID(ctx, id1)
		ctx = observability.WithCorrelationID(ctx, id2)

		// Latest value should win
		assert.Equal(t, id2, observability.GetCorrelationID(ctx))
	})
}

// Property: Empty context returns empty string
func TestProperty_EmptyContextReturnsEmpty(t *testing.T) {
	ctx := context.Background()

	assert.Empty(t, observability.GetCorrelationID(ctx))
	assert.Empty(t, observability.GetRequestID(ctx))
	assert.Empty(t, observability.GetTraceID(ctx))
	assert.Empty(t, observability.GetSpanID(ctx))
}
