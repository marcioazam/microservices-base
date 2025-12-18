package property

import (
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/platform/resilience-service/internal/policy"
	"pgregory.net/rapid"
)

// **Feature: resilience-service-state-of-art-2025, Property 20: Policy Validation Rejects Invalid Configurations**
// **Validates: Requirements 7.1**
func TestProperty_PolicyValidationRejectsInvalidConfigurations(t *testing.T) {
	engine := policy.NewEngine(policy.Config{})

	t.Run("empty_name_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			p := &resilience.ResiliencePolicy{
				Name: "",
			}
			err := engine.Validate(p)
			if err == nil {
				t.Fatal("expected error for empty name")
			}
		})
	})

	t.Run("negative_failure_threshold_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			threshold := rapid.IntRange(-10, 10).Draw(t, "threshold")

			p := &resilience.ResiliencePolicy{
				Name: "test",
				CircuitBreaker: &resilience.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 3,
					Timeout:          30 * time.Second,
				},
			}
			err := engine.Validate(p)
			shouldBeInvalid := threshold <= 0
			if (err != nil) != shouldBeInvalid {
				t.Fatalf("validation mismatch for threshold=%d: err=%v, shouldBeInvalid=%v", threshold, err, shouldBeInvalid)
			}
		})
	})

	t.Run("negative_retry_attempts_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			attempts := rapid.IntRange(-5, 10).Draw(t, "attempts")

			p := &resilience.ResiliencePolicy{
				Name: "test",
				Retry: &resilience.RetryConfig{
					MaxAttempts:   attempts,
					BaseDelay:     100 * time.Millisecond,
					MaxDelay:      10 * time.Second,
					Multiplier:    2.0,
					JitterPercent: 0.1,
				},
			}
			err := engine.Validate(p)
			shouldBeInvalid := attempts <= 0
			if (err != nil) != shouldBeInvalid {
				t.Fatalf("validation mismatch for attempts=%d: err=%v, shouldBeInvalid=%v", attempts, err, shouldBeInvalid)
			}
		})
	})

	t.Run("invalid_multiplier_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			multiplier := rapid.Float64Range(0.0, 3.0).Draw(t, "multiplier")

			p := &resilience.ResiliencePolicy{
				Name: "test",
				Retry: &resilience.RetryConfig{
					MaxAttempts:   3,
					BaseDelay:     100 * time.Millisecond,
					MaxDelay:      10 * time.Second,
					Multiplier:    multiplier,
					JitterPercent: 0.1,
				},
			}
			err := engine.Validate(p)
			shouldBeInvalid := multiplier < 1.0
			if (err != nil) != shouldBeInvalid {
				t.Fatalf("validation mismatch for multiplier=%f: err=%v, shouldBeInvalid=%v", multiplier, err, shouldBeInvalid)
			}
		})
	})

	t.Run("negative_rate_limit_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			limit := rapid.IntRange(-10, 10).Draw(t, "limit")

			p := &resilience.ResiliencePolicy{
				Name: "test",
				RateLimit: &resilience.RateLimitConfig{
					Limit:  limit,
					Window: time.Minute,
				},
			}
			err := engine.Validate(p)
			shouldBeInvalid := limit <= 0
			if (err != nil) != shouldBeInvalid {
				t.Fatalf("validation mismatch for limit=%d: err=%v, shouldBeInvalid=%v", limit, err, shouldBeInvalid)
			}
		})
	})

	t.Run("negative_bulkhead_concurrent_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxConcurrent := rapid.IntRange(-10, 10).Draw(t, "maxConcurrent")

			p := &resilience.ResiliencePolicy{
				Name: "test",
				Bulkhead: &resilience.BulkheadConfig{
					MaxConcurrent: maxConcurrent,
					MaxQueue:      10,
				},
			}
			err := engine.Validate(p)
			shouldBeInvalid := maxConcurrent <= 0
			if (err != nil) != shouldBeInvalid {
				t.Fatalf("validation mismatch for maxConcurrent=%d: err=%v, shouldBeInvalid=%v", maxConcurrent, err, shouldBeInvalid)
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 21: Policy Definition Round-Trip**
// **Validates: Requirements 7.4**
func TestProperty_PolicyDefinitionRoundTrip(t *testing.T) {
	t.Run("round_trip_preserves_policy", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			name := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{1,20}`).Draw(t, "name")
			version := rapid.Int64Range(1, 100).Draw(t, "version")
			failureThreshold := rapid.IntRange(1, 10).Draw(t, "failureThreshold")
			successThreshold := rapid.IntRange(1, 10).Draw(t, "successThreshold")

			original := &resilience.ResiliencePolicy{
				Name:    name,
				Version: version,
				CircuitBreaker: &resilience.CircuitBreakerConfig{
					FailureThreshold: failureThreshold,
					SuccessThreshold: successThreshold,
					Timeout:          30 * time.Second,
				},
				Retry: &resilience.RetryConfig{
					MaxAttempts:   3,
					BaseDelay:     100 * time.Millisecond,
					MaxDelay:      10 * time.Second,
					Multiplier:    2.0,
					JitterPercent: 0.1,
				},
			}

			// Serialize
			data, err := policy.MarshalPolicy(original)
			if err != nil {
				t.Fatalf("MarshalPolicy error: %v", err)
			}

			// Deserialize
			restored, err := policy.UnmarshalPolicy(data)
			if err != nil {
				t.Fatalf("UnmarshalPolicy error: %v", err)
			}

			// Compare key fields
			if original.Name != restored.Name {
				t.Fatalf("name mismatch: %s != %s", original.Name, restored.Name)
			}
			if original.Version != restored.Version {
				t.Fatalf("version mismatch: %d != %d", original.Version, restored.Version)
			}

			// Compare circuit breaker
			if original.CircuitBreaker.FailureThreshold != restored.CircuitBreaker.FailureThreshold {
				t.Fatalf("failure threshold mismatch")
			}
			if original.CircuitBreaker.SuccessThreshold != restored.CircuitBreaker.SuccessThreshold {
				t.Fatalf("success threshold mismatch")
			}
		})
	})

	t.Run("serialize_then_deserialize_then_serialize_equals", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			name := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{1,20}`).Draw(t, "name")

			original := &resilience.ResiliencePolicy{
				Name:    name,
				Version: 1,
				CircuitBreaker: &resilience.CircuitBreakerConfig{
					FailureThreshold: 5,
					SuccessThreshold: 3,
					Timeout:          30 * time.Second,
				},
			}

			// First serialize
			data1, err := policy.MarshalPolicy(original)
			if err != nil {
				t.Fatalf("first MarshalPolicy error: %v", err)
			}

			// Deserialize
			restored, err := policy.UnmarshalPolicy(data1)
			if err != nil {
				t.Fatalf("UnmarshalPolicy error: %v", err)
			}

			// Serialize again
			data2, err := policy.MarshalPolicy(restored)
			if err != nil {
				t.Fatalf("second MarshalPolicy error: %v", err)
			}

			// Should produce same JSON
			if string(data1) != string(data2) {
				t.Fatal("round trip produced different JSON")
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 12: Policy Reload Validation**
// **Validates: Requirements 15.2, 15.3**
func TestProperty_PolicyReloadValidation(t *testing.T) {
	t.Run("invalid_policy_update_preserves_existing", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			validName := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{1,20}`).Draw(t, "validName")
			invalidThreshold := rapid.IntRange(-10, 0).Draw(t, "invalidThreshold")

			engine := policy.NewEngine(policy.Config{})

			// Create and add a valid policy
			validPolicy := &resilience.ResiliencePolicy{
				Name: validName,
				CircuitBreaker: &resilience.CircuitBreakerConfig{
					FailureThreshold: 5,
					SuccessThreshold: 3,
					Timeout:          30 * time.Second,
				},
			}
			if err := engine.UpdatePolicy(validPolicy); err != nil {
				t.Fatalf("failed to add valid policy: %v", err)
			}

			// Try to update with invalid policy
			invalidPolicy := &resilience.ResiliencePolicy{
				Name: validName,
				CircuitBreaker: &resilience.CircuitBreakerConfig{
					FailureThreshold: invalidThreshold, // Invalid
					SuccessThreshold: 3,
					Timeout:          30 * time.Second,
				},
			}
			err := engine.UpdatePolicy(invalidPolicy)

			// Should fail
			if err == nil {
				t.Fatal("expected error for invalid policy")
			}

			// Original policy should still be there
			existing, getErr := engine.GetPolicy(validName)
			if getErr != nil {
				t.Fatalf("failed to get existing policy: %v", getErr)
			}

			// Should have original values
			if existing.CircuitBreaker.FailureThreshold != 5 {
				t.Fatal("original policy was modified")
			}
		})
	})
}
