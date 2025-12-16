package timeout

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/testutil"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-microservice, Property 8: Timeout Enforcement**
// **Validates: Requirements 3.1**
func TestProperty_TimeoutEnforcement(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("operation_cancelled_after_timeout", prop.ForAll(
		func(timeoutMs int) bool {
			timeout := time.Duration(timeoutMs) * time.Millisecond

			cfg := domain.TimeoutConfig{
				Default: timeout,
			}

			manager := New(Config{
				ServiceName: "test-service",
				Config:      cfg,
			})

			// Operation that takes longer than timeout
			operationDuration := timeout + 100*time.Millisecond

			start := time.Now()
			err := manager.Execute(context.Background(), "test-op", func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(operationDuration):
					return nil
				}
			})
			elapsed := time.Since(start)

			// Should have timed out
			if err == nil {
				return false
			}

			// Should be a timeout error
			resErr, ok := err.(*domain.ResilienceError)
			if !ok || resErr.Code != domain.ErrTimeout {
				return false
			}

			// Should have completed within timeout + small epsilon
			epsilon := 50 * time.Millisecond
			return elapsed < timeout+epsilon
		},
		gen.IntRange(10, 100),
	))

	props.Property("successful_operation_completes", prop.ForAll(
		func(timeoutMs int) bool {
			timeout := time.Duration(timeoutMs) * time.Millisecond

			cfg := domain.TimeoutConfig{
				Default: timeout,
			}

			manager := New(Config{
				ServiceName: "test-service",
				Config:      cfg,
			})

			// Operation that completes quickly
			err := manager.Execute(context.Background(), "test-op", func(ctx context.Context) error {
				return nil
			})

			return err == nil
		},
		gen.IntRange(50, 200),
	))

	props.TestingRun(t)
}

// **Feature: resilience-microservice, Property 9: Operation-Specific Timeout Precedence**
// **Validates: Requirements 3.2**
func TestProperty_OperationSpecificTimeoutPrecedence(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("operation_specific_overrides_default", prop.ForAll(
		func(defaultMs int, specificMs int) bool {
			defaultTimeout := time.Duration(defaultMs) * time.Millisecond
			specificTimeout := time.Duration(specificMs) * time.Millisecond

			cfg := domain.TimeoutConfig{
				Default: defaultTimeout,
				PerOp: map[string]time.Duration{
					"specific-op": specificTimeout,
				},
			}

			manager := New(Config{
				ServiceName: "test-service",
				Config:      cfg,
			})

			// Operation-specific should return specific timeout
			actualSpecific := manager.GetTimeout("specific-op")
			if actualSpecific != specificTimeout {
				return false
			}

			// Other operations should return default
			actualDefault := manager.GetTimeout("other-op")
			if actualDefault != defaultTimeout {
				return false
			}

			return true
		},
		gen.IntRange(100, 500),
		gen.IntRange(50, 200),
	))

	props.TestingRun(t)
}

// **Feature: resilience-microservice, Property 10: Timeout Configuration Validation**
// **Validates: Requirements 3.4**
func TestProperty_TimeoutConfigurationValidation(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("non_positive_timeout_rejected", prop.ForAll(
		func(timeoutMs int) bool {
			timeout := time.Duration(timeoutMs) * time.Millisecond

			cfg := domain.TimeoutConfig{
				Default: timeout,
			}

			err := ValidateConfig(cfg)

			// Should be invalid if <= 0
			shouldBeInvalid := timeoutMs <= 0
			return (err != nil) == shouldBeInvalid
		},
		gen.IntRange(-100, 100),
	))

	props.Property("exceeds_max_timeout_rejected", prop.ForAll(
		func(timeoutMin int) bool {
			// Create timeout that exceeds max (5 minutes)
			timeout := time.Duration(timeoutMin) * time.Minute

			cfg := domain.TimeoutConfig{
				Default: timeout,
			}

			err := ValidateConfig(cfg)

			// Should be invalid if > 5 minutes
			shouldBeInvalid := timeout > MaxTimeout
			return (err != nil) == shouldBeInvalid
		},
		gen.IntRange(1, 10),
	))

	props.Property("valid_timeout_accepted", prop.ForAll(
		func(timeoutSec int) bool {
			timeout := time.Duration(timeoutSec) * time.Second

			cfg := domain.TimeoutConfig{
				Default: timeout,
			}

			err := ValidateConfig(cfg)

			// Should be valid if within bounds
			shouldBeValid := timeout > 0 && timeout <= MaxTimeout
			return (err == nil) == shouldBeValid
		},
		gen.IntRange(1, 300),
	))

	props.TestingRun(t)
}
