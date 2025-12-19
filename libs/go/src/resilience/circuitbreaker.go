package resilience

import (
	"context"
	"sync"
	"time"

	"github.com/authcorp/libs/go/src/functional"
)

// State represents circuit breaker state.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	config        CircuitBreakerConfig
	state         State
	failures      int
	successes     int
	lastFailure   time.Time
	mu            sync.RWMutex
	correlationID string
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config CircuitBreakerConfig) (*CircuitBreaker, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}, nil
}

// Execute runs the operation through the circuit breaker.
func (cb *CircuitBreaker) Execute(ctx context.Context, op func(context.Context) error) error {
	if err := cb.canExecute(); err != nil {
		return err
	}

	err := op(ctx)
	cb.recordResult(err)
	return err
}

// ExecuteWithResult runs operation returning Result[T].
func ExecuteWithResult[T any](cb *CircuitBreaker, ctx context.Context, op func(context.Context) (T, error)) functional.Result[T] {
	if err := cb.canExecute(); err != nil {
		return functional.Err[T](err)
	}

	value, err := op(ctx)
	cb.recordResult(err)
	if err != nil {
		return functional.Err[T](err)
	}
	return functional.Ok(value)
}

func (cb *CircuitBreaker) canExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil
	case StateOpen:
		if time.Since(cb.lastFailure) > cb.config.Timeout {
			cb.transitionTo(StateHalfOpen)
			return nil
		}
		return NewCircuitOpenError(
			cb.config.Name,
			cb.correlationID,
			cb.lastFailure,
			cb.config.Timeout-time.Since(cb.lastFailure),
			float64(cb.failures)/float64(cb.config.FailureThreshold),
		)
	case StateHalfOpen:
		return nil
	}
	return nil
}

func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailure = time.Now()
	cb.successes = 0

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		cb.transitionTo(StateOpen)
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transitionTo(StateClosed)
		}
	}
}

func (cb *CircuitBreaker) transitionTo(newState State) {
	oldState := cb.state
	cb.state = newState

	if newState == StateClosed {
		cb.failures = 0
		cb.successes = 0
	}

	if cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(oldState.String(), newState.String())
	}
}

// State returns current circuit state.
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// SetCorrelationID sets correlation ID for error tracking.
func (cb *CircuitBreaker) SetCorrelationID(id string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.correlationID = id
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
}
