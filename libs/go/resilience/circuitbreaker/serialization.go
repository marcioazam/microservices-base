package circuitbreaker

import (
	"encoding/json"
	"fmt"

	"github.com/auth-platform/libs/go/resilience"
)

// circuitStateJSON is the JSON representation of CircuitState.
type circuitStateJSON struct {
	ServiceName     string  `json:"service_name"`
	State           string  `json:"state"`
	FailureCount    int     `json:"failure_count"`
	SuccessCount    int     `json:"success_count"`
	LastFailureTime *string `json:"last_failure_time,omitempty"`
	LastStateChange string  `json:"last_state_change"`
	Version         int64   `json:"version"`
}

// MarshalState serializes a CircuitBreakerState to JSON.
func MarshalState(s resilience.CircuitBreakerState) ([]byte, error) {
	js := circuitStateJSON{
		ServiceName:     s.ServiceName,
		State:           s.State.String(),
		FailureCount:    s.FailureCount,
		SuccessCount:    s.SuccessCount,
		LastStateChange: resilience.MarshalTime(s.LastStateChange),
		Version:         s.Version,
	}

	if s.LastFailureTime != nil {
		t := resilience.MarshalTime(*s.LastFailureTime)
		js.LastFailureTime = &t
	}

	return json.Marshal(js)
}

// UnmarshalState deserializes a CircuitBreakerState from JSON.
func UnmarshalState(data []byte) (resilience.CircuitBreakerState, error) {
	var js circuitStateJSON
	if err := json.Unmarshal(data, &js); err != nil {
		return resilience.CircuitBreakerState{}, fmt.Errorf("unmarshal circuit state: %w", err)
	}

	state, err := ParseCircuitState(js.State)
	if err != nil {
		return resilience.CircuitBreakerState{}, err
	}

	lastChange, err := resilience.UnmarshalTime(js.LastStateChange)
	if err != nil {
		return resilience.CircuitBreakerState{}, fmt.Errorf("parse last_state_change: %w", err)
	}

	result := resilience.CircuitBreakerState{
		ServiceName:     js.ServiceName,
		State:           state,
		FailureCount:    js.FailureCount,
		SuccessCount:    js.SuccessCount,
		LastStateChange: lastChange,
		Version:         js.Version,
	}

	if js.LastFailureTime != nil {
		t, err := resilience.UnmarshalTime(*js.LastFailureTime)
		if err != nil {
			return resilience.CircuitBreakerState{}, fmt.Errorf("parse last_failure_time: %w", err)
		}
		result.LastFailureTime = &t
	}

	return result, nil
}

// ParseCircuitState parses a string into CircuitState.
func ParseCircuitState(s string) (resilience.CircuitState, error) {
	switch s {
	case "CLOSED":
		return resilience.StateClosed, nil
	case "OPEN":
		return resilience.StateOpen, nil
	case "HALF_OPEN":
		return resilience.StateHalfOpen, nil
	default:
		return 0, fmt.Errorf("unknown circuit state: %s", s)
	}
}

// StateStore defines the interface for persisting circuit breaker state.
type StateStore interface {
	// Save persists the circuit breaker state.
	Save(state resilience.CircuitBreakerState) error

	// Load retrieves the circuit breaker state.
	Load(serviceName string) (resilience.CircuitBreakerState, error)

	// Delete removes the circuit breaker state.
	Delete(serviceName string) error
}
