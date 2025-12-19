package audit

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
)

// **Feature: resilience-service-modernization, Property 6: Structured JSON Logging**
// **Validates: Requirements 5.3**
func TestStructuredJSONLogging(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Log output is valid JSON", prop.ForAll(
		func(eventType string, action string, resource string) bool {
			var buf bytes.Buffer
			logger := NewLogger(Config{
				Output: &buf,
				Level:  slog.LevelInfo,
			})

			event := domain.AuditEvent{
				ID:            domain.GenerateEventID(),
				Type:          eventType,
				Timestamp:     time.Now(),
				CorrelationID: "test-correlation",
				Action:        action,
				Resource:      resource,
				Outcome:       "success",
			}

			logger.Log(event)

			// Verify output is valid JSON
			output := buf.String()
			if output == "" {
				t.Log("Empty output")
				return false
			}

			var parsed map[string]any
			if err := json.Unmarshal([]byte(output), &parsed); err != nil {
				t.Logf("Invalid JSON: %v, output: %s", err, output)
				return false
			}

			return true
		},
		gen.AlphaString(),
		gen.AlphaString(),
		gen.AlphaString(),
	))

	properties.Property("Log contains required fields", prop.ForAll(
		func(eventType string) bool {
			var buf bytes.Buffer
			logger := NewLogger(Config{
				Output: &buf,
				Level:  slog.LevelInfo,
			})

			event := domain.AuditEvent{
				ID:            domain.GenerateEventID(),
				Type:          eventType,
				Timestamp:     time.Now(),
				CorrelationID: "test-correlation",
				Action:        "test-action",
				Resource:      "test-resource",
				Outcome:       "success",
			}

			logger.Log(event)

			output := buf.String()
			var parsed map[string]any
			if err := json.Unmarshal([]byte(output), &parsed); err != nil {
				return false
			}

			// Check required fields are present
			requiredFields := []string{"msg", "level", "time"}
			for _, field := range requiredFields {
				if _, ok := parsed[field]; !ok {
					t.Logf("Missing field: %s", field)
					return false
				}
			}

			return true
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

func TestLogger_LogResilienceEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(Config{
		Output: &buf,
		Level:  slog.LevelInfo,
	})

	event := domain.ResilienceEvent{
		ID:            domain.GenerateEventID(),
		Type:          domain.EventCircuitStateChange,
		ServiceName:   "test-service",
		Timestamp:     time.Now(),
		CorrelationID: "test-correlation",
		TraceID:       "trace-123",
		SpanID:        "span-456",
		Metadata: map[string]any{
			"previous_state": "CLOSED",
			"new_state":      "OPEN",
		},
	}

	logger.LogResilienceEvent(event)

	output := buf.String()
	if output == "" {
		t.Error("Expected log output")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(Config{
		Output: &buf,
		Level:  slog.LevelError,
	})

	logger.Error("test error", domain.NewCircuitOpenError("test-service"))

	output := buf.String()
	if output == "" {
		t.Error("Expected log output")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}

	if parsed["level"] != "ERROR" {
		t.Errorf("Expected ERROR level, got %v", parsed["level"])
	}
}

func TestLogger_GeneratesEventID(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(Config{
		Output: &buf,
		Level:  slog.LevelInfo,
	})

	// Event without ID
	event := domain.AuditEvent{
		Type:          "test",
		Timestamp:     time.Now(),
		CorrelationID: "test-correlation",
	}

	logger.Log(event)

	output := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}

	// Should have generated an event_id
	eventID, ok := parsed["event_id"]
	if !ok {
		t.Error("Expected event_id to be generated")
	}

	// Should be a valid UUID v7
	if eventID != nil && !domain.IsValidUUIDv7(eventID.(string)) {
		t.Errorf("Generated event_id is not valid UUID v7: %v", eventID)
	}
}

func TestValidateEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    domain.AuditEvent
		expected []string
	}{
		{
			name:     "empty event",
			event:    domain.AuditEvent{},
			expected: []string{"id", "type", "timestamp", "correlation_id"},
		},
		{
			name: "complete event",
			event: domain.AuditEvent{
				ID:            "test-id",
				Type:          "test-type",
				Timestamp:     time.Now(),
				CorrelationID: "test-correlation",
			},
			expected: nil,
		},
		{
			name: "missing id",
			event: domain.AuditEvent{
				Type:          "test-type",
				Timestamp:     time.Now(),
				CorrelationID: "test-correlation",
			},
			expected: []string{"id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			missing := ValidateEvent(tt.event)
			if len(missing) != len(tt.expected) {
				t.Errorf("Expected %d missing fields, got %d", len(tt.expected), len(missing))
			}
		})
	}
}

func TestHasRequiredFields(t *testing.T) {
	completeEvent := domain.AuditEvent{
		ID:            "test-id",
		Type:          "test-type",
		Timestamp:     time.Now(),
		CorrelationID: "test-correlation",
	}

	if !HasRequiredFields(completeEvent) {
		t.Error("Complete event should have all required fields")
	}

	incompleteEvent := domain.AuditEvent{
		Type: "test-type",
	}

	if HasRequiredFields(incompleteEvent) {
		t.Error("Incomplete event should not have all required fields")
	}
}
