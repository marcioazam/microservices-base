package observability_test

import (
	"context"
	"testing"

	"github.com/auth-platform/cache-service/internal/observability"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestCorrelationID(t *testing.T) {
	ctx := context.Background()

	assert.Empty(t, observability.GetCorrelationID(ctx))

	ctx = observability.WithCorrelationID(ctx, "corr-123")
	assert.Equal(t, "corr-123", observability.GetCorrelationID(ctx))

	ctx = observability.WithCorrelationID(ctx, "corr-456")
	assert.Equal(t, "corr-456", observability.GetCorrelationID(ctx))
}

func TestRequestID(t *testing.T) {
	ctx := context.Background()

	assert.Empty(t, observability.GetRequestID(ctx))

	ctx = observability.WithRequestID(ctx, "req-123")
	assert.Equal(t, "req-123", observability.GetRequestID(ctx))
}

func TestUserID(t *testing.T) {
	ctx := context.Background()

	assert.Empty(t, observability.GetUserID(ctx))

	ctx = observability.WithUserID(ctx, "user-123")
	assert.Equal(t, "user-123", observability.GetUserID(ctx))
}

func TestTraceIDFromOpenTelemetry(t *testing.T) {
	// Without a span, trace ID should be empty
	ctx := context.Background()
	assert.Empty(t, observability.GetTraceID(ctx))

	// With a real span, trace ID should be extracted
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(ctx)
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	traceID := observability.GetTraceID(ctx)
	assert.NotEmpty(t, traceID)
	assert.Len(t, traceID, 32) // Trace IDs are 32 hex characters
}

func TestSpanIDFromOpenTelemetry(t *testing.T) {
	// Without a span, span ID should be empty
	ctx := context.Background()
	assert.Empty(t, observability.GetSpanID(ctx))

	// With a real span, span ID should be extracted
	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(ctx)
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	spanID := observability.GetSpanID(ctx)
	assert.NotEmpty(t, spanID)
	assert.Len(t, spanID, 16) // Span IDs are 16 hex characters
}

func TestMultipleContextValues(t *testing.T) {
	ctx := context.Background()

	ctx = observability.WithCorrelationID(ctx, "corr-1")
	ctx = observability.WithRequestID(ctx, "req-1")
	ctx = observability.WithUserID(ctx, "user-1")

	assert.Equal(t, "corr-1", observability.GetCorrelationID(ctx))
	assert.Equal(t, "req-1", observability.GetRequestID(ctx))
	assert.Equal(t, "user-1", observability.GetUserID(ctx))
}

func TestContextValueIsolation(t *testing.T) {
	ctx1 := context.Background()
	ctx1 = observability.WithCorrelationID(ctx1, "corr-1")

	ctx2 := context.Background()
	ctx2 = observability.WithCorrelationID(ctx2, "corr-2")

	assert.Equal(t, "corr-1", observability.GetCorrelationID(ctx1))
	assert.Equal(t, "corr-2", observability.GetCorrelationID(ctx2))
}

func TestEmptyValues(t *testing.T) {
	ctx := context.Background()

	ctx = observability.WithCorrelationID(ctx, "")
	assert.Empty(t, observability.GetCorrelationID(ctx))
}

func TestContextChaining(t *testing.T) {
	ctx := context.Background()

	ctx = observability.WithCorrelationID(ctx, "corr")
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	assert.Equal(t, "corr", observability.GetCorrelationID(childCtx))
}

func TestExtractContext(t *testing.T) {
	ctx := context.Background()
	ctx = observability.WithCorrelationID(ctx, "corr-123")
	ctx = observability.WithRequestID(ctx, "req-456")
	ctx = observability.WithUserID(ctx, "user-789")

	extracted := observability.ExtractContext(ctx)

	assert.Equal(t, "corr-123", extracted["correlation_id"])
	assert.Equal(t, "req-456", extracted["request_id"])
	assert.Equal(t, "user-789", extracted["user_id"])
}

func TestExtractContextEmpty(t *testing.T) {
	ctx := context.Background()
	extracted := observability.ExtractContext(ctx)

	assert.Empty(t, extracted)
}

func TestExtractContextWithOpenTelemetry(t *testing.T) {
	ctx := context.Background()
	ctx = observability.WithCorrelationID(ctx, "corr-123")

	tp := sdktrace.NewTracerProvider()
	defer tp.Shutdown(ctx)
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	extracted := observability.ExtractContext(ctx)

	assert.Equal(t, "corr-123", extracted["correlation_id"])
	assert.NotEmpty(t, extracted["trace_id"])
	assert.NotEmpty(t, extracted["span_id"])
}

func TestNoopSpanReturnsEmptyIDs(t *testing.T) {
	ctx := context.Background()

	// Create a noop span context
	ctx = trace.ContextWithSpanContext(ctx, trace.SpanContext{})

	assert.Empty(t, observability.GetTraceID(ctx))
	assert.Empty(t, observability.GetSpanID(ctx))
}
