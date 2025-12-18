package property

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	liberror "github.com/auth-platform/libs/go/error"
	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/libs/go/resilience/rand"
	"github.com/auth-platform/libs/go/resilience/retry"
	"pgregory.net/rapid"
)

// **Feature: resilience-service-state-of-art-2025, Property 4: Retry Delay with Exponential Backoff and Jitter**
// **Validates: Requirements 2.2, 2.3**
func TestProperty_RetryDelayWithExponentialBackoffAndJitter(t *testing.T) {
	t.Run("delay_within_jitter_bounds", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			attempt := rapid.IntRange(0, 5).Draw(t, "attempt")
			baseDelayMs := rapid.IntRange(10, 500).Draw(t, "baseDelayMs")
			maxDelayMs := rapid.IntRange(1000, 10000).Draw(t, "maxDelayMs")
			multiplier := rapid.Float64Range(1.0, 3.0).Draw(t, "multiplier")
			jitterPercent := rapid.Float64Range(0.0, 0.5).Draw(t, "jitterPercent")

			if baseDelayMs >= maxDelayMs {
				return // Skip invalid configs
			}

			config := resilience.RetryConfig{
				MaxAttempts:   10,
				BaseDelay:     time.Duration(baseDelayMs) * time.Millisecond,
				MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
				Multiplier:    multiplier,
				JitterPercent: jitterPercent,
			}

			handler := retry.New(retry.Config{
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
					t.Fatalf("delay %v outside bounds [%v, %v]", delay, time.Duration(minDelay), time.Duration(maxDelay))
				}
			}
		})
	})

	t.Run("delay_capped_at_max", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			attempt := rapid.IntRange(5, 20).Draw(t, "attempt")
			maxDelayMs := rapid.IntRange(500, 5000).Draw(t, "maxDelayMs")

			config := resilience.RetryConfig{
				MaxAttempts:   10,
				BaseDelay:     100 * time.Millisecond,
				MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
				Multiplier:    10.0, // High multiplier to exceed max quickly
				JitterPercent: 0.0,  // No jitter for deterministic test
			}

			handler := retry.New(retry.Config{
				ServiceName: "test-service",
				Config:      config,
			})

			delay := handler.CalculateDelay(attempt)

			// Delay should never exceed max
			if delay > config.MaxDelay {
				t.Fatalf("delay %v exceeds max %v", delay, config.MaxDelay)
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 5: Retry Exhaustion Returns Final Error**
// **Validates: Requirements 2.4**
func TestProperty_RetryExhaustionReturnsFinalError(t *testing.T) {
	t.Run("exhaustion_includes_attempt_count", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxAttempts := rapid.IntRange(1, 5).Draw(t, "maxAttempts")

			config := resilience.RetryConfig{
				MaxAttempts:   maxAttempts,
				BaseDelay:     time.Millisecond,
				MaxDelay:      10 * time.Millisecond,
				Multiplier:    1.0,
				JitterPercent: 0.0,
			}

			handler := retry.New(retry.Config{
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
				t.Fatalf("attempt count %d != %d", attemptCount, maxAttempts)
			}

			// Should return retry exhausted error
			var resErr *liberror.ResilienceError
			if !errors.As(err, &resErr) {
				t.Fatalf("expected ResilienceError, got %T", err)
			}

			if resErr.Code != liberror.ErrRetryExhausted {
				t.Fatalf("expected ErrRetryExhausted, got %s", resErr.Code)
			}

			// Should include attempt count in metadata
			attempts, ok := resErr.Metadata["attempts"].(int)
			if !ok || attempts != maxAttempts {
				t.Fatalf("expected attempts=%d in metadata, got %v", maxAttempts, resErr.Metadata["attempts"])
			}
		})
	})

	t.Run("exhaustion_wraps_final_error", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxAttempts := rapid.IntRange(1, 5).Draw(t, "maxAttempts")

			config := resilience.RetryConfig{
				MaxAttempts:   maxAttempts,
				BaseDelay:     time.Millisecond,
				MaxDelay:      10 * time.Millisecond,
				Multiplier:    1.0,
				JitterPercent: 0.0,
			}

			handler := retry.New(retry.Config{
				ServiceName: "test-service",
				Config:      config,
			})

			finalErr := errors.New("final error")

			err := handler.Execute(context.Background(), func() error {
				return finalErr
			})

			// Should wrap the final error
			if !errors.Is(err, finalErr) {
				t.Fatalf("error should wrap final error")
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 6: Open Circuit Blocks Retry Attempts**
// **Validates: Requirements 2.5**
func TestProperty_OpenCircuitBlocksRetryAttempts(t *testing.T) {
	t.Run("open_circuit_returns_immediately", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			maxAttempts := rapid.IntRange(1, 10).Draw(t, "maxAttempts")

			config := resilience.RetryConfig{
				MaxAttempts:   maxAttempts,
				BaseDelay:     time.Second, // Long delay
				MaxDelay:      10 * time.Second,
				Multiplier:    2.0,
				JitterPercent: 0.0,
			}

			handler := retry.New(retry.Config{
				ServiceName: "test-service",
				Config:      config,
			})

			// Create a mock circuit breaker that's always open
			cb := &mockCircuitBreaker{state: resilience.StateOpen}

			attemptCount := 0
			start := time.Now()

			err := handler.ExecuteWithCircuitBreaker(context.Background(), cb, func() error {
				attemptCount++
				return errors.New("should not be called")
			})

			elapsed := time.Since(start)

			// Should return immediately (no retries)
			if elapsed > 100*time.Millisecond {
				t.Fatalf("took too long: %v", elapsed)
			}

			// Should not have called operation
			if attemptCount != 0 {
				t.Fatalf("operation was called %d times", attemptCount)
			}

			// Should return circuit open error
			var resErr *liberror.ResilienceError
			if !errors.As(err, &resErr) {
				t.Fatalf("expected ResilienceError, got %T", err)
			}

			if resErr.Code != liberror.ErrCircuitOpen {
				t.Fatalf("expected ErrCircuitOpen, got %s", resErr.Code)
			}
		})
	})
}

// mockCircuitBreaker is a test implementation of CircuitBreaker.
type mockCircuitBreaker struct {
	state        resilience.CircuitState
	successCount int
	failureCount int
}

func (m *mockCircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	return operation()
}

func (m *mockCircuitBreaker) GetState() resilience.CircuitState {
	return m.state
}

func (m *mockCircuitBreaker) GetFullState() resilience.CircuitBreakerState {
	return resilience.CircuitBreakerState{State: m.state}
}

func (m *mockCircuitBreaker) RecordSuccess() {
	m.successCount++
}

func (m *mockCircuitBreaker) RecordFailure() {
	m.failureCount++
}

func (m *mockCircuitBreaker) Reset() {
	m.state = resilience.StateClosed
	m.successCount = 0
	m.failureCount = 0
}

// **Feature: resilience-service-state-of-art-2025, Property 9: Retry Delay Bounds**
// **Validates: Requirements 8.1**
func TestProperty_RetryDelayBounds(t *testing.T) {
	t.Run("delay_is_non_negative_and_bounded", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			attempt := rapid.IntRange(0, 10).Draw(t, "attempt")
			baseDelayMs := rapid.IntRange(10, 500).Draw(t, "baseDelayMs")
			maxDelayMs := rapid.IntRange(1000, 10000).Draw(t, "maxDelayMs")
			multiplier := rapid.Float64Range(1.0, 3.0).Draw(t, "multiplier")
			jitterPercent := rapid.Float64Range(0.0, 0.5).Draw(t, "jitterPercent")

			if baseDelayMs >= maxDelayMs || baseDelayMs <= 0 {
				return // Skip invalid configs
			}

			config := resilience.RetryConfig{
				MaxAttempts:   10,
				BaseDelay:     time.Duration(baseDelayMs) * time.Millisecond,
				MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
				Multiplier:    multiplier,
				JitterPercent: jitterPercent,
			}

			handler := retry.New(retry.Config{
				ServiceName: "test-service",
				Config:      config,
			})

			// Test multiple times due to randomness
			for i := 0; i < 20; i++ {
				delay := handler.CalculateDelay(attempt)

				// Property 9: Delay must be non-negative
				if delay < 0 {
					t.Fatalf("delay is negative: %v", delay)
				}

				// Property 9: Delay must not exceed MaxDelay
				if delay > config.MaxDelay {
					t.Fatalf("delay %v exceeds MaxDelay %v", delay, config.MaxDelay)
				}
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 10: Deterministic Retry with Fixed Seed**
// **Validates: Requirements 8.2**
func TestProperty_DeterministicRetryWithFixedSeed(t *testing.T) {
	t.Run("same_seed_produces_same_delays", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			seed := rapid.Int64Range(1, 1000000).Draw(t, "seed")
			attempt := rapid.IntRange(0, 10).Draw(t, "attempt")
			baseDelayMs := rapid.IntRange(10, 500).Draw(t, "baseDelayMs")
			maxDelayMs := rapid.IntRange(1000, 10000).Draw(t, "maxDelayMs")

			if baseDelayMs >= maxDelayMs || baseDelayMs <= 0 {
				return // Skip invalid configs
			}

			config := resilience.RetryConfig{
				MaxAttempts:   10,
				BaseDelay:     time.Duration(baseDelayMs) * time.Millisecond,
				MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
				Multiplier:    2.0,
				JitterPercent: 0.3,
			}

			// Create two handlers with the same deterministic seed
			handler1 := retry.New(retry.Config{
				ServiceName: "test-service",
				Config:      config,
				RandSource:  rand.NewDeterministicRandSource(seed),
			})

			handler2 := retry.New(retry.Config{
				ServiceName: "test-service",
				Config:      config,
				RandSource:  rand.NewDeterministicRandSource(seed),
			})

			// Both handlers should produce the same delay for the same attempt
			delay1 := handler1.CalculateDelay(attempt)
			delay2 := handler2.CalculateDelay(attempt)

			if delay1 != delay2 {
				t.Fatalf("delays differ: %v vs %v for seed %d, attempt %d", delay1, delay2, seed, attempt)
			}
		})
	})
}
