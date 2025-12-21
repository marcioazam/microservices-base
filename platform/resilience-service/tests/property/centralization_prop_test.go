package property

import (
	"encoding/json"
	"testing"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"pgregory.net/rapid"
)

// **Feature: platform-resilience-modernization, Property 1: Event ID Uniqueness**
// **Validates: Requirements 2.2**
func TestProperty_EventIDUniqueness(t *testing.T) {
	t.Run("generated_event_ids_are_unique", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			count := rapid.IntRange(10, 100).Draw(t, "count")
			seen := make(map[string]bool)
			for i := 0; i < count; i++ {
				id := resilience.GenerateEventID()
				if seen[id] {
					t.Fatalf("duplicate event ID: %s", id)
				}
				seen[id] = true
			}
		})
	})

	t.Run("event_id_has_correct_format", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			_ = rapid.IntRange(1, 50).Draw(t, "iteration")
			id := resilience.GenerateEventID()
			// Format: "20060102150405-a1b2c3d4" (14 chars + 1 dash + 8 hex chars = 23 chars)
			if len(id) != 23 {
				t.Fatalf("event ID length %d != 23", len(id))
			}
			if id[14] != '-' {
				t.Fatalf("event ID missing dash at position 14: %s", id)
			}
		})
	})
}

// **Feature: platform-resilience-modernization, Property 2: Nil Emitter Safety**
// **Validates: Requirements 4.2**
func TestProperty_NilEmitterSafety(t *testing.T) {
	t.Run("emit_event_with_nil_emitter_does_not_panic", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			eventType := rapid.String().Draw(t, "eventType")

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("EmitEvent panicked with nil emitter: %v", r)
				}
			}()

			event := domain.ResilienceEvent{
				ID:        resilience.GenerateEventID(),
				Type:      resilience.EventType(eventType),
				Timestamp: resilience.NowUTC(),
			}

			domain.EmitEvent(nil, event)
		})
	})

	t.Run("emit_audit_event_with_nil_emitter_does_not_panic", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			action := rapid.String().Draw(t, "action")

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("EmitAuditEvent panicked with nil emitter: %v", r)
				}
			}()

			event := domain.AuditEvent{
				ID:        resilience.GenerateEventID(),
				Action:    action,
				Timestamp: resilience.NowUTC(),
			}

			domain.EmitAuditEvent(nil, event)
		})
	})
}

// **Feature: resilience-service-modernization-2025, Property 8: Event JSON Serialization**
// **Validates: Requirements 9.2**
func TestProperty_EventJSONSerialization(t *testing.T) {
	t.Run("resilience_event_json_marshaling_succeeds", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			serviceName := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,49}`).Draw(t, "serviceName")
			eventTypeIdx := rapid.IntRange(0, 5).Draw(t, "eventTypeIdx")

			eventTypes := []resilience.EventType{
				resilience.EventCircuitStateChange,
				resilience.EventRetryAttempt,
				resilience.EventTimeout,
				resilience.EventRateLimitHit,
				resilience.EventBulkheadRejection,
				resilience.EventHealthChange,
			}
			eventType := eventTypes[eventTypeIdx%len(eventTypes)]

			event := domain.ResilienceEvent{
				ID:            resilience.GenerateEventID(),
				Type:          eventType,
				ServiceName:   serviceName,
				Timestamp:     resilience.NowUTC(),
				CorrelationID: "test-correlation",
				Metadata:      map[string]any{"key": "value"},
			}

			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("JSON marshal failed: %v", err)
			}

			var decoded map[string]any
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("JSON unmarshal failed: %v", err)
			}
		})
	})

	t.Run("audit_event_json_marshaling_succeeds", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			action := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_]{0,29}`).Draw(t, "action")
			resource := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_]{0,29}`).Draw(t, "resource")

			event := domain.AuditEvent{
				ID:            resilience.GenerateEventID(),
				Type:          "test-audit",
				Timestamp:     resilience.NowUTC(),
				CorrelationID: "test-correlation",
				Action:        action,
				Resource:      resource,
				Outcome:       "success",
				Metadata:      map[string]any{"key": "value"},
			}

			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("JSON marshal failed: %v", err)
			}

			var decoded map[string]any
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("JSON unmarshal failed: %v", err)
			}
		})
	})
}
