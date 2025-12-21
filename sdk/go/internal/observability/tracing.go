// Package observability provides tracing and logging for the Auth Platform SDK.
package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/auth-platform/sdk-go"

// Tracer wraps OpenTelemetry tracer for SDK operations.
type Tracer struct {
	tracer trace.Tracer
}

// NewTracer creates a new tracer instance.
func NewTracer() *Tracer {
	return &Tracer{
		tracer: otel.Tracer(tracerName),
	}
}

// StartSpan starts a new span for an operation.
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// TokenValidationSpan creates a span for token validation operations.
func (t *Tracer) TokenValidationSpan(ctx context.Context) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "auth.token.validate",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
}

// TokenRefreshSpan creates a span for token refresh operations.
func (t *Tracer) TokenRefreshSpan(ctx context.Context) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "auth.token.refresh",
		trace.WithSpanKind(trace.SpanKindClient),
	)
}

// JWKSFetchSpan creates a span for JWKS fetch operations.
func (t *Tracer) JWKSFetchSpan(ctx context.Context, uri string) (context.Context, trace.Span) {
	ctx, span := t.tracer.Start(ctx, "auth.jwks.fetch",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	// Only add non-sensitive attributes
	span.SetAttributes(attribute.String("jwks.uri_host", SanitizeURI(uri)))
	return ctx, span
}

// ClientCredentialsSpan creates a span for client credentials flow.
func (t *Tracer) ClientCredentialsSpan(ctx context.Context) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "auth.oauth.client_credentials",
		trace.WithSpanKind(trace.SpanKindClient),
	)
}

// DPoPProofSpan creates a span for DPoP proof generation.
func (t *Tracer) DPoPProofSpan(ctx context.Context, method, uri string) (context.Context, trace.Span) {
	ctx, span := t.tracer.Start(ctx, "auth.dpop.generate_proof",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	span.SetAttributes(
		attribute.String("dpop.method", method),
		attribute.String("dpop.uri_host", SanitizeURI(uri)),
	)
	return ctx, span
}

// RecordError records an error on the current span.
func RecordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSuccess marks the span as successful.
func SetSuccess(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}

// AddAttribute adds a non-sensitive attribute to the span.
func AddAttribute(span trace.Span, key string, value string) {
	// Filter out potentially sensitive values
	if IsSensitiveKey(key) {
		return
	}
	span.SetAttributes(attribute.String(key, value))
}

// SanitizeURI extracts only the host from a URI for tracing.
func SanitizeURI(uri string) string {
	// Only return host portion, not full path or query params
	if len(uri) > 100 {
		return uri[:100] + "..."
	}
	return uri
}

// IsSensitiveKey checks if a key might contain sensitive data.
func IsSensitiveKey(key string) bool {
	sensitiveKeys := []string{
		"token", "secret", "password", "key", "credential",
		"authorization", "bearer", "jwt", "access_token",
	}
	for _, s := range sensitiveKeys {
		if containsIgnoreCase(key, s) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && containsLower(s, substr)))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldSlice(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFoldSlice(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if toLower(a[i]) != toLower(b[i]) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
