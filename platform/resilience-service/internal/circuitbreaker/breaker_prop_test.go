package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
)

// **Feature: resilience-service-modernization, Property 3: Centralized Event Emission (Circuit Breaker)**
// **Validates: Requirements 2.1, 2.2, 2.3, 4.1**
func TestCentralizedEventEmission(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Circuit breaker events use UUID v7 IDs", prop.ForAll(
		func(serviceName string, failureThreshold int) bool {
			if failureThreshold < 1 {
				failureThreshold = 1
			}
			if failureThreshold > 10 {
				failureThreshold = 10
			}

			emitter := NewMockEventEmitter()
			builder := domain.NewEventBuilder(emitter, serviceName, nil)

			breaker := New(Config{
				ServiceName:  serviceName,
				EventBuilder: builder,
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: failureThreshold,
					SuccessThreshold: 1,
					Timeout:          time.Second,
				},
			})

			// Trigger state change by recording failures
			for i := 0; i < failureThreshold; i++ {
				breaker.RecordFailure()
			}

			events := emitter.GetStateChangeEvents()
			if len(events) == 0 {
				return true // No events emitted is valid if threshold not reached
			}

			// Verify all events have valid UUID v7 IDs
			for _, event := range events {
				if !domain.IsValidUUIDv7(event.ID) {
					t.Logf("Event ID is not valid UUID v7: %s", event.ID)
					return false
				}
			}

			return true
		},
		gen.AlphaString(),
		gen.IntRange(1, 10),
	))

	properties.Property("Circuit breaker events have consistent field population", prop.ForAll(
		func(serviceName string) bool {
			emitter := NewMockEventEmitter()
			correlationID := "test-correlation-123"
			correlationFn := func() string { return correlationID }
			builder := domain.NewEventBuilder(emitter, serviceName, correlationFn)

			breaker := New(Config{
				ServiceName:  serviceName,
				EventBuilder: builder,
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: 1,
					SuccessThreshold: 1,
					Timeout:          time.Second,
				},
			})

			// Trigger state change
			breaker.RecordFailure()

			events := emitter.GetStateChangeEvents()
			if len(events) == 0 {
				t.Log("No events emitted")
				return false
			}

			event := events[0]

			// Verify service name
			if event.ServiceName != serviceName {
				t.Logf("ServiceName mismatch: expected %s, got %s", serviceName, event.ServiceName)
				return false
			}

			// Verify correlation ID
			if event.CorrelationID != correlationID {
				t.Logf("CorrelationID mismatch: expected %s, got %s", correlationID, event.CorrelationID)
				return false
			}

			// Verify event type
			if event.Type != domain.EventCircuitStateChange {
				t.Logf("Type mismatch: expected %s, got %s", domain.EventCircuitStateChange, event.Type)
				return false
			}

			return true
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Circuit opens after failure threshold", prop.ForAll(
		func(threshold int) bool {
			if threshold < 1 {
				threshold = 1
			}
			if threshold > 20 {
				threshold = 20
			}

			breaker := New(Config{
				ServiceName: "test",
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: threshold,
					SuccessThreshold: 1,
					Timeout:          time.Second,
				},
			})

			// Record failures up to threshold - 1
			for i := 0; i < threshold-1; i++ {
				breaker.RecordFailure()
				if breaker.GetState() != domain.StateClosed {
					return false
				}
			}

			// One more failure should open the circuit
			breaker.RecordFailure()
			return breaker.GetState() == domain.StateOpen
		},
		gen.IntRange(1, 20),
	))

	properties.Property("Circuit closes after success threshold in half-open", prop.ForAll(
		func(successThreshold int) bool {
			if successThreshold < 2 {
				successThreshold = 2
			}
			if successThreshold > 10 {
				successThreshold = 10
			}

			breaker := New(Config{
				ServiceName: "test",
				Config: domain.CircuitBreakerConfig{
					FailureThreshold: 1,
					SuccessThreshold: successThreshold,
					Timeout:          time.Millisecond,
				},
			})

			// Open the circuit
			breaker.RecordFailure()
			if breaker.GetState() != domain.StateOpen {
				return false
			}

			// Wait for timeout to allow half-open transition
			time.Sleep(2 * time.Millisecond)

			// allowRequest triggers half-open transition
			breaker.allowRequest()
			if breaker.GetState() != domain.StateHalfOpen {
				return false
			}

			// Record successes up to threshold - 1
			for i := 0; i < successThreshold-1; i++ {
				breaker.RecordSuccess()
				if breaker.GetState() != domain.StateHalfOpen {
					return false
				}
			}

			// One more success should close the circuit
			breaker.RecordSuccess()
			return breaker.GetState() == domain.StateClosed
		},
		gen.IntRange(2, 10),
	))

	properties.TestingRun(t)
}

func TestCircuitBreaker_Execute(t *testing.T) {
	breaker := New(Config{
		ServiceName: "test",
		Config: domain.CircuitBreakerConfig{
			FailureThreshold: 3,
			SuccessThreshold: 2,
			Timeout:          time.Second,
		},
	})

	// Successful execution
	err := breaker.Execute(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Failed execution
	testErr := errors.New("test error")
	err = breaker.Execute(context.Background(), func() error {
		return testErr
	})
	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}
}

func TestCircuitBreaker_NilEventBuilder(t *testing.T) {
	breaker := New(Config{
		ServiceName:  "test",
		EventBuilder: nil,
		Config: domain.CircuitBreakerConfig{
			FailureThreshold: 1,
			SuccessThreshold: 1,
			Timeout:          time.Second,
		},
	})

	// Should not panic with nil event builder
	breaker.RecordFailure()
	if breaker.GetState() != domain.StateOpen {
		t.Error("Circuit should be open")
	}
}
