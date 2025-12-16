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

// **Feature: resilience-microservice, Property 2: Circuit Breaker State Serialization Round-Trip**
// **Validates: Requirements 1.6**
func TestProperty_CircuitBreakerStateSerializationRoundTrip(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	// Generator for CircuitBreakerState with valid timestamps
	genState := gopter.CombineGens(
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		gen.IntRange(0, 2),
		gen.IntRange(0, 100),
		gen.IntRange(0, 100),
		gen.Bool(),
		gen.Int64Range(1, 100),
	).Map(func(vals []interface{}) domain.CircuitBreakerState {
		state := domain.CircuitBreakerState{
			ServiceName:     vals[0].(string),
			State:           domain.CircuitState(vals[1].(int)),
			FailureCount:    vals[2].(int),
			SuccessCount:    vals[3].(int),
			LastStateChange: time.Now().Truncate(time.Nanosecond),
			Version:         vals[5].(int64),
		}

		if vals[4].(bool) {
			t := time.Now().Add(-time.Hour).Truncate(time.Nanosecond)
			state.LastFailureTime = &t
		}

		return state
	})

	props.Property("round_trip_preserves_state", prop.ForAll(
		func(original domain.CircuitBreakerState) bool {
			// Serialize
			data, err := MarshalState(original)
			if err != nil {
				return false
			}

			// Deserialize
			restored, err := UnmarshalState(data)
			if err != nil {
				return false
			}

			// Compare fields
			if original.ServiceName != restored.ServiceName {
				return false
			}
			if original.State != restored.State {
				return false
			}
			if original.FailureCount != restored.FailureCount {
				return false
			}
			if original.SuccessCount != restored.SuccessCount {
				return false
			}
			if original.Version != restored.Version {
				return false
			}

			// Compare timestamps with nanosecond precision
			if !original.LastStateChange.Equal(restored.LastStateChange) {
				return false
			}

			// Compare optional LastFailureTime
			if original.LastFailureTime == nil && restored.LastFailureTime != nil {
				return false
			}
			if original.LastFailureTime != nil && restored.LastFailureTime == nil {
				return false
			}
			if original.LastFailureTime != nil && restored.LastFailureTime != nil {
				if !original.LastFailureTime.Equal(*restored.LastFailureTime) {
					return false
				}
			}

			return true
		},
		genState,
	))

	props.TestingRun(t)
}
