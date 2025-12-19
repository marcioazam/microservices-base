package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 24: Circuit Breaker State Transitions**
// **Validates: Requirements 21.4, 21.5, 21.6, 21.7**
func TestCircuitBreakerStateTransitions(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Closed -> Open on failure threshold
	properties.Property("transitions from Closed to Open on failure threshold", prop.ForAll(
		func(threshold int) bool {
			if threshold < 1 || threshold > 10 {
				return true
			}
			cb := New[int]("test",
				WithFailureThreshold(threshold),
				WithTimeout(time.Hour),
			)

			for i := 0; i < threshold; i++ {
				cb.Execute(context.Background(), func() (int, error) {
					return 0, errors.New("fail")
				})
			}

			return cb.State() == StateOpen
		},
		gen.IntRange(1, 10),
	))

	// Open -> HalfOpen on timeout
	properties.Property("transitions from Open to HalfOpen on timeout", prop.ForAll(
		func(timeoutMs int) bool {
			if timeoutMs < 1 || timeoutMs > 100 {
				return true
			}
			timeout := time.Duration(timeoutMs) * time.Millisecond
			cb := New[int]("test",
				WithFailureThreshold(1),
				WithTimeout(timeout),
			)

			// Trigger open state
			cb.Execute(context.Background(), func() (int, error) {
				return 0, errors.New("fail")
			})

			if cb.State() != StateOpen {
				return false
			}

			// Wait for timeout
			time.Sleep(timeout + 10*time.Millisecond)

			return cb.State() == StateHalfOpen
		},
		gen.IntRange(1, 50),
	))

	// HalfOpen -> Closed on success threshold
	properties.Property("transitions from HalfOpen to Closed on success threshold", prop.ForAll(
		func(successThreshold int) bool {
			if successThreshold < 1 || successThreshold > 5 {
				return true
			}
			cb := New[int]("test",
				WithFailureThreshold(1),
				WithSuccessThreshold(successThreshold),
				WithTimeout(time.Millisecond),
				WithHalfOpenMaxCalls(successThreshold+1),
			)

			// Trigger open state
			cb.Execute(context.Background(), func() (int, error) {
				return 0, errors.New("fail")
			})

			// Wait for half-open
			time.Sleep(5 * time.Millisecond)

			// Succeed enough times
			for i := 0; i < successThreshold; i++ {
				cb.Execute(context.Background(), func() (int, error) {
					return 42, nil
				})
			}

			return cb.State() == StateClosed
		},
		gen.IntRange(1, 5),
	))

	// HalfOpen -> Open on failure
	properties.Property("transitions from HalfOpen to Open on failure", prop.ForAll(
		func(_ int) bool {
			cb := New[int]("test",
				WithFailureThreshold(1),
				WithTimeout(time.Millisecond),
			)

			// Trigger open state
			cb.Execute(context.Background(), func() (int, error) {
				return 0, errors.New("fail")
			})

			// Wait for half-open
			time.Sleep(5 * time.Millisecond)

			if cb.State() != StateHalfOpen {
				return false
			}

			// Fail in half-open
			cb.Execute(context.Background(), func() (int, error) {
				return 0, errors.New("fail again")
			})

			return cb.State() == StateOpen
		},
		gen.Int(),
	))

	properties.TestingRun(t)
}

func TestCircuitBreakerBasicOperations(t *testing.T) {
	t.Run("New creates circuit breaker in closed state", func(t *testing.T) {
		cb := New[int]("test")
		if cb.State() != StateClosed {
			t.Errorf("expected closed state, got %v", cb.State())
		}
	})

	t.Run("Execute passes through in closed state", func(t *testing.T) {
		cb := New[int]("test")
		result, err := cb.Execute(context.Background(), func() (int, error) {
			return 42, nil
		})
		if err != nil || result != 42 {
			t.Errorf("expected 42, got %d, err: %v", result, err)
		}
	})

	t.Run("Execute returns error when open", func(t *testing.T) {
		cb := New[int]("test", WithFailureThreshold(1), WithTimeout(time.Hour))

		// Trigger open
		cb.Execute(context.Background(), func() (int, error) {
			return 0, errors.New("fail")
		})

		_, err := cb.Execute(context.Background(), func() (int, error) {
			return 42, nil
		})

		if !errors.Is(err, ErrCircuitOpen) {
			t.Errorf("expected ErrCircuitOpen, got %v", err)
		}
	})

	t.Run("Reset forces closed state", func(t *testing.T) {
		cb := New[int]("test", WithFailureThreshold(1), WithTimeout(time.Hour))

		// Trigger open
		cb.Execute(context.Background(), func() (int, error) {
			return 0, errors.New("fail")
		})

		cb.Reset()

		if cb.State() != StateClosed {
			t.Errorf("expected closed state after reset, got %v", cb.State())
		}
	})
}

func TestCircuitBreakerMetrics(t *testing.T) {
	cb := New[int]("test", WithFailureThreshold(3))

	// Some failures
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func() (int, error) {
			return 0, errors.New("fail")
		})
	}

	metrics := cb.Metrics()
	if metrics.Failures != 2 {
		t.Errorf("expected 2 failures, got %d", metrics.Failures)
	}
	if metrics.State != StateClosed {
		t.Errorf("expected closed state, got %v", metrics.State)
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State(%d).String() = %s, want %s", tt.state, got, tt.expected)
		}
	}
}
