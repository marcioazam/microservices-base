package retry

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/testutil"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-microservice, Property 7: Retry Policy Configuration Round-Trip**
// **Validates: Requirements 2.6**
func TestProperty_RetryPolicyConfigurationRoundTrip(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	// Generator for valid PolicyDefinition
	genPolicy := gopter.CombineGens(
		gen.IntRange(1, 10),                // MaxAttempts
		gen.IntRange(10, 500),              // BaseDelayMs
		gen.IntRange(1000, 10000),          // MaxDelayMs
		gen.Float64Range(1.0, 5.0),         // Multiplier
		gen.Float64Range(0.0, 0.5),         // JitterPercent
		gen.SliceOfN(3, gen.AlphaString()), // RetryOn
	).Map(func(vals []interface{}) PolicyDefinition {
		return PolicyDefinition{
			MaxAttempts:   vals[0].(int),
			BaseDelayMs:   vals[1].(int),
			MaxDelayMs:    vals[2].(int),
			Multiplier:    vals[3].(float64),
			JitterPercent: vals[4].(float64),
			RetryOn:       vals[5].([]string),
		}
	}).SuchThat(func(def PolicyDefinition) bool {
		// Ensure base < max
		return def.BaseDelayMs < def.MaxDelayMs
	})

	props.Property("round_trip_preserves_policy", prop.ForAll(
		func(original PolicyDefinition) bool {
			// Convert to RetryConfig
			cfg := FromDefinition(original)

			// Serialize to JSON
			data, err := MarshalPolicy(cfg)
			if err != nil {
				return false
			}

			// Parse back
			restored, err := ParsePolicy(data)
			if err != nil {
				return false
			}

			// Compare
			if cfg.MaxAttempts != restored.MaxAttempts {
				return false
			}
			if cfg.BaseDelay != restored.BaseDelay {
				return false
			}
			if cfg.MaxDelay != restored.MaxDelay {
				return false
			}
			if cfg.Multiplier != restored.Multiplier {
				return false
			}
			if cfg.JitterPercent != restored.JitterPercent {
				return false
			}

			// Compare RetryableErrors
			if len(cfg.RetryableErrors) != len(restored.RetryableErrors) {
				return false
			}
			for i, v := range cfg.RetryableErrors {
				if v != restored.RetryableErrors[i] {
					return false
				}
			}

			return true
		},
		genPolicy,
	))

	props.Property("parse_then_marshal_then_parse_equals_original", prop.ForAll(
		func(original PolicyDefinition) bool {
			// First parse
			cfg1, err := ParsePolicy(mustMarshal(original))
			if err != nil {
				return false
			}

			// Marshal
			data, err := MarshalPolicy(cfg1)
			if err != nil {
				return false
			}

			// Parse again
			cfg2, err := ParsePolicy(data)
			if err != nil {
				return false
			}

			// Should be equal
			return configsEqual(cfg1, cfg2)
		},
		genPolicy,
	))

	props.TestingRun(t)
}

func TestProperty_InvalidPoliciesRejected(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("invalid_max_attempts_rejected", prop.ForAll(
		func(maxAttempts int) bool {
			def := PolicyDefinition{
				MaxAttempts:   maxAttempts,
				BaseDelayMs:   100,
				MaxDelayMs:    1000,
				Multiplier:    2.0,
				JitterPercent: 0.1,
			}

			err := ValidatePolicy(def)
			// Should be invalid if < 1 or > 10
			shouldBeInvalid := maxAttempts < 1 || maxAttempts > 10
			return (err != nil) == shouldBeInvalid
		},
		gen.IntRange(-5, 15),
	))

	props.Property("invalid_multiplier_rejected", prop.ForAll(
		func(multiplier float64) bool {
			def := PolicyDefinition{
				MaxAttempts:   3,
				BaseDelayMs:   100,
				MaxDelayMs:    1000,
				Multiplier:    multiplier,
				JitterPercent: 0.1,
			}

			err := ValidatePolicy(def)
			// Should be invalid if < 1.0 or > 5.0
			shouldBeInvalid := multiplier < 1.0 || multiplier > 5.0
			return (err != nil) == shouldBeInvalid
		},
		gen.Float64Range(0.0, 7.0),
	))

	props.TestingRun(t)
}

func mustMarshal(def PolicyDefinition) []byte {
	cfg := &domain.RetryConfig{
		MaxAttempts:     def.MaxAttempts,
		BaseDelay:       time.Duration(def.BaseDelayMs) * time.Millisecond,
		MaxDelay:        time.Duration(def.MaxDelayMs) * time.Millisecond,
		Multiplier:      def.Multiplier,
		JitterPercent:   def.JitterPercent,
		RetryableErrors: def.RetryOn,
	}
	data, _ := MarshalPolicy(cfg)
	return data
}

func configsEqual(a, b *domain.RetryConfig) bool {
	if a.MaxAttempts != b.MaxAttempts {
		return false
	}
	if a.BaseDelay != b.BaseDelay {
		return false
	}
	if a.MaxDelay != b.MaxDelay {
		return false
	}
	if a.Multiplier != b.Multiplier {
		return false
	}
	if a.JitterPercent != b.JitterPercent {
		return false
	}
	if len(a.RetryableErrors) != len(b.RetryableErrors) {
		return false
	}
	for i, v := range a.RetryableErrors {
		if v != b.RetryableErrors[i] {
			return false
		}
	}
	return true
}
