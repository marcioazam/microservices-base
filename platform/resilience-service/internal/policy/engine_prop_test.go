package policy

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/testutil"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-microservice, Property 20: Policy Validation Rejects Invalid Configurations**
// **Validates: Requirements 7.1**
func TestProperty_PolicyValidationRejectsInvalidConfigurations(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	engine := NewEngine(Config{})

	props.Property("empty_name_rejected", prop.ForAll(
		func(_ int) bool {
			policy := &domain.ResiliencePolicy{
				Name: "",
			}
			err := engine.Validate(policy)
			return err != nil
		},
		gen.Int(),
	))

	props.Property("negative_failure_threshold_rejected", prop.ForAll(
		func(threshold int) bool {
			policy := &domain.ResiliencePolicy{
				Name: "test",
				CircuitBreaker: &domain.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 3,
					Timeout:          30 * time.Second,
				},
			}
			err := engine.Validate(policy)
			shouldBeInvalid := threshold <= 0
			return (err != nil) == shouldBeInvalid
		},
		gen.IntRange(-10, 10),
	))

	props.Property("negative_retry_attempts_rejected", prop.ForAll(
		func(attempts int) bool {
			policy := &domain.ResiliencePolicy{
				Name: "test",
				Retry: &domain.RetryConfig{
					MaxAttempts:   attempts,
					BaseDelay:     100 * time.Millisecond,
					MaxDelay:      10 * time.Second,
					Multiplier:    2.0,
					JitterPercent: 0.1,
				},
			}
			err := engine.Validate(policy)
			shouldBeInvalid := attempts <= 0
			return (err != nil) == shouldBeInvalid
		},
		gen.IntRange(-5, 10),
	))

	props.Property("invalid_multiplier_rejected", prop.ForAll(
		func(multiplier float64) bool {
			policy := &domain.ResiliencePolicy{
				Name: "test",
				Retry: &domain.RetryConfig{
					MaxAttempts:   3,
					BaseDelay:     100 * time.Millisecond,
					MaxDelay:      10 * time.Second,
					Multiplier:    multiplier,
					JitterPercent: 0.1,
				},
			}
			err := engine.Validate(policy)
			shouldBeInvalid := multiplier < 1.0
			return (err != nil) == shouldBeInvalid
		},
		gen.Float64Range(0.0, 3.0),
	))

	props.Property("negative_rate_limit_rejected", prop.ForAll(
		func(limit int) bool {
			policy := &domain.ResiliencePolicy{
				Name: "test",
				RateLimit: &domain.RateLimitConfig{
					Limit:  limit,
					Window: time.Minute,
				},
			}
			err := engine.Validate(policy)
			shouldBeInvalid := limit <= 0
			return (err != nil) == shouldBeInvalid
		},
		gen.IntRange(-10, 10),
	))

	props.Property("negative_bulkhead_concurrent_rejected", prop.ForAll(
		func(maxConcurrent int) bool {
			policy := &domain.ResiliencePolicy{
				Name: "test",
				Bulkhead: &domain.BulkheadConfig{
					MaxConcurrent: maxConcurrent,
					MaxQueue:      10,
				},
			}
			err := engine.Validate(policy)
			shouldBeInvalid := maxConcurrent <= 0
			return (err != nil) == shouldBeInvalid
		},
		gen.IntRange(-10, 10),
	))

	props.TestingRun(t)
}

// **Feature: resilience-microservice, Property 21: Policy Definition Round-Trip**
// **Validates: Requirements 7.4**
func TestProperty_PolicyDefinitionRoundTrip(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	// Generator for valid policies
	genPolicy := gopter.CombineGens(
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		gen.IntRange(1, 100),
		gen.IntRange(1, 10),
		gen.IntRange(1, 10),
	).Map(func(vals []interface{}) *domain.ResiliencePolicy {
		return &domain.ResiliencePolicy{
			Name:    vals[0].(string),
			Version: vals[1].(int),
			CircuitBreaker: &domain.CircuitBreakerConfig{
				FailureThreshold: vals[2].(int),
				SuccessThreshold: vals[3].(int),
				Timeout:          30 * time.Second,
			},
			Retry: &domain.RetryConfig{
				MaxAttempts:   3,
				BaseDelay:     100 * time.Millisecond,
				MaxDelay:      10 * time.Second,
				Multiplier:    2.0,
				JitterPercent: 0.1,
			},
			Timeout: &domain.TimeoutConfig{
				Default: 5 * time.Second,
			},
			RateLimit: &domain.RateLimitConfig{
				Algorithm: domain.TokenBucket,
				Limit:     1000,
				Window:    time.Minute,
				BurstSize: 100,
			},
			Bulkhead: &domain.BulkheadConfig{
				MaxConcurrent: 100,
				MaxQueue:      50,
				QueueTimeout:  5 * time.Second,
			},
		}
	})

	props.Property("round_trip_preserves_policy", prop.ForAll(
		func(original *domain.ResiliencePolicy) bool {
			// Serialize
			data, err := MarshalPolicy(original)
			if err != nil {
				return false
			}

			// Deserialize
			restored, err := UnmarshalPolicy(data)
			if err != nil {
				return false
			}

			// Compare key fields
			if original.Name != restored.Name {
				return false
			}
			if original.Version != restored.Version {
				return false
			}

			// Compare circuit breaker
			if original.CircuitBreaker != nil && restored.CircuitBreaker != nil {
				if original.CircuitBreaker.FailureThreshold != restored.CircuitBreaker.FailureThreshold {
					return false
				}
				if original.CircuitBreaker.SuccessThreshold != restored.CircuitBreaker.SuccessThreshold {
					return false
				}
			}

			// Compare retry
			if original.Retry != nil && restored.Retry != nil {
				if original.Retry.MaxAttempts != restored.Retry.MaxAttempts {
					return false
				}
				if original.Retry.Multiplier != restored.Retry.Multiplier {
					return false
				}
			}

			return true
		},
		genPolicy,
	))

	props.Property("serialize_then_deserialize_then_serialize_equals", prop.ForAll(
		func(original *domain.ResiliencePolicy) bool {
			// First serialize
			data1, err := MarshalPolicy(original)
			if err != nil {
				return false
			}

			// Deserialize
			restored, err := UnmarshalPolicy(data1)
			if err != nil {
				return false
			}

			// Serialize again
			data2, err := MarshalPolicy(restored)
			if err != nil {
				return false
			}

			// Should produce same JSON
			return string(data1) == string(data2)
		},
		genPolicy,
	))

	props.TestingRun(t)
}
