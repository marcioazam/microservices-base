package property

import (
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/libs/go/resilience/retry"
	"pgregory.net/rapid"
)

// **Feature: resilience-service-state-of-art-2025, Property 7: Retry Policy Configuration Round-Trip**
// **Validates: Requirements 2.6**
func TestProperty_RetryPolicyConfigurationRoundTrip(t *testing.T) {
	t.Run("round_trip_preserves_policy", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxAttempts := rapid.IntRange(1, 10).Draw(t, "maxAttempts")
			baseDelayMs := rapid.IntRange(10, 500).Draw(t, "baseDelayMs")
			maxDelayMs := rapid.IntRange(1000, 10000).Draw(t, "maxDelayMs")
			multiplier := rapid.Float64Range(1.0, 5.0).Draw(t, "multiplier")
			jitterPercent := rapid.Float64Range(0.0, 0.5).Draw(t, "jitterPercent")

			if baseDelayMs >= maxDelayMs {
				return // Skip invalid configs
			}

			original := retry.PolicyDefinition{
				MaxAttempts:   maxAttempts,
				BaseDelayMs:   baseDelayMs,
				MaxDelayMs:    maxDelayMs,
				Multiplier:    multiplier,
				JitterPercent: jitterPercent,
				RetryOn:       []string{"error1", "error2"},
			}

			// Convert to RetryConfig
			cfg := retry.FromDefinition(original)

			// Serialize to JSON
			data, err := retry.MarshalPolicy(cfg)
			if err != nil {
				t.Fatalf("MarshalPolicy error: %v", err)
			}

			// Parse back
			restored, err := retry.ParsePolicy(data)
			if err != nil {
				t.Fatalf("ParsePolicy error: %v", err)
			}

			// Compare
			if cfg.MaxAttempts != restored.MaxAttempts {
				t.Fatalf("MaxAttempts mismatch: %d != %d", cfg.MaxAttempts, restored.MaxAttempts)
			}
			if cfg.BaseDelay != restored.BaseDelay {
				t.Fatalf("BaseDelay mismatch: %v != %v", cfg.BaseDelay, restored.BaseDelay)
			}
			if cfg.MaxDelay != restored.MaxDelay {
				t.Fatalf("MaxDelay mismatch: %v != %v", cfg.MaxDelay, restored.MaxDelay)
			}
			if cfg.Multiplier != restored.Multiplier {
				t.Fatalf("Multiplier mismatch: %f != %f", cfg.Multiplier, restored.Multiplier)
			}
			if cfg.JitterPercent != restored.JitterPercent {
				t.Fatalf("JitterPercent mismatch: %f != %f", cfg.JitterPercent, restored.JitterPercent)
			}

			// Compare RetryableErrors
			if len(cfg.RetryableErrors) != len(restored.RetryableErrors) {
				t.Fatalf("RetryableErrors length mismatch: %d != %d", len(cfg.RetryableErrors), len(restored.RetryableErrors))
			}
			for i, v := range cfg.RetryableErrors {
				if v != restored.RetryableErrors[i] {
					t.Fatalf("RetryableErrors[%d] mismatch: %s != %s", i, v, restored.RetryableErrors[i])
				}
			}
		})
	})

	t.Run("parse_then_marshal_then_parse_equals_original", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxAttempts := rapid.IntRange(1, 10).Draw(t, "maxAttempts")
			baseDelayMs := rapid.IntRange(10, 500).Draw(t, "baseDelayMs")
			maxDelayMs := rapid.IntRange(1000, 10000).Draw(t, "maxDelayMs")
			multiplier := rapid.Float64Range(1.0, 5.0).Draw(t, "multiplier")
			jitterPercent := rapid.Float64Range(0.0, 0.5).Draw(t, "jitterPercent")

			if baseDelayMs >= maxDelayMs {
				return // Skip invalid configs
			}

			original := retry.PolicyDefinition{
				MaxAttempts:   maxAttempts,
				BaseDelayMs:   baseDelayMs,
				MaxDelayMs:    maxDelayMs,
				Multiplier:    multiplier,
				JitterPercent: jitterPercent,
			}

			// First parse
			cfg1, err := retry.ParsePolicy(mustMarshalRetry(original))
			if err != nil {
				t.Fatalf("first ParsePolicy error: %v", err)
			}

			// Marshal
			data, err := retry.MarshalPolicy(cfg1)
			if err != nil {
				t.Fatalf("MarshalPolicy error: %v", err)
			}

			// Parse again
			cfg2, err := retry.ParsePolicy(data)
			if err != nil {
				t.Fatalf("second ParsePolicy error: %v", err)
			}

			// Should be equal
			if !retryConfigsEqual(cfg1, cfg2) {
				t.Fatal("configs not equal after round trip")
			}
		})
	})
}

func TestProperty_InvalidRetryPoliciesRejected(t *testing.T) {
	t.Run("invalid_max_attempts_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxAttempts := rapid.IntRange(-5, 15).Draw(t, "maxAttempts")

			def := retry.PolicyDefinition{
				MaxAttempts:   maxAttempts,
				BaseDelayMs:   100,
				MaxDelayMs:    1000,
				Multiplier:    2.0,
				JitterPercent: 0.1,
			}

			err := retry.ValidatePolicy(def)
			// Should be invalid if < 1 or > 10
			shouldBeInvalid := maxAttempts < 1 || maxAttempts > 10
			if (err != nil) != shouldBeInvalid {
				t.Fatalf("validation mismatch for maxAttempts=%d: err=%v, shouldBeInvalid=%v", maxAttempts, err, shouldBeInvalid)
			}
		})
	})

	t.Run("invalid_multiplier_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			multiplier := rapid.Float64Range(0.0, 7.0).Draw(t, "multiplier")

			def := retry.PolicyDefinition{
				MaxAttempts:   3,
				BaseDelayMs:   100,
				MaxDelayMs:    1000,
				Multiplier:    multiplier,
				JitterPercent: 0.1,
			}

			err := retry.ValidatePolicy(def)
			// Should be invalid if < 1.0 or > 5.0
			shouldBeInvalid := multiplier < 1.0 || multiplier > 5.0
			if (err != nil) != shouldBeInvalid {
				t.Fatalf("validation mismatch for multiplier=%f: err=%v, shouldBeInvalid=%v", multiplier, err, shouldBeInvalid)
			}
		})
	})
}

func mustMarshalRetry(def retry.PolicyDefinition) []byte {
	cfg := &resilience.RetryConfig{
		MaxAttempts:     def.MaxAttempts,
		BaseDelay:       time.Duration(def.BaseDelayMs) * time.Millisecond,
		MaxDelay:        time.Duration(def.MaxDelayMs) * time.Millisecond,
		Multiplier:      def.Multiplier,
		JitterPercent:   def.JitterPercent,
		RetryableErrors: def.RetryOn,
	}
	data, _ := retry.MarshalPolicy(cfg)
	return data
}

func retryConfigsEqual(a, b *resilience.RetryConfig) bool {
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
