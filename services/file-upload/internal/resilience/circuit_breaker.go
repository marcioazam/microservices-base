// Package resilience provides resilience patterns for file upload service.
package resilience

import (
	"sync"
	"time"
)

// State represents circuit breaker state.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// String returns the string representation of the state.
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

// Config holds circuit breaker configuration.
type Config struct {
	Name             string
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenMaxCalls int
}

// DefaultConfigs returns default configurations for dependencies.
func DefaultConfigs() map[string]Config {
	return map[string]Config{
		"s3": {
			Name:             "s3",
			FailureThreshold: 5,
			ResetTimeout:     30 * time.Second,
			HalfOpenMaxCalls: 1,
		},
		"cache": {
			Name:             "cache",
			FailureThreshold: 3,
			ResetTimeout:     15 * time.Second,
			HalfOpenMaxCalls: 1,
		},
		"database": {
			Name:             "database",
			FailureThreshold: 3,
			ResetTimeout:     15 * time.Second,
			HalfOpenMaxCalls: 1,
		},
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	name             string
	failureThreshold int
	resetTimeout     time.Duration
	halfOpenMaxCalls int

	state           State
	failures        int
	successes       int
	lastFailureTime time.Time
	halfOpenCalls   int
	mu              sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(cfg Config) *CircuitBreaker {
	return &CircuitBreaker{
		name:             cfg.Name,
		failureThreshold: cfg.FailureThreshold,
		resetTimeout:     cfg.ResetTimeout,
		halfOpenMaxCalls: cfg.HalfOpenMaxCalls,
		state:            StateClosed,
	}
}

// Allow checks if a request should be allowed.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.halfOpenCalls = 0
			return true
		}
		return false
	case StateHalfOpen:
		if cb.halfOpenCalls < cb.halfOpenMaxCalls {
			cb.halfOpenCalls++
			return true
		}
		return false
	}
	return false
}

// RecordSuccess records a successful call.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.halfOpenMaxCalls {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
		}
	}
}

// RecordFailure records a failed call.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.failureThreshold {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.successes = 0
	}
}

// State returns the current state.
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Name returns the circuit breaker name.
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// Failures returns the current failure count.
func (cb *CircuitBreaker) Failures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenCalls = 0
}

// Metrics returns circuit breaker metrics.
type Metrics struct {
	Name     string
	State    string
	Failures int
}

// GetMetrics returns current metrics.
func (cb *CircuitBreaker) GetMetrics() Metrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return Metrics{
		Name:     cb.name,
		State:    cb.state.String(),
		Failures: cb.failures,
	}
}
