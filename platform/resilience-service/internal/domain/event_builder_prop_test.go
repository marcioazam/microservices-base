package domain

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"go.opentelemetry.io/otel/trace"
)

// mockEmitter is a test implementation of EventEmitter.
type mockEmitter struct {
	mu     sync.Mutex
	events []ResilienceEvent
}

func newMockEmitter() *mockEmitter {
	return &mockEmitter{events: make([]ResilienceEvent, 0)}
}

func (m *mockEmitter) Emit(event ResilienceEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockEmitter) EmitAudit(event AuditEvent) {}

func (m *mockEmitter) GetEvents() []ResilienceEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]ResilienceEvent, len(m.events))
	copy(result, m.events)
	return result
}

// **Feature: resilience-service-modernization, Property 4: EventBuilder Automatic Field Population**
// **Validates: Requirements 4.2, 4.4**
func TestEventBuilderAutomaticFieldPopulation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	eventTypes := []ResilienceEventType{
		EventCircuitStateChange,
		EventRetryAttempt,
		EventTimeout,
		EventRateLimitHit,
		EventBulkheadRejection,
		EventHealthChange,
	}

	properties.Property("EventBuilder populates all required fields", prop.ForAll(
		func(serviceName string, correlationID string, eventTypeIdx int) bool {
			eventType := eventTypes[eventTypeIdx%len(eventTypes)]
			correlationFn := func() string { return correlationID }

			emitter := newMockEmitter()
			builder := NewEventBuilder(emitter, serviceName, correlationFn)

			before := time.Now()
			event := builder.Build(eventType, map[string]any{"test": "value"})
			after := time.Now()

			// Test ID is valid UUID v7
			if !IsValidUUIDv7(event.ID) {
				t.Logf("ID is not valid UUID v7: %s", event.ID)
				return false
			}

			// Test Timestamp is within 1 second of build time
			if event.Timestamp.Before(before.Add(-time.Second)) || event.Timestamp.After(after.Add(time.Second)) {
				t.Logf("Timestamp out of range: %v", event.Timestamp)
				return false
			}

			// Test Type matches requested type
			if event.Type != eventType {
				t.Logf("Type mismatch: expected %s, got %s", eventType, event.Type)
				return false
			}

			// Test ServiceName is populated
			if event.ServiceName != serviceName {
				t.Logf("ServiceName mismatch: expected %s, got %s", serviceName, event.ServiceName)
				return false
			}

			// Test CorrelationID is populated
			if event.CorrelationID != correlationID {
				t.Logf("CorrelationID mismatch: expected %s, got %s", correlationID, event.CorrelationID)
				return false
			}

			return true
		},
		gen.AlphaString(),
		gen.AlphaString(),
		gen.IntRange(0, 100),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-service-modernization, Property 5: Nil Emitter Safety**
// **Validates: Requirements 4.3**
func TestNilEmitterSafety(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Emit with nil emitter does not panic", prop.ForAll(
		func(serviceName string) bool {
			// Test with nil emitter in constructor
			builder := NewEventBuilder(nil, serviceName, nil)

			// This should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Panic occurred: %v", r)
				}
			}()

			builder.Emit(EventCircuitStateChange, map[string]any{"test": "value"})
			return true
		},
		gen.AlphaString(),
	))

	properties.Property("Emit on nil builder does not panic", prop.ForAll(
		func(_ int) bool {
			var builder *EventBuilder

			// This should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Panic occurred: %v", r)
				}
			}()

			builder.Emit(EventCircuitStateChange, map[string]any{"test": "value"})
			return true
		},
		gen.Int(),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-service-modernization, Property 8: Event Serialization Backward Compatibility**
// **Validates: Requirements 10.2**
func TestEventSerializationBackwardCompatibility(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("JSON serialization produces expected fields", prop.ForAll(
		func(serviceName string, correlationID string) bool {
			correlationFn := func() string { return correlationID }
			builder := NewEventBuilder(newMockEmitter(), serviceName, correlationFn)

			event := builder.Build(EventCircuitStateChange, map[string]any{
				"previous_state": "CLOSED",
				"new_state":      "OPEN",
			})

			// Serialize to JSON
			data, err := json.Marshal(event)
			if err != nil {
				t.Logf("Failed to marshal: %v", err)
				return false
			}

			// Deserialize back
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Logf("Failed to unmarshal: %v", err)
				return false
			}

			// Verify all required fields are present
			requiredFields := []string{"id", "type", "service_name", "timestamp", "correlation_id"}
			for _, field := range requiredFields {
				if _, ok := parsed[field]; !ok {
					t.Logf("Missing required field: %s", field)
					return false
				}
			}

			// Verify metadata is present
			if _, ok := parsed["metadata"]; !ok {
				t.Logf("Missing metadata field")
				return false
			}

			return true
		},
		gen.AlphaString(),
		gen.AlphaString(),
	))

	properties.Property("Deserialization produces equivalent event", prop.ForAll(
		func(serviceName string) bool {
			builder := NewEventBuilder(newMockEmitter(), serviceName, nil)
			original := builder.Build(EventRetryAttempt, map[string]any{"attempt": 1})

			// Round-trip serialization
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}

			var restored ResilienceEvent
			if err := json.Unmarshal(data, &restored); err != nil {
				return false
			}

			// Verify key fields match
			if restored.ID != original.ID {
				return false
			}
			if restored.Type != original.Type {
				return false
			}
			if restored.ServiceName != original.ServiceName {
				return false
			}

			return true
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-service-modernization, Property 7: Trace Context Propagation**
// **Validates: Requirements 7.4**
func TestTraceContextPropagation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Events include TraceID and SpanID from context", prop.ForAll(
		func(serviceName string) bool {
			builder := NewEventBuilder(newMockEmitter(), serviceName, nil)

			// Create a context with valid trace info
			traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
			spanID, _ := trace.SpanIDFromHex("0102030405060708")
			spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			})
			ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

			event := builder.BuildWithContext(ctx, EventCircuitStateChange, nil)

			// Verify trace context is propagated
			if event.TraceID != traceID.String() {
				t.Logf("TraceID mismatch: expected %s, got %s", traceID.String(), event.TraceID)
				return false
			}
			if event.SpanID != spanID.String() {
				t.Logf("SpanID mismatch: expected %s, got %s", spanID.String(), event.SpanID)
				return false
			}

			return true
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

func TestEventBuilder_DefaultCorrelation(t *testing.T) {
	builder := NewEventBuilder(newMockEmitter(), "test-service", nil)
	event := builder.Build(EventCircuitStateChange, nil)

	if event.CorrelationID != "" {
		t.Errorf("Expected empty correlation ID with default function, got %s", event.CorrelationID)
	}
}

func TestEventBuilder_EmitActuallyEmits(t *testing.T) {
	emitter := newMockEmitter()
	builder := NewEventBuilder(emitter, "test-service", nil)

	builder.Emit(EventCircuitStateChange, map[string]any{"test": true})

	events := emitter.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}
