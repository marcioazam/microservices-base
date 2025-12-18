package property

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/libs/go/resilience/circuitbreaker"
	"github.com/auth-platform/platform/resilience-service/tests/testutil"
	"pgregory.net/rapid"
)

// MockEventEmitter for testing circuit breaker events
type MockEventEmitter struct {
	events []resilience.Event
}

func NewMockEventEmitter() *MockEventEmitter {
	return &MockEventEmitter{events: make([]resilience.Event, 0)}
}

func (m *MockEventEmitter) Emit(event resilience.Event) {
	m.events = append(m.events, event)
}

func (m *MockEventEmitter) EmitAudit(event resilience.AuditEvent) {
	// Not used in these tests
}

func (m *MockEventEmitter) Clear() {
	m.events = make([]resilience.Event, 0)
}

func (m *MockEventEmitter) GetEvents() []resilience.Event {
	return m.events
}

func (m *MockEventEmitter) GetStateChangeEvents() []resilience.Event {
	var result []resilience.Event
	for _, e := range m.events {
		if e.Type == resilience.EventCircuitStateChange {
			result = append(result, e)
		}
	}
	return result
}

// **Feature: resilience-microservice, Property 3: Circuit State Change Event Emission**
// **Validates: Requirements 1.5**
func TestProperty_CircuitStateChangeEventEmission(t *testing.T) {
	t.Run("state_change_emits_exactly_one_event", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			serviceNameLen := rapid.IntRange(1, 29).Draw(t, "serviceNameLen")
			correlationIDLen := rapid.IntRange(8, 36).Draw(t, "correlationIDLen")
			threshold := rapid.IntRange(1, 10).Draw(t, "threshold")

			serviceName := testutil.GenerateAlphaString(serviceNameLen)
			correlationID := testutil.GenerateAlphanumericString(correlationIDLen)

			emitter := NewMockEventEmitter()

			cb := circuitbreaker.New(circuitbreaker.Config{
				ServiceName:  serviceName,
				EventEmitter: emitter,
				CorrelationFn: func() string {
					return correlationID
				},
				Config: resilience.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 1,
					Timeout:          time.Second,
				},
			})

			emitter.Clear()

			for i := 0; i < threshold; i++ {
				cb.RecordFailure()
			}

			events := emitter.GetStateChangeEvents()

			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}

			event := events[0]

			if event.ServiceName != serviceName {
				t.Fatalf("service name mismatch: %s != %s", event.ServiceName, serviceName)
			}
			if event.CorrelationID != correlationID {
				t.Fatalf("correlation ID mismatch: %s != %s", event.CorrelationID, correlationID)
			}
			if event.Type != resilience.EventCircuitStateChange {
				t.Fatalf("event type should be circuit_state_change, got %v", event.Type)
			}
			prevState, ok := event.Metadata["previous_state"].(string)
			if !ok || prevState != "CLOSED" {
				t.Fatalf("previous state should be CLOSED, got %v", prevState)
			}
			newState, ok := event.Metadata["new_state"].(string)
			if !ok || newState != "OPEN" {
				t.Fatalf("new state should be OPEN, got %v", newState)
			}
		})
	})

	t.Run("each_transition_emits_one_event", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			threshold := rapid.IntRange(1, 5).Draw(t, "threshold")

			emitter := NewMockEventEmitter()

			cb := circuitbreaker.New(circuitbreaker.Config{
				ServiceName:  "test-service",
				EventEmitter: emitter,
				Config: resilience.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 1,
					Timeout:          time.Millisecond,
				},
			})

			emitter.Clear()

			for i := 0; i < threshold; i++ {
				cb.RecordFailure()
			}

			time.Sleep(2 * time.Millisecond)

			_ = cb.Execute(context.Background(), func() error { return nil })

			events := emitter.GetStateChangeEvents()

			// CLOSED -> OPEN, OPEN -> HALF_OPEN, HALF_OPEN -> CLOSED = 3 events
			if len(events) != 3 {
				t.Fatalf("expected 3 events, got %d", len(events))
			}
		})
	})

	t.Run("no_event_when_state_unchanged", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			successCount := rapid.IntRange(1, 20).Draw(t, "successCount")

			emitter := NewMockEventEmitter()

			cb := circuitbreaker.New(circuitbreaker.Config{
				ServiceName:  "test-service",
				EventEmitter: emitter,
				Config: resilience.CircuitBreakerConfig{
					FailureThreshold: 10,
					SuccessThreshold: 1,
					Timeout:          time.Second,
				},
			})

			emitter.Clear()

			for i := 0; i < successCount; i++ {
				cb.RecordSuccess()
			}

			events := emitter.GetStateChangeEvents()

			if len(events) != 0 {
				t.Fatalf("expected 0 events, got %d", len(events))
			}
		})
	})
}