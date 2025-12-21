package resilience

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func defaultTestParameters() *gopter.TestParameters {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	return params
}

// **Feature: resilience-lib-extraction, Property 1: Event ID Uniqueness**
// **Validates: Requirements 2.1**
func TestProperty_EventIDUniqueness(t *testing.T) {
	params := defaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("generated_event_ids_are_unique", prop.ForAll(
		func(count int) bool {
			seen := make(map[string]bool)
			for i := 0; i < count; i++ {
				id := GenerateEventID()
				if seen[id] {
					return false
				}
				seen[id] = true
			}
			return true
		},
		gen.IntRange(10, 100),
	))

	props.Property("event_id_has_correct_format", prop.ForAll(
		func(_ int) bool {
			id := GenerateEventID()
			// Format: "20060102150405-a1b2c3d4" (14 chars + 1 dash + 8 hex chars = 23 chars)
			if len(id) != 23 {
				return false
			}
			if id[14] != '-' {
				return false
			}
			return true
		},
		gen.IntRange(1, 50),
	))

	props.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 2: Correlation Function Nil Safety**
// **Validates: Requirements 2.2**
func TestProperty_CorrelationFuncNilSafety(t *testing.T) {
	params := defaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("ensure_correlation_func_returns_non_nil_for_nil_input", prop.ForAll(
		func(_ int) bool {
			fn := EnsureCorrelationFunc(nil)
			return fn != nil && fn() == ""
		},
		gen.IntRange(1, 100),
	))

	props.Property("ensure_correlation_func_preserves_non_nil_input", prop.ForAll(
		func(expected string) bool {
			custom := func() string { return expected }
			fn := EnsureCorrelationFunc(custom)
			return fn() == expected
		},
		gen.AnyString(),
	))

	props.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 3: Time Serialization Round-Trip**
// **Validates: Requirements 2.3**
func TestProperty_TimeSerializationRoundTrip(t *testing.T) {
	params := defaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("marshal_unmarshal_time_round_trip", prop.ForAll(
		func(unixNano int64) bool {
			// Generate a reasonable time (between 2000 and 2100)
			if unixNano < 0 {
				unixNano = -unixNano
			}
			unixNano = unixNano % (100 * 365 * 24 * 60 * 60 * 1e9) // ~100 years in nanoseconds
			unixNano += 946684800 * 1e9                            // Start from year 2000

			original := time.Unix(0, unixNano).UTC()
			marshaled := MarshalTime(original)
			unmarshaled, err := UnmarshalTime(marshaled)
			if err != nil {
				return false
			}
			return original.Equal(unmarshaled)
		},
		gen.Int64(),
	))

	props.Property("marshal_unmarshal_time_ptr_round_trip", prop.ForAll(
		func(unixNano int64) bool {
			if unixNano < 0 {
				unixNano = -unixNano
			}
			unixNano = unixNano % (100 * 365 * 24 * 60 * 60 * 1e9)
			unixNano += 946684800 * 1e9

			original := time.Unix(0, unixNano).UTC()
			marshaled := MarshalTimePtr(&original)
			unmarshaled, err := UnmarshalTimePtr(marshaled)
			if err != nil {
				return false
			}
			return unmarshaled != nil && original.Equal(*unmarshaled)
		},
		gen.Int64(),
	))

	props.Property("nil_time_ptr_marshals_to_empty_string", prop.ForAll(
		func(_ int) bool {
			return MarshalTimePtr(nil) == ""
		},
		gen.IntRange(1, 100),
	))

	props.Property("empty_string_unmarshals_to_nil_time_ptr", prop.ForAll(
		func(_ int) bool {
			ptr, err := UnmarshalTimePtr("")
			return err == nil && ptr == nil
		},
		gen.IntRange(1, 100),
	))

	props.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 7: Nil Emitter Safety**
// **Validates: Requirements 4.2**
func TestProperty_NilEmitterSafety(t *testing.T) {
	params := defaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("emit_event_with_nil_emitter_does_not_panic", prop.ForAll(
		func(eventType string) bool {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("EmitEvent panicked with nil emitter: %v", r)
				}
			}()

			event := Event{
				ID:        GenerateEventID(),
				Type:      EventType(eventType),
				Timestamp: NowUTC(),
			}

			EmitEvent(nil, event)
			return true
		},
		gen.AnyString(),
	))

	props.Property("emit_audit_event_with_nil_emitter_does_not_panic", prop.ForAll(
		func(action string) bool {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("EmitAuditEvent panicked with nil emitter: %v", r)
				}
			}()

			event := AuditEvent{
				ID:        GenerateEventID(),
				Action:    action,
				Timestamp: NowUTC(),
			}

			EmitAuditEvent(nil, event)
			return true
		},
		gen.AnyString(),
	))

	props.TestingRun(t)
}
