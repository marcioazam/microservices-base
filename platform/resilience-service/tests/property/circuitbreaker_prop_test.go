package property

import (
	"context"
	"errors"
	"testing"
	"time"

	liberror "github.com/auth-platform/libs/go/error"
	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/libs/go/resilience/circuitbreaker"
	"pgregory.net/rapid"
)

// **Feature: resilience-service-state-of-art-2025, Property 1: Circuit Breaker State Machine Correctness**
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4**
func TestProperty_CircuitBreakerStateMachine(t *testing.T) {
	t.Run("closed_to_open_on_failure_threshold", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			threshold := rapid.IntRange(1, 20).Draw(t, "threshold")
			cb := circuitbreaker.New(circuitbreaker.Config{
				ServiceName: "test-service",
				Config: resilience.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 1,
					Timeout:          time.Second,
				},
			})

			for i := 0; i < threshold; i++ {
				if cb.GetState() == resilience.StateOpen {
					t.Fatalf("circuit opened before reaching threshold at iteration %d", i)
				}
				cb.RecordFailure()
			}

			if cb.GetState() != resilience.StateOpen {
				t.Fatalf("circuit should be open after %d failures", threshold)
			}
		})
	})

	t.Run("open_to_halfopen_after_timeout", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			timeoutMs := rapid.IntRange(10, 50).Draw(t, "timeoutMs")
			timeout := time.Duration(timeoutMs) * time.Millisecond
			cb := circuitbreaker.New(circuitbreaker.Config{
				ServiceName: "test-service",
				Config: resilience.CircuitBreakerConfig{
					FailureThreshold: 1,
					SuccessThreshold: 1,
					Timeout:          timeout,
				},
			})

			cb.RecordFailure()
			if cb.GetState() != resilience.StateOpen {
				t.Fatal("circuit should be open after failure")
			}

			time.Sleep(timeout + 10*time.Millisecond)

			_ = cb.Execute(context.Background(), func() error { return nil })

			if cb.GetState() != resilience.StateClosed {
				t.Fatal("circuit should be closed after successful execution in half-open state")
			}
		})
	})

	t.Run("execute_returns_error_when_open", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			threshold := rapid.IntRange(1, 10).Draw(t, "threshold")
			cb := circuitbreaker.New(circuitbreaker.Config{
				ServiceName: "test-service",
				Config: resilience.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 1,
					Timeout:          time.Hour,
				},
			})

			for i := 0; i < threshold; i++ {
				cb.RecordFailure()
			}

			err := cb.Execute(context.Background(), func() error {
				return nil
			})

			var resErr *liberror.ResilienceError
			if !errors.As(err, &resErr) || resErr.Code != liberror.ErrCircuitOpen {
				t.Fatalf("expected circuit open error, got %v", err)
			}
		})
	})
}
