package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 1: Circuit Breaker State Transitions**
// **Validates: Requirements 1.1**
func TestCircuitBreakerStateTransitions(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("failures up to threshold transitions CLOSED to OPEN", prop.ForAll(
		func(threshold int) bool {
			if threshold < 1 {
				threshold = 1
			}
			if threshold > 20 {
				threshold = 20
			}

			cb := New(Config{
				ServiceName: "test-service",
				Config: resilience.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 2,
					Timeout:          time.Second,
				},
			})

			// Initial state should be CLOSED
			if cb.GetState() != resilience.StateClosed {
				return false
			}

			// Record failures up to threshold
			for i := 0; i < threshold; i++ {
				cb.RecordFailure()
			}

			// State should now be OPEN
			return cb.GetState() == resilience.StateOpen
		},
		gen.IntRange(1, 20),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 2: Circuit Breaker Half-Open Recovery**
// **Validates: Requirements 1.1**
func TestCircuitBreakerHalfOpenRecovery(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("after timeout, OPEN transitions to HALF_OPEN on request", prop.ForAll(
		func(timeoutMs int) bool {
			if timeoutMs < 1 {
				timeoutMs = 1
			}
			if timeoutMs > 100 {
				timeoutMs = 100
			}

			timeout := time.Duration(timeoutMs) * time.Millisecond

			cb := New(Config{
				ServiceName: "test-service",
				Config: resilience.CircuitBreakerConfig{
					FailureThreshold: 1,
					SuccessThreshold: 1,
					Timeout:          timeout,
				},
			})

			// Force to OPEN state
			cb.RecordFailure()
			if cb.GetState() != resilience.StateOpen {
				return false
			}

			// Wait for timeout
			time.Sleep(timeout + 10*time.Millisecond)

			// Execute should transition to HALF_OPEN
			_ = cb.Execute(context.Background(), func() error {
				return nil
			})

			// After success in HALF_OPEN, should be CLOSED
			return cb.GetState() == resilience.StateClosed
		},
		gen.IntRange(1, 100),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 11: Serialization Round-Trip**
// **Validates: Requirements 5.1, 5.3**
func TestCircuitBreakerStateRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("marshal then unmarshal preserves state", prop.ForAll(
		func(serviceName string, stateInt int, failureCount int, successCount int) bool {
			if serviceName == "" {
				serviceName = "test"
			}
			if len(serviceName) > 50 {
				serviceName = serviceName[:50]
			}

			state := resilience.CircuitState(stateInt % 3)
			now := time.Now().UTC().Truncate(time.Nanosecond)

			original := resilience.CircuitBreakerState{
				ServiceName:     serviceName,
				State:           state,
				FailureCount:    failureCount % 100,
				SuccessCount:    successCount % 100,
				LastStateChange: now,
				Version:         1,
			}

			data, err := MarshalState(original)
			if err != nil {
				return false
			}

			restored, err := UnmarshalState(data)
			if err != nil {
				return false
			}

			return original.ServiceName == restored.ServiceName &&
				original.State == restored.State &&
				original.FailureCount == restored.FailureCount &&
				original.SuccessCount == restored.SuccessCount &&
				original.Version == restored.Version
		},
		gen.AlphaString(),
		gen.IntRange(0, 2),
		gen.IntRange(0, 100),
		gen.IntRange(0, 100),
	))

	properties.TestingRun(t)
}

func TestCircuitBreakerExecuteRejectsWhenOpen(t *testing.T) {
	cb := New(Config{
		ServiceName: "test-service",
		Config: resilience.CircuitBreakerConfig{
			FailureThreshold: 1,
			SuccessThreshold: 1,
			Timeout:          time.Hour,
		},
	})

	// Force to OPEN state
	cb.RecordFailure()

	err := cb.Execute(context.Background(), func() error {
		return nil
	})

	if err == nil {
		t.Error("expected error when circuit is open")
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := New(Config{
		ServiceName: "test-service",
		Config: resilience.CircuitBreakerConfig{
			FailureThreshold: 1,
			SuccessThreshold: 1,
			Timeout:          time.Hour,
		},
	})

	// Force to OPEN state
	cb.RecordFailure()
	if cb.GetState() != resilience.StateOpen {
		t.Fatal("expected OPEN state")
	}

	// Reset
	cb.Reset()

	if cb.GetState() != resilience.StateClosed {
		t.Error("expected CLOSED state after reset")
	}

	state := cb.GetFullState()
	if state.FailureCount != 0 || state.SuccessCount != 0 {
		t.Error("expected counts to be reset")
	}
}

func TestParseCircuitState(t *testing.T) {
	tests := []struct {
		input    string
		expected resilience.CircuitState
		wantErr  bool
	}{
		{"CLOSED", resilience.StateClosed, false},
		{"OPEN", resilience.StateOpen, false},
		{"HALF_OPEN", resilience.StateHalfOpen, false},
		{"INVALID", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseCircuitState(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseCircuitState(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("ParseCircuitState(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestCircuitBreakerExecuteWithError(t *testing.T) {
	cb := New(Config{
		ServiceName: "test-service",
		Config: resilience.CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          time.Second,
		},
	})

	testErr := errors.New("test error")
	err := cb.Execute(context.Background(), func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("expected test error, got %v", err)
	}

	state := cb.GetFullState()
	if state.FailureCount != 1 {
		t.Errorf("expected failure count 1, got %d", state.FailureCount)
	}
}
