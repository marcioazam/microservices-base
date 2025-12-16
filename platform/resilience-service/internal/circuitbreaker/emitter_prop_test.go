package circuitbreaker

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/testutil"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-microservice, Property 3: Circuit State Change Event Emission**
// **Validates: Requirements 1.5**
func TestProperty_CircuitStateChangeEventEmission(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("state_change_emits_exactly_one_event", prop.ForAll(
		func(serviceNameLen int, correlationIDLen int, threshold int) bool {
			// Generate deterministic strings based on length
			serviceName := testutil.GenerateAlphaString(serviceNameLen)
			correlationID := testutil.GenerateAlphanumericString(correlationIDLen)

			emitter := NewMockEventEmitter()

			cb := New(Config{
				ServiceName:  serviceName,
				EventEmitter: emitter,
				CorrelationFn: func() string {
					return correlationID
				},
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 1,
					Timeout:          time.Second,
				},
			})

			// Clear any initial events
			emitter.Clear()

			// Trigger state change: Closed → Open
			for i := 0; i < threshold; i++ {
				cb.RecordFailure()
			}

			events := emitter.GetStateChangeEvents()

			// Should emit exactly one event
			if len(events) != 1 {
				return false
			}

			event := events[0]

			// Verify event contains required fields
			if event.ServiceName != serviceName {
				return false
			}
			if event.CorrelationID != correlationID {
				return false
			}
			if event.Type != domain.EventCircuitStateChange {
				return false
			}
			if event.Timestamp.IsZero() {
				return false
			}
			if event.ID == "" {
				return false
			}

			// Verify metadata
			metadata := event.Metadata
			if metadata == nil {
				return false
			}
			if metadata["previous_state"] != "CLOSED" {
				return false
			}
			if metadata["new_state"] != "OPEN" {
				return false
			}

			return true
		},
		gen.IntRange(1, 29), // serviceName length
		gen.IntRange(8, 36), // correlationID length
		gen.IntRange(1, 10), // threshold
	))

	props.Property("each_transition_emits_one_event", prop.ForAll(
		func(threshold int) bool {
			emitter := NewMockEventEmitter()

			cb := New(Config{
				ServiceName:  "test-service",
				EventEmitter: emitter,
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 1,
					Timeout:          time.Millisecond,
				},
			})

			emitter.Clear()

			// Transition 1: Closed → Open
			for i := 0; i < threshold; i++ {
				cb.RecordFailure()
			}

			// Wait for timeout
			time.Sleep(2 * time.Millisecond)

			// Transition 2: Open → HalfOpen (triggered by allowRequest)
			cb.allowRequest()

			// Transition 3: HalfOpen → Closed
			cb.RecordSuccess()

			events := emitter.GetStateChangeEvents()

			// Should have exactly 3 events for 3 transitions
			return len(events) == 3
		},
		gen.IntRange(1, 5),
	))

	props.Property("no_event_when_state_unchanged", prop.ForAll(
		func(successCount int) bool {
			emitter := NewMockEventEmitter()

			cb := New(Config{
				ServiceName:  "test-service",
				EventEmitter: emitter,
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: 10, // High threshold
					SuccessThreshold: 1,
					Timeout:          time.Second,
				},
			})

			emitter.Clear()

			// Record successes (should not change state from Closed)
			for i := 0; i < successCount; i++ {
				cb.RecordSuccess()
			}

			events := emitter.GetStateChangeEvents()

			// Should emit no events
			return len(events) == 0
		},
		gen.IntRange(1, 20),
	))

	props.TestingRun(t)
}
