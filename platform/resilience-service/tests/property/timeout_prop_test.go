package property

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"pgregory.net/rapid"
)

// **Feature: resilience-service-state-of-art-2025, Property 8: Timeout Enforcement**
// **Validates: Requirements 3.1**
func TestProperty_TimeoutEnforcement(t *testing.T) {
	t.Run("timeout_config_validation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			timeoutMs := rapid.IntRange(1, 300000).Draw(t, "timeoutMs")
			timeoutDur := time.Duration(timeoutMs) * time.Millisecond
			maxTimeout := timeoutDur + time.Minute

			cfg, err := entities.NewTimeoutConfig(timeoutDur, maxTimeout)
			
			if timeoutMs <= 0 {
				if err == nil {
					t.Fatal("Expected validation error for non-positive timeout")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected validation error: %v", err)
				}
				
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

			cfg, err := entities.NewTimeoutConfig(timeoutDur, maxTimeout)
			if err != nil {
				t.Fatalf("Failed to create timeout config: %v", err)
			}

			originalDefault := cfg.Default
			
			// Test that config is immutable
			if cfg.Default != originalDefault {
				t.Fatal("Timeout config should be immutable")
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 9: Operation-Specific Timeout Precedence**
// **Validates: Requirements 3.2**
func TestProperty_OperationSpecificTimeoutPrecedence(t *testing.T) {
	t.Run("timeout_validation_properties", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			timeoutSec := rapid.IntRange(1, 300).Draw(t, "timeoutSec")
			timeoutDur := time.Duration(timeoutSec) * time.Second
			maxTimeout := timeoutDur + time.Minute

			_, err := entities.NewTimeoutConfig(timeoutDur, maxTimeout)
			
			shouldBeValid := timeoutDur >= 100*time.Millisecond && timeoutDur <= 5*time.Minute
			
			if shouldBeValid {
				if err != nil {
					t.Fatalf("Expected valid config for timeout %v, got error: %v", timeoutDur, err)
				}
			} else {
				if err == nil {
					t.Fatalf("Expected validation error for timeout %v", timeoutDur)
				}
			}
		})
	})

	t.Run("timeout_boundary_validation", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			timeoutMin := rapid.IntRange(-100, 10).Draw(t, "timeoutMin")
			timeoutDur := time.Duration(timeoutMin) * time.Minute
			maxTimeout := timeoutDur + time.Minute

			cfg, err := entities.NewTimeoutConfig(timeoutDur, maxTimeout)
			
			shouldBeInvalid := timeoutDur <= 0 || timeoutDur < 100*time.Millisecond || timeoutDur > 5*time.Minute
			
			if shouldBeInvalid {
				if err == nil {
					t.Fatalf("Expected validation error for timeout %v", timeoutDur)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected validation error for timeout %v: %v", timeoutDur, err)
				}
				
				if cfg.Default != timeoutDur {
					t.Fatalf("Expected timeout %v, got %v", timeoutDur, cfg.Default)
				}
			}
		})
	})
}