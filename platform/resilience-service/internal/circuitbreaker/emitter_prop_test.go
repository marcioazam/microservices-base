package circuitbreaker

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-microservice, Property 3: Circuit State Change Event Emission**
// **Validates: Requirements 1.5**
func TestProperty_CircuitStateChangeEventEmission(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 50
	props := gopter.NewProperties(params)

	props.Property("state_change_emits_exactly_one_event", prop.ForAll(
		func(threshold int) bool {
			emitter := NewMockEventEmitter()
			eventBuilder := domain.NewEventBuilder(emitter, "test-service", nil)

			cb := New(Config{
				ServiceName:  "test-service",
				EventBuilder: eventBuilder,
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 1,
					Timeout:          time.Second,
				},
			})

			emitter.Clear()

			// Trigger state change: Closed → Open
			for i := 0; i < threshold; i++ {
				cb.RecordFailure()
			}

			events := emitter.GetStateChangeEvents()
			return len(events) == 1
		},
		gen.IntRange(1, 10),
	))

	props.TestingRun(t)
}
