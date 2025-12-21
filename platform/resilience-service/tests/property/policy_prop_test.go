package property

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"pgregory.net/rapid"
)

// TestProperty_PolicyValidationRejectsInvalidConfigurations validates policy validation.
func TestProperty_PolicyValidationRejectsInvalidConfigurations(t *testing.T) {
	t.Run("empty_name_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			_, err := entities.NewPolicy("")
			if err == nil {
				t.Fatal("expected error for empty name")
			}
		})
	})

	t.Run("negative_failure_threshold_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			threshold := rapid.IntRange(-10, 0).Draw(t, "threshold")

			result := entities.NewCircuitBreakerConfig(threshold, 3, 30*time.Second, 2)
			if result.IsOk() {
				t.Fatalf("expected validation error for threshold=%d", threshold)
			}
		})
	})

	t.Run("negative_retry_attempts_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			attempts := rapid.IntRange(-5, 0).Draw(t, "attempts")

			result := entities.NewRetryConfig(attempts, 100*time.Millisecond, 10*time.Second, 2.0, 0.1)
			if result.IsOk() {
				t.Fatalf("expected validation error for attempts=%d", attempts)
			}
		})
	})

	t.Run("negative_rate_limit_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			limit := rapid.IntRange(-10, 0).Draw(t, "limit")

			result := entities.NewRateLimitConfig("token_bucket", limit, time.Minute, 10)
			if result.IsOk() {
				t.Fatalf("expected validation error for limit=%d", limit)
			}
		})
	})

	t.Run("negative_bulkhead_concurrent_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxConcurrent := rapid.IntRange(-10, 0).Draw(t, "maxConcurrent")

			result := entities.NewBulkheadConfig(maxConcurrent, 10, time.Second)
			if result.IsOk() {
				t.Fatalf("expected validation error for maxConcurrent=%d", maxConcurrent)
			}
		})
	})
}

// TestProperty_PolicyClonePreservesData validates clone preserves data.
func TestProperty_PolicyClonePreservesData(t *testing.T) {
	t.Run("clone_preserves_policy", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			name := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{1,20}`).Draw(t, "name")
			failureThreshold := rapid.IntRange(1, 100).Draw(t, "failureThreshold")
			successThreshold := rapid.IntRange(1, min(10, failureThreshold)).Draw(t, "successThreshold")

			original, err := entities.NewPolicy(name)
			if err != nil {
				t.Fatalf("NewPolicy error: %v", err)
			}

			cbResult := entities.NewCircuitBreakerConfig(failureThreshold, successThreshold, 30*time.Second, 2)
			if cbResult.IsErr() {
				t.Fatalf("NewCircuitBreakerConfig error: %v", cbResult.UnwrapErr())
			}

			result := original.SetCircuitBreaker(cbResult.Unwrap())
			if result.IsErr() {
				t.Fatalf("SetCircuitBreaker error: %v", result.UnwrapErr())
			}

			clone := original.Clone()

			if original.Name() != clone.Name() {
				t.Fatalf("name mismatch: %s != %s", original.Name(), clone.Name())
			}

			if original.Version() != clone.Version() {
				t.Fatalf("version mismatch: %d != %d", original.Version(), clone.Version())
			}

			if original.CircuitBreaker().IsSome() != clone.CircuitBreaker().IsSome() {
				t.Fatal("circuit breaker presence mismatch")
			}
		})
	})
}

// TestProperty_PolicyValidConfigurations validates valid configurations are accepted.
func TestProperty_PolicyValidConfigurations(t *testing.T) {
	t.Run("valid_circuit_breaker_accepted", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			failureThreshold := rapid.IntRange(1, 100).Draw(t, "failureThreshold")
			successThreshold := rapid.IntRange(1, min(10, failureThreshold)).Draw(t, "successThreshold")
			timeoutSec := rapid.IntRange(1, 300).Draw(t, "timeoutSec")
			probeCount := rapid.IntRange(1, 10).Draw(t, "probeCount")

			result := entities.NewCircuitBreakerConfig(
				failureThreshold,
				successThreshold,
				time.Duration(timeoutSec)*time.Second,
				probeCount,
			)

			if result.IsErr() {
				t.Fatalf("valid config rejected: %v", result.UnwrapErr())
			}
		})
	})

	t.Run("valid_retry_accepted", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxAttempts := rapid.IntRange(1, 10).Draw(t, "maxAttempts")
			baseDelayMs := rapid.IntRange(1, 1000).Draw(t, "baseDelayMs")
			maxDelayMs := rapid.IntRange(1000, 60000).Draw(t, "maxDelayMs")
			multiplier := rapid.Float64Range(1.0, 5.0).Draw(t, "multiplier")

			result := entities.NewRetryConfig(
				maxAttempts,
				time.Duration(baseDelayMs)*time.Millisecond,
				time.Duration(maxDelayMs)*time.Millisecond,
				multiplier,
				0.1,
			)

			if result.IsErr() {
				t.Fatalf("valid config rejected: %v", result.UnwrapErr())
			}
		})
	})
}
