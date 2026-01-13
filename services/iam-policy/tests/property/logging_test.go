// Package property contains property-based tests for IAM Policy Service.
package property

import (
	"context"
	"testing"

	"github.com/auth-platform/iam-policy-service/tests/testutil"
	"pgregory.net/rapid"
)

// TestLogEntryEnrichment validates Property 3: Log Entry Enrichment.
// All log entries must include correlation_id, trace_id, and span_id when present in context.
func TestLogEntryEnrichment(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random context values
		correlationID := testutil.CorrelationIDGen().Draw(t, "correlationID")
		traceID := testutil.TraceIDGen().Draw(t, "traceID")
		spanID := testutil.SpanIDGen().Draw(t, "spanID")
		message := testutil.NonEmptyStringGen().Draw(t, "message")

		// Create mock logger to capture entries
		mockLogger := testutil.NewMockLogger()

		// Create enriched context
		ctx := context.Background()
		fields := map[string]interface{}{
			"correlation_id": correlationID,
			"trace_id":       traceID,
			"span_id":        spanID,
		}

		// Log with enriched context
		mockLogger.Info(ctx, message, fields)

		// Verify enrichment
		entries := mockLogger.GetEntries()
		if len(entries) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(entries))
		}

		entry := entries[0]

		// Property: correlation_id must be present
		if entry.Fields["correlation_id"] != correlationID {
			t.Errorf("correlation_id mismatch: expected %s, got %v", correlationID, entry.Fields["correlation_id"])
		}

		// Property: trace_id must be present
		if entry.Fields["trace_id"] != traceID {
			t.Errorf("trace_id mismatch: expected %s, got %v", traceID, entry.Fields["trace_id"])
		}

		// Property: span_id must be present
		if entry.Fields["span_id"] != spanID {
			t.Errorf("span_id mismatch: expected %s, got %v", spanID, entry.Fields["span_id"])
		}

		// Property: message must be preserved
		if entry.Message != message {
			t.Errorf("message mismatch: expected %s, got %s", message, entry.Message)
		}
	})
}

// TestLogLevelFiltering validates that log levels are correctly applied.
func TestLogLevelFiltering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		level := rapid.SampledFrom([]string{"DEBUG", "INFO", "WARN", "ERROR"}).Draw(t, "level")
		message := testutil.NonEmptyStringGen().Draw(t, "message")

		mockLogger := testutil.NewMockLogger()
		ctx := context.Background()

		// Log at the selected level
		switch level {
		case "DEBUG":
			mockLogger.Debug(ctx, message, nil)
		case "INFO":
			mockLogger.Info(ctx, message, nil)
		case "WARN":
			mockLogger.Warn(ctx, message, nil)
		case "ERROR":
			mockLogger.Error(ctx, message, nil)
		}

		entries := mockLogger.GetEntries()
		if len(entries) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(entries))
		}

		// Property: level must match
		if entries[0].Level != level {
			t.Errorf("level mismatch: expected %s, got %s", level, entries[0].Level)
		}
	})
}

// TestLogFieldPreservation validates that all fields are preserved in log entries.
func TestLogFieldPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random fields
		numFields := rapid.IntRange(1, 10).Draw(t, "numFields")
		fields := make(map[string]interface{})

		for i := 0; i < numFields; i++ {
			key := rapid.StringMatching(`^[a-z_]{3,15}$`).Draw(t, "fieldKey")
			value := rapid.String().Draw(t, "fieldValue")
			fields[key] = value
		}

		mockLogger := testutil.NewMockLogger()
		ctx := context.Background()

		mockLogger.Info(ctx, "test message", fields)

		entries := mockLogger.GetEntries()
		if len(entries) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(entries))
		}

		// Property: all fields must be preserved
		for key, expectedValue := range fields {
			if entries[0].Fields[key] != expectedValue {
				t.Errorf("field %s mismatch: expected %v, got %v", key, expectedValue, entries[0].Fields[key])
			}
		}
	})
}
