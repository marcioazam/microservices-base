package circuitbreaker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
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

// MarshalJSON implements json.Marshaler for CircuitBreakerState.
func MarshalState(s domain.CircuitBreakerState) ([]byte, error) {
	js := circuitStateJSON{
		ServiceName:     s.ServiceName,
		State:           s.State.String(),
		FailureCount:    s.FailureCount,
		SuccessCount:    s.SuccessCount,
		LastStateChange: s.LastStateChange.Format(time.RFC3339Nano),
		Version:         s.Version,
	}

	if s.LastFailureTime != nil {
		t := s.LastFailureTime.Format(time.RFC3339Nano)
		js.LastFailureTime = &t
	}

	return json.Marshal(js)
}

// UnmarshalState deserializes a CircuitBreakerState from JSON.
func UnmarshalState(data []byte) (domain.CircuitBreakerState, error) {
	var js circuitStateJSON
	if err := json.Unmarshal(data, &js); err != nil {
		return domain.CircuitBreakerState{}, fmt.Errorf("unmarshal circuit state: %w", err)
	}

	state, err := parseCircuitState(js.State)
	if err != nil {
		return domain.CircuitBreakerState{}, err
	}

	lastChange, err := time.Parse(time.RFC3339Nano, js.LastStateChange)
	if err != nil {
		return domain.CircuitBreakerState{}, fmt.Errorf("parse last_state_change: %w", err)
	}

	result := domain.CircuitBreakerState{
		ServiceName:     js.ServiceName,
		State:           state,
		FailureCount:    js.FailureCount,
		SuccessCount:    js.SuccessCount,
		LastStateChange: lastChange,
		Version:         js.Version,
	}

	if js.LastFailureTime != nil {
		t, err := time.Parse(time.RFC3339Nano, *js.LastFailureTime)
		if err != nil {
			return domain.CircuitBreakerState{}, fmt.Errorf("parse last_failure_time: %w", err)
		}
		result.LastFailureTime = &t
	}

	return result, nil
}

// parseCircuitState parses a string into CircuitState.
func parseCircuitState(s string) (domain.CircuitState, error) {
	switch s {
	case "CLOSED":
		return domain.StateClosed, nil
	case "OPEN":
		return domain.StateOpen, nil
	case "HALF_OPEN":
		return domain.StateHalfOpen, nil
	default:
		return 0, fmt.Errorf("unknown circuit state: %s", s)
	}
}

// StateStore defines the interface for persisting circuit breaker state.
type StateStore interface {
	// Save persists the circuit breaker state.
	Save(state domain.CircuitBreakerState) error

	// Load retrieves the circuit breaker state.
	Load(serviceName string) (domain.CircuitBreakerState, error)

	// Delete removes the circuit breaker state.
	Delete(serviceName string) error
}
