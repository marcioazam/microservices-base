package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/testutil"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-microservice, Property 1: Circuit Breaker State Machine Correctness**
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4**
func TestProperty_CircuitBreakerStateMachine(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	// Property 1.1: Closed → Open when failures >= threshold
	props.Property("closed_to_open_on_failure_threshold", prop.ForAll(
		func(threshold int) bool {
			cb := New(Config{
				ServiceName: "test-service",
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 1,
					Timeout:          time.Second,
				},
			})

			// Record failures up to threshold
			for i := 0; i < threshold; i++ {
				if cb.GetState() == domain.StateOpen {
					return false // Should not be open yet
				}
				cb.RecordFailure()
			}

			return cb.GetState() == domain.StateOpen
		},
		gen.IntRange(1, 20),
	))

	// Property 1.2: Open → HalfOpen after timeout
	props.Property("open_to_halfopen_after_timeout", prop.ForAll(
		func(timeoutMs int) bool {
			timeout := time.Duration(timeoutMs) * time.Millisecond
			cb := New(Config{
				ServiceName: "test-service",
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: 1,
					SuccessThreshold: 1,
					Timeout:          timeout,
				},
			})

			// Force open
			cb.RecordFailure()
			if cb.GetState() != domain.StateOpen {
				return false
			}

			// Wait for timeout
			time.Sleep(timeout + 10*time.Millisecond)

			// Next request should transition to half-open
			_ = cb.Execute(context.Background(), func() error { return nil })

			return cb.GetState() == domain.StateClosed // Success in half-open closes it
		},
		gen.IntRange(10, 50),
	))

	// Property 1.3: HalfOpen → Closed on success threshold
	props.Property("halfopen_to_closed_on_success", prop.ForAll(
		func(successThreshold int) bool {
			cb := New(Config{
				ServiceName: "test-service",
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: 1,
					SuccessThreshold: successThreshold,
					Timeout:          time.Millisecond,
				},
			})

			// Force open then wait for half-open
			cb.RecordFailure()
			time.Sleep(2 * time.Millisecond)

			// Trigger transition to half-open
			cb.allowRequest()

			if cb.GetState() != domain.StateHalfOpen {
				return false
			}

			// Record successes
			for i := 0; i < successThreshold; i++ {
				cb.RecordSuccess()
			}

			return cb.GetState() == domain.StateClosed
		},
		gen.IntRange(1, 10),
	))

	// Property 1.4: HalfOpen → Open on any failure
	props.Property("halfopen_to_open_on_failure", prop.ForAll(
		func(successThreshold int) bool {
			cb := New(Config{
				ServiceName: "test-service",
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: 1,
					SuccessThreshold: successThreshold,
					Timeout:          time.Millisecond,
				},
			})

			// Force open then wait for half-open
			cb.RecordFailure()
			time.Sleep(2 * time.Millisecond)

			// Trigger transition to half-open
			cb.allowRequest()

			if cb.GetState() != domain.StateHalfOpen {
				return false
			}

			// Record a failure
			cb.RecordFailure()

			return cb.GetState() == domain.StateOpen
		},
		gen.IntRange(2, 10),
	))

	// Property: Execute returns circuit open error when open
	props.Property("execute_returns_error_when_open", prop.ForAll(
		func(threshold int) bool {
			cb := New(Config{
				ServiceName: "test-service",
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 1,
					Timeout:          time.Hour, // Long timeout
				},
			})

			// Force open
			for i := 0; i < threshold; i++ {
				cb.RecordFailure()
			}

			err := cb.Execute(context.Background(), func() error {
				return nil
			})

			var resErr *domain.ResilienceError
			return errors.As(err, &resErr) && resErr.Code == domain.ErrCircuitOpen
		},
		gen.IntRange(1, 10),
	))

	props.TestingRun(t)
}
