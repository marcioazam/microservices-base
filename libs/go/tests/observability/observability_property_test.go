// Feature: go-libs-state-of-art-2025, Property 8: Observability Context Propagation
// Validates: Requirements 10.3, 10.4, 10.5
package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/observability"
	"pgregory.net/rapid"
)

// Property 9: Log Entry Timestamp Format (ISO 8601)
func TestLogEntryTimestampFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		message := rapid.StringMatching(`[a-zA-Z ]{5,50}`).Draw(t, "message")

		var buf bytes.Buffer
		logger := observability.NewLogger("test-service").WithOutput(&buf)

		logger.Info(message)

		var entry observability.LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("failed to parse log entry: %v", err)
		}

		// Verify timestamp is valid RFC3339
		_, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if err != nil {
			t.Fatalf("timestamp not RFC3339: %s, error: %v", entry.Timestamp, err)
		}

		// Verify timestamp ends with Z (UTC)
		if !strings.HasSuffix(entry.Timestamp, "Z") {
			t.Fatalf("timestamp should be UTC: %s", entry.Timestamp)
		}
	})
}

// Property 10: Trace Context Propagation
func TestTraceContextPropagation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		traceID := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "traceID")
		spanID := rapid.StringMatching(`[a-f0-9]{16}`).Draw(t, "spanID")
		correlationID := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "correlationID")

		ctx := context.Background()
		ctx = observability.WithTraceContext(ctx, traceID, spanID)
		ctx = observability.WithCorrelationID(ctx, correlationID)

		// Extract and verify
		extractedTrace, extractedSpan := observability.TraceContextFromContext(ctx)
		extractedCorr := observability.CorrelationIDFromContext(ctx)

		if extractedTrace != traceID {
			t.Fatalf("trace ID not preserved: got %s, want %s", extractedTrace, traceID)
		}
		if extractedSpan != spanID {
			t.Fatalf("span ID not preserved: got %s, want %s", extractedSpan, spanID)
		}
		if extractedCorr != correlationID {
			t.Fatalf("correlation ID not preserved: got %s, want %s", extractedCorr, correlationID)
		}

		// Verify propagation
		propagated := observability.PropagateContext(ctx)
		propTrace, propSpan := observability.TraceContextFromContext(propagated)
		propCorr := observability.CorrelationIDFromContext(propagated)

		if propTrace != traceID || propSpan != spanID || propCorr != correlationID {
			t.Fatal("context values not propagated correctly")
		}
	})
}

// Property 11: PII Redaction in Logs
func TestPIIRedactionInLogs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		sensitiveKey := rapid.SampledFrom([]string{
			"password", "token", "api_key", "secret", "credential",
		}).Draw(t, "key")
		sensitiveValue := rapid.StringMatching(`[a-zA-Z0-9]{16,32}`).Draw(t, "value")

		var buf bytes.Buffer
		logger := observability.NewLogger("test-service").WithOutput(&buf)

		logger.Info("test message", map[string]any{
			sensitiveKey: sensitiveValue,
		})

		output := buf.String()

		// Sensitive value should be redacted
		if strings.Contains(output, sensitiveValue) {
			t.Fatalf("sensitive value should be redacted: %s", output)
		}

		// Should contain [REDACTED]
		if !strings.Contains(output, "[REDACTED]") {
			t.Fatalf("output should contain [REDACTED]: %s", output)
		}
	})
}

// Property 12: Log Level Filtering
func TestLogLevelFiltering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		minLevel := rapid.SampledFrom([]observability.Level{
			observability.LevelDebug,
			observability.LevelInfo,
			observability.LevelWarn,
			observability.LevelError,
		}).Draw(t, "minLevel")

		var buf bytes.Buffer
		logger := observability.NewLogger("test").WithOutput(&buf).WithLevel(minLevel)

		// Log at all levels
		logger.Debug("debug")
		logger.Info("info")
		logger.Warn("warn")
		logger.Error("error")

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")

		// Count expected lines based on level
		expectedCount := 0
		if minLevel <= observability.LevelDebug {
			expectedCount = 4
		} else if minLevel <= observability.LevelInfo {
			expectedCount = 3
		} else if minLevel <= observability.LevelWarn {
			expectedCount = 2
		} else {
			expectedCount = 1
		}

		if len(lines) != expectedCount {
			t.Fatalf("expected %d log lines for level %s, got %d", expectedCount, minLevel, len(lines))
		}
	})
}

// Property 13: Correlation ID Generation Uniqueness
func TestCorrelationIDUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(10, 100).Draw(t, "count")
		seen := make(map[string]bool)

		for i := 0; i < count; i++ {
			id := observability.GenerateCorrelationID()
			if seen[id] {
				t.Fatalf("duplicate correlation ID: %s", id)
			}
			seen[id] = true

			// Verify format (32 hex chars)
			if len(id) != 32 {
				t.Fatalf("correlation ID should be 32 chars: %s", id)
			}
		}
	})
}

// Property 14: User Context Preservation
func TestUserContextPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := rapid.StringMatching(`[a-f0-9]{24}`).Draw(t, "userID")
		tenantID := rapid.StringMatching(`[a-f0-9]{24}`).Draw(t, "tenantID")
		roleCount := rapid.IntRange(1, 5).Draw(t, "roleCount")
		roles := make([]string, roleCount)
		for i := 0; i < roleCount; i++ {
			roles[i] = rapid.SampledFrom([]string{"admin", "user", "viewer", "editor"}).Draw(t, "role")
		}

		user := observability.UserContext{
			UserID:   userID,
			TenantID: tenantID,
			Roles:    roles,
		}

		ctx := observability.WithUserContext(context.Background(), user)
		extracted, ok := observability.UserContextFromContext(ctx)

		if !ok {
			t.Fatal("user context not found")
		}
		if extracted.UserID != userID {
			t.Fatalf("user ID mismatch: got %s, want %s", extracted.UserID, userID)
		}
		if extracted.TenantID != tenantID {
			t.Fatalf("tenant ID mismatch: got %s, want %s", extracted.TenantID, tenantID)
		}
		if len(extracted.Roles) != len(roles) {
			t.Fatalf("roles count mismatch: got %d, want %d", len(extracted.Roles), len(roles))
		}
	})
}
