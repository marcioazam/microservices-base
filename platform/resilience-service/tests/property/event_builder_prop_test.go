// Package property contains property-based tests for the resilience service.
package property

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"go.opentelemetry.io/otel/trace"
)

// mockEventEmitter is a test implementation of EventEmitter.
type mockEventEmitter struct {
	mu     sync.Mutex
	events []domain.ResilienceEvent
}

func newMockEventEmitter() *mockEventEmitter {
	return &mockEventEmitter{events: make([]domain.ResilienceEvent, 0)}
}

func (m *mockEventEmitter) Emit(event domain.ResilienceEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockEventEmitter) EmitAudit(event domain.AuditEvent) {}

func (m *mockEventEmitter) GetEvents() []domain.ResilienceEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]domain.ResilienceEvent, len(m.events))
	copy(result, m.events)
	return result
}

// TestEventBuilderAutomaticFieldPopulation validates EventBuilder field population.
func TestEventBuilderAutomaticFieldPopulation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	eventTypes := []domain.EventType{
		domain.EventCircuitOpen,
		domain.EventCircuitClosed,
		domain.EventCircuitHalfOpen,
		domain.EventRateLimited,
		domain.EventTimeout,
		domain.EventBulkheadFull,
		domain.EventRetryAttempt,
		domain.EventRetryExhausted,
	}

	properties.Property("EventBuilder populates all required fields", prop.ForAll(
		func(serviceName string, correlationID string, eventTypeIdx int) bool {
			if serviceName == "" {
				serviceName = "test-service"
			}
			eventType := eventTypes[eventTypeIdx%len(eventTypes)]
			correlationFn := func() string { return correlationID }

			emitter := newMockEventEmitter()
			builder := domain.NewEventBuilder(emitter, serviceName, correlationFn)

			before := time.Now()
			event := builder.Build(eventType, map[string]any{"test": "value"})
			after := time.Now()

			// Test ID is valid UUID v7
			if !domain.IsValidUUIDv7(event.ID) {
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

// TestNilEmitterSafety validates nil emitter handling.
func TestNilEmitterSafety(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Emit with nil emitter does not panic", prop.ForAll(
		func(serviceName string) bool {
			if serviceName == "" {
				serviceName = "test-service"
			}
			builder := domain.NewEventBuilder(nil, serviceName, nil)

			defer func() {
				if r := recover(); r != nil {
					t.Logf("Panic occurred: %v", r)
				}
			}()

			builder.Emit(domain.EventCircuitOpen, map[string]any{"test": "value"})
			return true
		},
		gen.AlphaString(),
	))

	properties.Property("Emit on nil builder does not panic", prop.ForAll(
		func(_ int) bool {
			var builder *domain.EventBuilder

			defer func() {
				if r := recover(); r != nil {
					t.Logf("Panic occurred: %v", r)
				}
			}()

			builder.Emit(domain.EventCircuitOpen, map[string]any{"test": "value"})
			return true
		},
		gen.Int(),
	))

	properties.TestingRun(t)
}

// TestEventSerializationRoundTrip validates JSON serialization.
func TestEventSerializationRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("JSON serialization produces expected fields", prop.ForAll(
		func(serviceName string, correlationID string) bool {
			if serviceName == "" {
				serviceName = "test-service"
			}
			correlationFn := func() string { return correlationID }
			builder := domain.NewEventBuilder(newMockEventEmitter(), serviceName, correlationFn)

			event := builder.Build(domain.EventCircuitOpen, map[string]any{
				"previous_state": "CLOSED",
				"new_state":      "OPEN",
			})

			data, err := json.Marshal(event)
			if err != nil {
				t.Logf("Failed to marshal: %v", err)
				return false
			}

			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Logf("Failed to unmarshal: %v", err)
				return false
			}

			requiredFields := []string{"id", "type", "service_name", "timestamp", "correlation_id"}
			for _, field := range requiredFields {
				if _, ok := parsed[field]; !ok {
					t.Logf("Missing required field: %s", field)
					return false
				}
			}

			return true
		},
		gen.AlphaString(),
		gen.AlphaString(),
	))

	properties.Property("Deserialization produces equivalent event", prop.ForAll(
		func(serviceName string) bool {
			if serviceName == "" {
				serviceName = "test-service"
			}
			builder := domain.NewEventBuilder(newMockEventEmitter(), serviceName, nil)
			original := builder.Build(domain.EventRetryAttempt, map[string]any{"attempt": 1})

			data, err := json.Marshal(original)
			if err != nil {
				return false
			}

			var restored domain.ResilienceEvent
			if err := json.Unmarshal(data, &restored); err != nil {
				return false
			}

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

// TestTraceContextPropagation validates trace context in events.
func TestTraceContextPropagation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Events include TraceID and SpanID from context", prop.ForAll(
		func(serviceName string) bool {
			if serviceName == "" {
				serviceName = "test-service"
			}
			builder := domain.NewEventBuilder(newMockEventEmitter(), serviceName, nil)

			traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
			spanID, _ := trace.SpanIDFromHex("0102030405060708")
			spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
			})
			ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

			event := builder.BuildWithContext(ctx, domain.EventCircuitOpen, nil)

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
	builder := domain.NewEventBuilder(newMockEventEmitter(), "test-service", nil)
	event := builder.Build(domain.EventCircuitOpen, nil)

	if event.CorrelationID != "" {
		t.Errorf("Expected empty correlation ID with default function, got %s", event.CorrelationID)
	}
}

func TestEventBuilder_EmitActuallyEmits(t *testing.T) {
	emitter := newMockEventEmitter()
	builder := domain.NewEventBuilder(emitter, "test-service", nil)

	builder.Emit(domain.EventCircuitOpen, map[string]any{"test": true})

	events := emitter.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}
