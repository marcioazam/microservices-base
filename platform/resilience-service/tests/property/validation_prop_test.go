package property

import (
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"pgregory.net/rapid"
)

// **Feature: platform-resilience-modernization, Property 4: Configuration Validation Correctness**
// **Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5**
func TestProperty_ConfigurationValidation(t *testing.T) {
	t.Run("valid_circuit_breaker_config_passes_validation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			failureThreshold := rapid.IntRange(1, 100).Draw(t, "failureThreshold")
			successThreshold := rapid.IntRange(1, 100).Draw(t, "successThreshold")
			timeoutMs := rapid.IntRange(100, 60000).Draw(t, "timeoutMs")

			cfg := resilience.CircuitBreakerConfig{
				FailureThreshold: failureThreshold,
				SuccessThreshold: successThreshold,
				Timeout:          time.Duration(timeoutMs) * time.Millisecond,
			}

			if err := cfg.Validate(); err != nil {
				t.Fatalf("valid config should pass validation: %v", err)
			}
		})
	})

	t.Run("valid_retry_config_passes_validation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxAttempts := rapid.IntRange(1, 10).Draw(t, "maxAttempts")
			baseDelayMs := rapid.IntRange(10, 1000).Draw(t, "baseDelayMs")
			maxDelayMs := rapid.IntRange(1001, 60000).Draw(t, "maxDelayMs")
			multiplier := rapid.Float64Range(1.0, 5.0).Draw(t, "multiplier")
			jitter := rapid.Float64Range(0.0, 1.0).Draw(t, "jitter")

			cfg := resilience.RetryConfig{
				MaxAttempts:   maxAttempts,
				BaseDelay:     time.Duration(baseDelayMs) * time.Millisecond,
				MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
				Multiplier:    multiplier,
				JitterPercent: jitter,
			}

			if err := cfg.Validate(); err != nil {
				t.Fatalf("valid config should pass validation: %v", err)
			}
		})
	})

	t.Run("valid_timeout_config_passes_validation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			timeoutMs := rapid.IntRange(100, 300000).Draw(t, "timeoutMs")

			cfg := resilience.TimeoutConfig{
				Default: time.Duration(timeoutMs) * time.Millisecond,
			}

			if err := cfg.Validate(); err != nil {
				t.Fatalf("valid config should pass validation: %v", err)
			}
		})
	})

	t.Run("valid_rate_limit_config_passes_validation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			limit := rapid.IntRange(1, 10000).Draw(t, "limit")
			windowMs := rapid.IntRange(1000, 60000).Draw(t, "windowMs")
			burstSize := rapid.IntRange(1, 1000).Draw(t, "burstSize")

			cfg := resilience.RateLimitConfig{
				Algorithm: resilience.TokenBucket,
				Limit:     limit,
				Window:    time.Duration(windowMs) * time.Millisecond,
				BurstSize: burstSize,
			}

			if err := cfg.Validate(); err != nil {
				t.Fatalf("valid config should pass validation: %v", err)
			}
		})
	})

	t.Run("valid_bulkhead_config_passes_validation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxConcurrent := rapid.IntRange(1, 1000).Draw(t, "maxConcurrent")
			maxQueue := rapid.IntRange(0, 1000).Draw(t, "maxQueue")
			timeoutMs := rapid.IntRange(100, 60000).Draw(t, "timeoutMs")

			cfg := resilience.BulkheadConfig{
				MaxConcurrent: maxConcurrent,
				MaxQueue:      maxQueue,
				QueueTimeout:  time.Duration(timeoutMs) * time.Millisecond,
			}

			if err := cfg.Validate(); err != nil {
				t.Fatalf("valid config should pass validation: %v", err)
			}
		})
	})

	t.Run("invalid_circuit_breaker_config_fails_validation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			failureThreshold := rapid.IntRange(-10, 5).Draw(t, "failureThreshold")
			successThreshold := rapid.IntRange(-10, 5).Draw(t, "successThreshold")
			timeoutMs := rapid.IntRange(-1000, 500).Draw(t, "timeoutMs")

			cfg := resilience.CircuitBreakerConfig{
				FailureThreshold: failureThreshold,
				SuccessThreshold: successThreshold,
				Timeout:          time.Duration(timeoutMs) * time.Millisecond,
			}

			err := cfg.Validate()
			shouldFail := failureThreshold <= 0 || successThreshold <= 0 || timeoutMs <= 0
			if (err != nil) != shouldFail {
				t.Fatalf("validation mismatch: err=%v, shouldFail=%v", err, shouldFail)
			}
		})
	})

	t.Run("invalid_retry_config_fails_validation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxAttempts := rapid.IntRange(-5, 5).Draw(t, "maxAttempts")
			baseDelayMs := rapid.IntRange(-100, 100).Draw(t, "baseDelayMs")
			maxDelayMs := rapid.IntRange(-100, 100).Draw(t, "maxDelayMs")
			multiplier := rapid.Float64Range(0.0, 2.0).Draw(t, "multiplier")
			jitter := rapid.Float64Range(-0.5, 1.5).Draw(t, "jitter")

			cfg := resilience.RetryConfig{
				MaxAttempts:   maxAttempts,
				BaseDelay:     time.Duration(baseDelayMs) * time.Millisecond,
				MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
				Multiplier:    multiplier,
				JitterPercent: jitter,
			}

			err := cfg.Validate()
			shouldFail := maxAttempts <= 0 || baseDelayMs <= 0 || maxDelayMs <= 0 || multiplier < 1.0 || jitter < 0 || jitter > 1
			if (err != nil) != shouldFail {
				t.Fatalf("validation mismatch: err=%v, shouldFail=%v", err, shouldFail)
			}
		})
	})
}
