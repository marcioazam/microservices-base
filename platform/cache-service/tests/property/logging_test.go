// Package property contains property-based tests.
package property

import (
	"context"
	"testing"

	"github.com/auth-platform/cache-service/internal/observability"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestCorrelationIDLogging tests Property 14: Correlation ID Logging.
// Validates: Requirements 6.6
func TestCorrelationIDLogging(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Property: Correlation ID set in context is retrievable
	properties.Property("correlation ID round-trip", prop.ForAll(
		func(correlationID string) bool {
			if correlationID == "" {
				return true // Skip empty strings
			}
			ctx := observability.WithCorrelationID(context.Background(), correlationID)
			retrieved := observability.GetCorrelationID(ctx)
			return retrieved == correlationID
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 100 }),
	))

	// Property: Request ID set in context is retrievable
	properties.Property("request ID round-trip", prop.ForAll(
		func(requestID string) bool {
			if requestID == "" {
				return true
			}
			ctx := observability.WithRequestID(context.Background(), requestID)
			retrieved := observability.GetRequestID(ctx)
			return retrieved == requestID
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 100 }),
	))

	// Property: Both IDs can coexist in context
	properties.Property("correlation and request ID coexistence", prop.ForAll(
		func(correlationID, requestID string) bool {
			ctx := context.Background()
			ctx = observability.WithCorrelationID(ctx, correlationID)
			ctx = observability.WithRequestID(ctx, requestID)

			return observability.GetCorrelationID(ctx) == correlationID &&
				observability.GetRequestID(ctx) == requestID
		},
		gen.Identifier(), // Non-empty alphanumeric string
		gen.Identifier(), // Non-empty alphanumeric string
	))

	// Property: Empty context returns empty string
	properties.Property("empty context returns empty correlation ID", prop.ForAll(
		func(_ int) bool {
			ctx := context.Background()
			return observability.GetCorrelationID(ctx) == ""
		},
		gen.Int(),
	))

	properties.TestingRun(t)
}
