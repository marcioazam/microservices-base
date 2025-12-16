package domain

import (
	"context"
	"time"
)

// CircuitState represents the circuit breaker state.
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig defines circuit breaker behavior.
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold" yaml:"failureThreshold"`
	SuccessThreshold int           `json:"success_threshold" yaml:"successThreshold"`
	Timeout          time.Duration `json:"timeout" yaml:"timeout"`
	ProbeCount       int           `json:"probe_count" yaml:"probeCount"`
}

// CircuitBreakerState represents persistent circuit state.
type CircuitBreakerState struct {
	ServiceName     string       `json:"service_name"`
	State           CircuitState `json:"state"`
	FailureCount    int          `json:"failure_count"`
	SuccessCount    int          `json:"success_count"`
	LastFailureTime *time.Time   `json:"last_failure_time,omitempty"`
	LastStateChange time.Time    `json:"last_state_change"`
	Version         int64        `json:"version"`
}

// CircuitBreaker manages state transitions for a protected service.
type CircuitBreaker interface {
	// Execute runs the operation with circuit breaker protection.
	Execute(ctx context.Context, operation func() error) error

	// GetState returns current circuit state.
	GetState() CircuitState

	// GetFullState returns the complete circuit breaker state.
	GetFullState() CircuitBreakerState

	// RecordSuccess records a successful operation.
	RecordSuccess()

	// RecordFailure records a failed operation.
	RecordFailure()

	// Reset forces circuit to closed state.
	Reset()
}

// CircuitStateChangeEvent represents a circuit state change event.
type CircuitStateChangeEvent struct {
	ServiceName   string       `json:"service_name"`
	PreviousState CircuitState `json:"previous_state"`
	NewState      CircuitState `json:"new_state"`
	CorrelationID string       `json:"correlation_id"`
	Timestamp     time.Time    `json:"timestamp"`
	FailureCount  int          `json:"failure_count"`
	SuccessCount  int          `json:"success_count"`
}
