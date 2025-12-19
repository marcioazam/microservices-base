// Package circuitbreaker provides a generic circuit breaker implementation.
package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// State represents the circuit breaker state.
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

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// Config holds circuit breaker configuration.
type Config struct {
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
	HalfOpenMaxCalls int
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		HalfOpenMaxCalls: 3,
	}
}

// Option configures a circuit breaker.
type Option func(*Config)

// WithFailureThreshold sets the failure threshold.
func WithFailureThreshold(n int) Option {
	return func(c *Config) { c.FailureThreshold = n }
}

// WithSuccessThreshold sets the success threshold.
func WithSuccessThreshold(n int) Option {
	return func(c *Config) { c.SuccessThreshold = n }
}

// WithTimeout sets the timeout duration.
func WithTimeout(d time.Duration) Option {
	return func(c *Config) { c.Timeout = d }
}

// WithHalfOpenMaxCalls sets the max calls in half-open state.
func WithHalfOpenMaxCalls(n int) Option {
	return func(c *Config) { c.HalfOpenMaxCalls = n }
}

// CircuitBreaker is a generic circuit breaker.
type CircuitBreaker[T any] struct {
	name   string
	config Config
	mu     sync.RWMutex

	state           State
	failures        int
	successes       int
	halfOpenCalls   int
	lastFailureTime time.Time
}

// New creates a new circuit breaker.
func New[T any](name string, opts ...Option) *CircuitBreaker[T] {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(&config)
	}
	return &CircuitBreaker[T]{
		name:   name,
		config: config,
		state:  StateClosed,
	}
}

// Execute runs the operation with circuit breaker protection.
func (cb *CircuitBreaker[T]) Execute(ctx context.Context, op func() (T, error)) (T, error) {
	var zero T

	if err := cb.beforeCall(); err != nil {
		return zero, err
	}

	result, err := op()

	cb.afterCall(err == nil)

	return result, err
}

// State returns the current circuit state.
func (cb *CircuitBreaker[T]) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.currentState()
}

// Reset forces the circuit to closed state.
func (cb *CircuitBreaker[T]) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenCalls = 0
}

// Name returns the circuit breaker name.
func (cb *CircuitBreaker[T]) Name() string {
	return cb.name
}

// Metrics returns current metrics.
func (cb *CircuitBreaker[T]) Metrics() Metrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return Metrics{
		State:           cb.currentState(),
		Failures:        cb.failures,
		Successes:       cb.successes,
		LastFailureTime: cb.lastFailureTime,
	}
}

// Metrics holds circuit breaker metrics.
type Metrics struct {
	State           State
	Failures        int
	Successes       int
	LastFailureTime time.Time
}

func (cb *CircuitBreaker[T]) beforeCall() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.currentState()

	switch state {
	case StateOpen:
		return ErrCircuitOpen
	case StateHalfOpen:
		if cb.halfOpenCalls >= cb.config.HalfOpenMaxCalls {
			return ErrCircuitOpen
		}
		cb.halfOpenCalls++
	}

	return nil
}

func (cb *CircuitBreaker[T]) afterCall(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.currentState()

	if success {
		cb.onSuccess(state)
	} else {
		cb.onFailure(state)
	}
}

func (cb *CircuitBreaker[T]) onSuccess(state State) {
	switch state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
			cb.halfOpenCalls = 0
		}
	}
}

func (cb *CircuitBreaker[T]) onFailure(state State) {
	cb.lastFailureTime = time.Now()

	switch state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.successes = 0
		cb.halfOpenCalls = 0
	}
}

func (cb *CircuitBreaker[T]) currentState() State {
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.halfOpenCalls = 0
			cb.successes = 0
		}
	}
	return cb.state
}
