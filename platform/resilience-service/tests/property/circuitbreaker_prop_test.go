package property

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"pgregory.net/rapid"
)

// TestProperty_CircuitBreakerConfigValidation validates circuit breaker configuration.
func TestProperty_CircuitBreakerConfigValidation(t *testing.T) {
	t.Run("valid_config_accepted", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			failureThreshold := rapid.IntRange(1, 20).Draw(t, "failureThreshold")
			successThreshold := rapid.IntRange(1, 10).Draw(t, "successThreshold")
			timeoutSec := rapid.IntRange(1, 60).Draw(t, "timeoutSec")

			// Ensure success <= failure for valid config
			if successThreshold > failureThreshold {
				successThreshold = failureThreshold
			}

			cfg := &entities.CircuitBreakerConfig{
				FailureThreshold: failureThreshold,
				SuccessThreshold: successThreshold,
				Timeout:          time.Duration(timeoutSec) * time.Second,
				ProbeCount:       2,
			}

			err := cfg.Validate()
			if err != nil {
				t.Fatalf("valid config rejected: %v", err)
			}
		})
	})

	t.Run("invalid_failure_threshold_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			failureThreshold := rapid.IntRange(-10, 0).Draw(t, "failureThreshold")

			cfg := &entities.CircuitBreakerConfig{
				FailureThreshold: failureThreshold,
				SuccessThreshold: 3,
				Timeout:          30 * time.Second,
				ProbeCount:       2,
			}

			err := cfg.Validate()
			if err == nil {
				t.Fatal("invalid config should be rejected")
			}
		})
	})

	t.Run("invalid_success_threshold_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			successThreshold := rapid.IntRange(-10, 0).Draw(t, "successThreshold")

			cfg := &entities.CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: successThreshold,
				Timeout:          30 * time.Second,
				ProbeCount:       2,
			}

			err := cfg.Validate()
			if err == nil {
				t.Fatal("invalid config should be rejected")
			}
		})
	})
}

// TestProperty_CircuitBreakerStateTransitions validates state transitions.
func TestProperty_CircuitBreakerStateTransitions(t *testing.T) {
	t.Run("state_string_representation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			stateInt := rapid.IntRange(0, 2).Draw(t, "state")
			state := domain.CircuitState(stateInt)

			str := state.String()
			switch state {
			case domain.StateClosed:
				if str != "CLOSED" {
					t.Fatalf("expected CLOSED, got %s", str)
				}
			case domain.StateOpen:
				if str != "OPEN" {
					t.Fatalf("expected OPEN, got %s", str)
				}
			case domain.StateHalfOpen:
				if str != "HALF_OPEN" {
					t.Fatalf("expected HALF_OPEN, got %s", str)
				}
			}
		})
	})
}
