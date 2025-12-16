package retry

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/testutil"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-microservice, Property 4: Retry Delay with Exponential Backoff and Jitter**
// **Validates: Requirements 2.2, 2.3**
func TestProperty_RetryDelayWithExponentialBackoffAndJitter(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("delay_within_jitter_bounds", prop.ForAll(
		func(attempt int, baseDelayMs int, maxDelayMs int, multiplier float64, jitterPercent float64) bool {
			if baseDelayMs >= maxDelayMs {
				return true // Skip invalid configs
			}

			config := domain.RetryConfig{
				MaxAttempts:   10,
				BaseDelay:     time.Duration(baseDelayMs) * time.Millisecond,
				MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
				Multiplier:    multiplier,
				JitterPercent: jitterPercent,
			}

			handler := New(Config{
				ServiceName: "test-service",
				Config:      config,
			})

			// Calculate expected base delay
			expectedBase := float64(config.BaseDelay) * math.Pow(multiplier, float64(attempt))
			if expectedBase > float64(config.MaxDelay) {
				expectedBase = float64(config.MaxDelay)
			}

			// Calculate bounds
			minDelay := expectedBase * (1 - jitterPercent)
			maxDelay := expectedBase * (1 + jitterPercent)

			// Test multiple times due to randomness
			for i := 0; i < 10; i++ {
				delay := handler.CalculateDelay(attempt)
				delayFloat := float64(delay)

				// Allow small epsilon for floating point
				epsilon := float64(time.Microsecond)
				if delayFloat < minDelay-epsilon || delayFloat > maxDelay+epsilon {
					return false
				}
			}

			return true
		},
		gen.IntRange(0, 5),
		gen.IntRange(10, 500),
		gen.IntRange(1000, 10000),
		gen.Float64Range(1.0, 3.0),
		gen.Float64Range(0.0, 0.5),
	))

	props.Property("delay_capped_at_max", prop.ForAll(
		func(attempt int, maxDelayMs int) bool {
			config := domain.RetryConfig{
				MaxAttempts:   10,
				BaseDelay:     100 * time.Millisecond,
				MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
				Multiplier:    10.0, // High multiplier to exceed max quickly
				JitterPercent: 0.0,  // No jitter for deterministic test
			}

			handler := New(Config{
				ServiceName: "test-service",
				Config:      config,
			})

			delay := handler.CalculateDelay(attempt)

			// Delay should never exceed max
			return delay <= config.MaxDelay
		},
		gen.IntRange(5, 20),
		gen.IntRange(500, 5000),
	))

	props.TestingRun(t)
}

// **Feature: resilience-microservice, Property 5: Retry Exhaustion Returns Final Error**
// **Validates: Requirements 2.4**
func TestProperty_RetryExhaustionReturnsFinalError(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("exhaustion_includes_attempt_count", prop.ForAll(
		func(maxAttempts int) bool {
			config := domain.RetryConfig{
				MaxAttempts:   maxAttempts,
				BaseDelay:     time.Millisecond,
				MaxDelay:      10 * time.Millisecond,
				Multiplier:    1.0,
				JitterPercent: 0.0,
			}

			handler := New(Config{
				ServiceName: "test-service",
				Config:      config,
			})

			attemptCount := 0
			testErr := errors.New("test error")

			err := handler.Execute(context.Background(), func() error {
				attemptCount++
				return testErr
			})

			// Should have made exactly maxAttempts
			if attemptCount != maxAttempts {
				return false
			}

			// Should return retry exhausted error
			var resErr *domain.ResilienceError
			if !errors.As(err, &resErr) {
				return false
			}

			if resErr.Code != domain.ErrRetryExhausted {
				return false
			}

			// Should include attempt count in metadata
			attempts, ok := resErr.Metadata["attempts"].(int)
			if !ok || attempts != maxAttempts {
				return false
			}

			return true
		},
		gen.IntRange(1, 5),
	))

	props.Property("exhaustion_wraps_final_error", prop.ForAll(
		func(maxAttempts int) bool {
			config := domain.RetryConfig{
				MaxAttempts:   maxAttempts,
				BaseDelay:     time.Millisecond,
				MaxDelay:      10 * time.Millisecond,
				Multiplier:    1.0,
				JitterPercent: 0.0,
			}

			handler := New(Config{
				ServiceName: "test-service",
				Config:      config,
			})

			finalErr := errors.New("final error")

			err := handler.Execute(context.Background(), func() error {
				return finalErr
			})

			// Should wrap the final error
			return errors.Is(err, finalErr)
		},
		gen.IntRange(1, 5),
	))

	props.TestingRun(t)
}

// **Feature: resilience-microservice, Property 6: Open Circuit Blocks Retry Attempts**
// **Validates: Requirements 2.5**
func TestProperty_OpenCircuitBlocksRetryAttempts(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("open_circuit_returns_immediately", prop.ForAll(
		func(maxAttempts int) bool {
			config := domain.RetryConfig{
				MaxAttempts:   maxAttempts,
				BaseDelay:     time.Second, // Long delay
				MaxDelay:      10 * time.Second,
				Multiplier:    2.0,
				JitterPercent: 0.0,
			}

			handler := New(Config{
				ServiceName: "test-service",
				Config:      config,
			})

			// Create a mock circuit breaker that's always open
			cb := &mockCircuitBreaker{state: domain.StateOpen}

			attemptCount := 0
			start := time.Now()

			err := handler.ExecuteWithCircuitBreaker(context.Background(), cb, func() error {
				attemptCount++
				return errors.New("should not be called")
			})

			elapsed := time.Since(start)

			// Should return immediately (no retries)
			if elapsed > 100*time.Millisecond {
				return false
			}

			// Should not have called operation
			if attemptCount != 0 {
				return false
			}

			// Should return circuit open error
			var resErr *domain.ResilienceError
			if !errors.As(err, &resErr) {
				return false
			}

			return resErr.Code == domain.ErrCircuitOpen
		},
		gen.IntRange(1, 10),
	))

	props.TestingRun(t)
}

// mockCircuitBreaker is a test implementation of CircuitBreaker.
type mockCircuitBreaker struct {
	state        domain.CircuitState
	successCount int
	failureCount int
}

func (m *mockCircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	return operation()
}

func (m *mockCircuitBreaker) GetState() domain.CircuitState {
	return m.state
}

func (m *mockCircuitBreaker) GetFullState() domain.CircuitBreakerState {
	return domain.CircuitBreakerState{State: m.state}
}

func (m *mockCircuitBreaker) RecordSuccess() {
	m.successCount++
}

func (m *mockCircuitBreaker) RecordFailure() {
	m.failureCount++
}

func (m *mockCircuitBreaker) Reset() {
	m.state = domain.StateClosed
	m.successCount = 0
	m.failureCount = 0
}
