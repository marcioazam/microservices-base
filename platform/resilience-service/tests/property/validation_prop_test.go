package property

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"pgregory.net/rapid"
)

// TestProperty_ConfigValidation validates configuration validation.
func TestProperty_ConfigValidation(t *testing.T) {
	t.Run("circuit_breaker_valid_config", func(t *testing.T) {
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

	t.Run("retry_valid_config", func(t *testing.T) {
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

	t.Run("rate_limit_valid_config", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			limit := rapid.IntRange(1, 10000).Draw(t, "limit")
			burstSize := rapid.IntRange(1, limit).Draw(t, "burstSize")
			windowSec := rapid.IntRange(1, 3600).Draw(t, "windowSec")

			result := entities.NewRateLimitConfig(
				"token_bucket",
				limit,
				time.Duration(windowSec)*time.Second,
				burstSize,
			)

			if result.IsErr() {
				t.Fatalf("valid config rejected: %v", result.UnwrapErr())
			}
		})
	})

	t.Run("timeout_valid_config", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			timeoutMs := rapid.IntRange(100, 30000).Draw(t, "timeoutMs")
			maxTimeoutMs := rapid.IntRange(timeoutMs+1000, 600000).Draw(t, "maxTimeoutMs")

			result := entities.NewTimeoutConfig(
				time.Duration(timeoutMs)*time.Millisecond,
				time.Duration(maxTimeoutMs)*time.Millisecond,
			)

			if result.IsErr() {
				t.Fatalf("valid config rejected: %v", result.UnwrapErr())
			}
		})
	})
}

// TestProperty_InvalidConfigRejected validates invalid configs are rejected.
func TestProperty_InvalidConfigRejected(t *testing.T) {
	t.Run("circuit_breaker_invalid_threshold", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			failureThreshold := rapid.IntRange(-100, 0).Draw(t, "failureThreshold")

			result := entities.NewCircuitBreakerConfig(
				failureThreshold,
				3,
				30*time.Second,
				1,
			)

			if result.IsOk() {
				t.Fatal("invalid config should be rejected")
			}
		})
	})

	t.Run("retry_invalid_attempts", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxAttempts := rapid.IntRange(-100, 0).Draw(t, "maxAttempts")

			result := entities.NewRetryConfig(
				maxAttempts,
				100*time.Millisecond,
				10*time.Second,
				2.0,
				0.1,
			)

			if result.IsOk() {
				t.Fatal("invalid config should be rejected")
			}
		})
	})

	t.Run("rate_limit_invalid_limit", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			limit := rapid.IntRange(-100, 0).Draw(t, "limit")

			result := entities.NewRateLimitConfig(
				"token_bucket",
				limit,
				time.Minute,
				10,
			)

			if result.IsOk() {
				t.Fatal("invalid config should be rejected")
			}
		})
	})
}
