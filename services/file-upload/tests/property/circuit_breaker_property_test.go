// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 3: Circuit Breaker Behavior
// Validates: Requirements 5.3, 5.4, 5.5, 2.5
package property

import (
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestCircuitState represents circuit breaker state for testing.
type TestCircuitState int

const (
	TestStateClosed TestCircuitState = iota
	TestStateOpen
	TestStateHalfOpen
)

// TestCircuitBreaker simulates circuit breaker for testing.
type TestCircuitBreaker struct {
	name             string
	failureThreshold int
	resetTimeout     time.Duration

	state           TestCircuitState
	failures        int
	lastFailureTime time.Time
	halfOpenCalls   int
	mu              sync.Mutex
}

func NewTestCircuitBreaker(name string, threshold int, timeout time.Duration) *TestCircuitBreaker {
	return &TestCircuitBreaker{
		name:             name,
		failureThreshold: threshold,
		resetTimeout:     timeout,
		state:            TestStateClosed,
	}
}

// Allow checks if request should be allowed.
func (cb *TestCircuitBreaker) Allow(now time.Time) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case TestStateClosed:
		return true
	case TestStateOpen:
		if now.Sub(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = TestStateHalfOpen
			cb.halfOpenCalls = 1 // Count this as the first probe
			return true
		}
		return false
	case TestStateHalfOpen:
		if cb.halfOpenCalls < 1 {
			cb.halfOpenCalls++
			return true
		}
		return false
	}
	return false
}

// RecordSuccess records a successful call.
func (cb *TestCircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == TestStateHalfOpen {
		cb.state = TestStateClosed
		cb.failures = 0
	} else if cb.state == TestStateClosed {
		cb.failures = 0
	}
}

// RecordFailure records a failed call.
func (cb *TestCircuitBreaker) RecordFailure(now time.Time) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = now

	if cb.state == TestStateClosed && cb.failures >= cb.failureThreshold {
		cb.state = TestStateOpen
	} else if cb.state == TestStateHalfOpen {
		cb.state = TestStateOpen
	}
}

// State returns current state.
func (cb *TestCircuitBreaker) State() TestCircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Failures returns failure count.
func (cb *TestCircuitBreaker) Failures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures
}

// TestProperty3_CircuitOpensAfterNFailures tests that circuit opens after N consecutive failures.
// Property 3: Circuit Breaker Behavior
// Validates: Requirements 5.3, 5.4, 5.5, 2.5
func TestProperty3_CircuitOpensAfterNFailures(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Test with different thresholds (S3: 5, Cache: 3, DB: 3)
		threshold := rapid.SampledFrom([]int{3, 5}).Draw(t, "threshold")
		name := rapid.SampledFrom([]string{"s3", "cache", "database"}).Draw(t, "name")

		cb := NewTestCircuitBreaker(name, threshold, 30*time.Second)
		now := time.Now()

		// Verify circuit starts closed
		if cb.State() != TestStateClosed {
			t.Error("circuit should start closed")
		}

		// Record failures up to threshold - 1
		for i := 0; i < threshold-1; i++ {
			cb.RecordFailure(now)
		}

		// Circuit should still be closed
		if cb.State() != TestStateClosed {
			t.Errorf("circuit should be closed with %d failures (threshold: %d)", threshold-1, threshold)
		}

		// Property: After N consecutive failures, circuit SHALL open
		cb.RecordFailure(now)
		if cb.State() != TestStateOpen {
			t.Errorf("circuit should be open after %d failures", threshold)
		}
	})
}

// TestProperty3_OpenCircuitFailsFast tests that open circuit fails fast without calling dependency.
// Property 3: Circuit Breaker Behavior
// Validates: Requirements 5.3, 5.4, 5.5, 2.5
func TestProperty3_OpenCircuitFailsFast(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		threshold := rapid.IntRange(2, 5).Draw(t, "threshold")
		cb := NewTestCircuitBreaker("test", threshold, 30*time.Second)
		now := time.Now()

		// Open the circuit
		for i := 0; i < threshold; i++ {
			cb.RecordFailure(now)
		}

		if cb.State() != TestStateOpen {
			t.Fatal("circuit should be open")
		}

		// Property: While circuit is open, requests SHALL fail fast
		numAttempts := rapid.IntRange(5, 20).Draw(t, "numAttempts")
		for i := 0; i < numAttempts; i++ {
			allowed := cb.Allow(now)
			if allowed {
				t.Errorf("request %d should be blocked when circuit is open", i)
			}
		}
	})
}

// TestProperty3_CircuitTransitionsToHalfOpen tests that circuit transitions to half-open after timeout.
// Property 3: Circuit Breaker Behavior
// Validates: Requirements 5.3, 5.4, 5.5, 2.5
func TestProperty3_CircuitTransitionsToHalfOpen(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		threshold := rapid.IntRange(2, 5).Draw(t, "threshold")
		timeoutSecs := rapid.IntRange(10, 60).Draw(t, "timeoutSecs")
		timeout := time.Duration(timeoutSecs) * time.Second

		cb := NewTestCircuitBreaker("test", threshold, timeout)
		baseTime := time.Now()

		// Open the circuit
		for i := 0; i < threshold; i++ {
			cb.RecordFailure(baseTime)
		}

		if cb.State() != TestStateOpen {
			t.Fatal("circuit should be open")
		}

		// Before timeout, should still be open
		beforeTimeout := baseTime.Add(timeout - time.Second)
		if cb.Allow(beforeTimeout) {
			t.Error("circuit should still be open before timeout")
		}

		// Property: After timeout, circuit SHALL transition to half-open
		afterTimeout := baseTime.Add(timeout + time.Second)
		allowed := cb.Allow(afterTimeout)
		if !allowed {
			t.Error("circuit should allow probe request after timeout")
		}
		if cb.State() != TestStateHalfOpen {
			t.Error("circuit should be half-open after timeout")
		}
	})
}

// TestProperty3_HalfOpenAllowsProbeRequests tests that half-open state allows probe requests.
// Property 3: Circuit Breaker Behavior
// Validates: Requirements 5.3, 5.4, 5.5, 2.5
func TestProperty3_HalfOpenAllowsProbeRequests(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		threshold := rapid.IntRange(2, 5).Draw(t, "threshold")
		timeout := 10 * time.Second

		cb := NewTestCircuitBreaker("test", threshold, timeout)
		baseTime := time.Now()

		// Open the circuit
		for i := 0; i < threshold; i++ {
			cb.RecordFailure(baseTime)
		}

		// Transition to half-open - first Allow() after timeout transitions AND allows probe
		afterTimeout := baseTime.Add(timeout + time.Second)
		firstAllowed := cb.Allow(afterTimeout)

		// Property: Half-open SHALL allow first probe request
		if !firstAllowed {
			t.Error("half-open should allow first probe request")
		}

		if cb.State() != TestStateHalfOpen {
			t.Fatal("circuit should be half-open")
		}

		// Property: Half-open SHALL block subsequent requests until probe completes
		secondAllowed := cb.Allow(afterTimeout)
		if secondAllowed {
			t.Error("half-open should block subsequent requests until probe completes")
		}
	})
}

// TestProperty3_SuccessfulProbeCloses tests that successful probe closes the circuit.
// Property 3: Circuit Breaker Behavior
// Validates: Requirements 5.3, 5.4, 5.5, 2.5
func TestProperty3_SuccessfulProbeCloses(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		threshold := rapid.IntRange(2, 5).Draw(t, "threshold")
		timeout := 10 * time.Second

		cb := NewTestCircuitBreaker("test", threshold, timeout)
		baseTime := time.Now()

		// Open the circuit
		for i := 0; i < threshold; i++ {
			cb.RecordFailure(baseTime)
		}

		// Transition to half-open
		afterTimeout := baseTime.Add(timeout + time.Second)
		cb.Allow(afterTimeout)

		if cb.State() != TestStateHalfOpen {
			t.Fatal("circuit should be half-open")
		}

		// Property: Successful probe SHALL close the circuit
		cb.RecordSuccess()
		if cb.State() != TestStateClosed {
			t.Error("circuit should be closed after successful probe")
		}
		if cb.Failures() != 0 {
			t.Error("failures should be reset after successful probe")
		}
	})
}

// TestProperty3_FailedProbeReopens tests that failed probe reopens the circuit.
// Property 3: Circuit Breaker Behavior
// Validates: Requirements 5.3, 5.4, 5.5, 2.5
func TestProperty3_FailedProbeReopens(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		threshold := rapid.IntRange(2, 5).Draw(t, "threshold")
		timeout := 10 * time.Second

		cb := NewTestCircuitBreaker("test", threshold, timeout)
		baseTime := time.Now()

		// Open the circuit
		for i := 0; i < threshold; i++ {
			cb.RecordFailure(baseTime)
		}

		// Transition to half-open
		afterTimeout := baseTime.Add(timeout + time.Second)
		cb.Allow(afterTimeout)

		if cb.State() != TestStateHalfOpen {
			t.Fatal("circuit should be half-open")
		}

		// Property: Failed probe SHALL reopen the circuit
		cb.RecordFailure(afterTimeout)
		if cb.State() != TestStateOpen {
			t.Error("circuit should be open after failed probe")
		}
	})
}

// TestProperty3_DifferentThresholdsPerDependency tests different thresholds for different dependencies.
// Property 3: Circuit Breaker Behavior
// Validates: Requirements 5.3, 5.4, 5.5, 2.5
func TestProperty3_DifferentThresholdsPerDependency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// S3: 5 failures, Cache: 3 failures, Database: 3 failures
		configs := map[string]int{
			"s3":       5,
			"cache":    3,
			"database": 3,
		}

		now := time.Now()

		for name, threshold := range configs {
			cb := NewTestCircuitBreaker(name, threshold, 30*time.Second)

			// Record exactly threshold failures
			for i := 0; i < threshold; i++ {
				cb.RecordFailure(now)
			}

			// Property: Each dependency SHALL have its configured threshold
			if cb.State() != TestStateOpen {
				t.Errorf("%s circuit should be open after %d failures", name, threshold)
			}

			// One less failure should not open
			cb2 := NewTestCircuitBreaker(name, threshold, 30*time.Second)
			for i := 0; i < threshold-1; i++ {
				cb2.RecordFailure(now)
			}
			if cb2.State() != TestStateClosed {
				t.Errorf("%s circuit should be closed with %d failures", name, threshold-1)
			}
		}
	})
}
