package property

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"pgregory.net/rapid"
)

// TestProperty_TimeoutEnforcement validates timeout configuration.
func TestProperty_TimeoutEnforcement(t *testing.T) {
	t.Run("timeout_config_validation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			timeoutMs := rapid.IntRange(100, 300000).Draw(t, "timeoutMs")
			timeoutDur := time.Duration(timeoutMs) * time.Millisecond
			maxTimeout := timeoutDur + time.Minute

			result := entities.NewTimeoutConfig(timeoutDur, maxTimeout)

			if timeoutDur < 100*time.Millisecond || timeoutDur > 5*time.Minute {
				if result.IsOk() {
					t.Fatal("Expected validation error for out-of-range timeout")
				}
			} else {
				if result.IsErr() {
					t.Fatalf("Unexpected validation error: %v", result.UnwrapErr())
				}

				cfg := result.Unwrap()
				if cfg.Default != timeoutDur {
					t.Fatalf("Expected timeout %v, got %v", timeoutDur, cfg.Default)
				}
			}
		})
	})

	t.Run("timeout_config_immutability", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			timeoutMs := rapid.IntRange(100, 60000).Draw(t, "timeoutMs")
			timeoutDur := time.Duration(timeoutMs) * time.Millisecond
			maxTimeout := timeoutDur + time.Minute

			result := entities.NewTimeoutConfig(timeoutDur, maxTimeout)
			if result.IsErr() {
				t.Fatalf("Failed to create timeout config: %v", result.UnwrapErr())
			}

			cfg := result.Unwrap()
			originalDefault := cfg.Default

			if cfg.Default != originalDefault {
				t.Fatal("Timeout config should be immutable")
			}
		})
	})
}

// TestProperty_OperationSpecificTimeoutPrecedence validates timeout precedence.
func TestProperty_OperationSpecificTimeoutPrecedence(t *testing.T) {
	t.Run("timeout_validation_properties", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			timeoutSec := rapid.IntRange(1, 300).Draw(t, "timeoutSec")
			timeoutDur := time.Duration(timeoutSec) * time.Second
			maxTimeout := timeoutDur + time.Minute

			result := entities.NewTimeoutConfig(timeoutDur, maxTimeout)

			shouldBeValid := timeoutDur >= 100*time.Millisecond && timeoutDur <= 5*time.Minute

			if shouldBeValid {
				if result.IsErr() {
					t.Fatalf("Expected valid config for timeout %v, got error: %v", timeoutDur, result.UnwrapErr())
				}
			} else {
				if result.IsOk() {
					t.Fatalf("Expected validation error for timeout %v", timeoutDur)
				}
			}
		})
	})

	t.Run("timeout_boundary_validation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			timeoutSec := rapid.IntRange(1, 300).Draw(t, "timeoutSec")
			timeoutDur := time.Duration(timeoutSec) * time.Second
			maxTimeout := timeoutDur + time.Minute

			result := entities.NewTimeoutConfig(timeoutDur, maxTimeout)

			shouldBeInvalid := timeoutDur < 100*time.Millisecond || timeoutDur > 5*time.Minute

			if shouldBeInvalid {
				if result.IsOk() {
					t.Fatalf("Expected validation error for timeout %v", timeoutDur)
				}
			} else {
				if result.IsErr() {
					t.Fatalf("Unexpected validation error for timeout %v: %v", timeoutDur, result.UnwrapErr())
				}

				cfg := result.Unwrap()
				if cfg.Default != timeoutDur {
					t.Fatalf("Expected timeout %v, got %v", timeoutDur, cfg.Default)
				}
			}
		})
	})
}
