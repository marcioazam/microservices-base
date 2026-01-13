package logging

import (
	"context"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/observability"
	"pgregory.net/rapid"
)

// Property 4: Context Propagation
// For any log entry created with a context containing correlation_id, trace_id,
// or span_id, the shipped log entry should contain those same values.
// Validates: Requirements 2.3, 5.3
func TestProperty_ContextPropagation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random context values
		correlationID := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "correlationID")
		traceID := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "traceID")
		spanID := rapid.StringMatching(`[a-f0-9]{16}`).Draw(t, "spanID")
		message := rapid.StringMatching(`[a-zA-Z0-9 ]{1,100}`).Draw(t, "message")

		// Create context with values
		ctx := context.Background()
		ctx = observability.WithCorrelationID(ctx, correlationID)
		ctx = observability.WithTraceContext(ctx, traceID, spanID)

		// Create a test buffer to capture entries
		var capturedEntries []LogEntry
		testBuffer := newLogBuffer(100, time.Hour, func(entries []LogEntry) error {
			capturedEntries = append(capturedEntries, entries...)
			return nil
		})

		// Create client with test buffer
		client := &Client{
			config: ClientConfig{
				ServiceName: "test-service",
				MinLevel:    LevelDebug,
			},
			buffer:   testBuffer,
			fallback: newLocalLogger("test-service"),
			fields:   make(map[string]any),
		}

		// Log a message
		client.Info(ctx, message)

		// Flush to capture
		testBuffer.Flush()

		// Verify context propagation
		if len(capturedEntries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(capturedEntries))
		}

		entry := capturedEntries[0]
		if entry.CorrelationID != correlationID {
			t.Errorf("CorrelationID mismatch: got %s, want %s", entry.CorrelationID, correlationID)
		}
		if entry.TraceID != traceID {
			t.Errorf("TraceID mismatch: got %s, want %s", entry.TraceID, traceID)
		}
		if entry.SpanID != spanID {
			t.Errorf("SpanID mismatch: got %s, want %s", entry.SpanID, spanID)
		}
	})
}

// Property 5: PII Redaction
// For any log message or field containing PII patterns, the shipped log should
// have those patterns redacted.
// Validates: Requirements 2.5
func TestProperty_PIIRedaction(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"email", "Contact user@example.com for help", "Contact [PII] for help"},
		{"phone", "Call 123-456-7890 now", "Call [PII] now"},
		{"ssn", "SSN is 123-45-6789", "SSN is [PII]"},
		{"credit_card", "Card 1234-5678-9012-3456", "Card [PII]"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := redactPII(tc.input)
			if result != tc.expected {
				t.Errorf("PII not redacted: got %s, want %s", result, tc.expected)
			}
		})
	}
}

// Property test for PII redaction with random inputs
func TestProperty_PIIRedactionRandom(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random email
		email := rapid.StringMatching(`[a-z]{5,10}@[a-z]{5,10}\.[a-z]{2,4}`).Draw(t, "email")
		message := "Contact " + email + " for help"

		result := redactPII(message)

		// Verify email is redacted
		if ContainsPII(result) {
			t.Errorf("PII not fully redacted in: %s", result)
		}
	})
}

// Property test for sensitive key redaction
func TestProperty_SensitiveKeyRedaction(t *testing.T) {
	sensitiveKeys := []string{
		"password", "passwd", "pwd",
		"token", "api_key", "apiKey", "secret",
		"ssn", "social_security",
		"credit_card", "card_number",
	}

	for _, key := range sensitiveKeys {
		t.Run(key, func(t *testing.T) {
			result := RedactSensitive(key, "sensitive_value")
			if result != "[REDACTED]" {
				t.Errorf("Key %s not redacted: got %v", key, result)
			}
		})
	}
}

// Property 9: Log Batching Respects Buffer Size
// For any sequence of log calls, the buffer should flush when reaching the
// configured size limit.
// Validates: Requirements 2.2, 2.6
func TestProperty_BufferSizeRespected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bufferSize := rapid.IntRange(5, 20).Draw(t, "bufferSize")
		logCount := rapid.IntRange(bufferSize, bufferSize*3).Draw(t, "logCount")

		flushCount := 0
		testBuffer := newLogBuffer(bufferSize, time.Hour, func(entries []LogEntry) error {
			flushCount++
			return nil
		})
		defer testBuffer.Close()

		// Add logs
		for i := 0; i < logCount; i++ {
			testBuffer.Add(LogEntry{
				Timestamp: time.Now(),
				Level:     LevelInfo,
				Message:   "test",
			})
		}

		// Final flush
		testBuffer.Flush()

		// Verify flush count
		expectedFlushes := (logCount + bufferSize - 1) / bufferSize
		if flushCount < expectedFlushes-1 || flushCount > expectedFlushes+1 {
			t.Errorf("Unexpected flush count: got %d, expected ~%d for %d logs with buffer %d",
				flushCount, expectedFlushes, logCount, bufferSize)
		}
	})
}

// Unit test for log levels
func TestLogLevels(t *testing.T) {
	logger := LocalOnly("test")
	defer logger.Close()

	ctx := context.Background()

	// Should not panic
	logger.Debug(ctx, "debug message")
	logger.Info(ctx, "info message")
	logger.Warn(ctx, "warn message")
	logger.Error(ctx, "error message")
}

// Unit test for With
func TestWith(t *testing.T) {
	logger := LocalOnly("test")
	defer logger.Close()

	childLogger := logger.With(
		String("key1", "value1"),
		Int("key2", 42),
	)

	if childLogger == logger {
		t.Error("With should return a new logger")
	}
}

// Unit test for FromContext
func TestFromContext(t *testing.T) {
	logger := LocalOnly("test")
	defer logger.Close()

	ctx := observability.WithCorrelationID(context.Background(), "test-correlation")
	childLogger := logger.FromContext(ctx)

	if childLogger == logger {
		t.Error("FromContext should return a new logger")
	}
}
