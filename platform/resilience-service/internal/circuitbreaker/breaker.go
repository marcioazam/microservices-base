// Package circuitbreaker implements the circuit breaker pattern.
package circuitbreaker

import (
	"context"
	"sync"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
)

// Breaker implements the CircuitBreaker interface.
type Breaker struct {
	mu           sync.RWMutex
	config       domain.CircuitBreakerConfig
	serviceName  string
	state        domain.CircuitState
	failureCount int
	successCount int
	lastFailure  *time.Time
	lastChange   time.Time
	version      int64
	openedAt     time.Time
	eventBuilder *domain.EventBuilder
}

// Config holds circuit breaker creation options.
type Config struct {
	ServiceName  string
	Config       domain.CircuitBreakerConfig
	EventBuilder *domain.EventBuilder
}

// New creates a new circuit breaker.
func New(cfg Config) *Breaker {
	return &Breaker{
		config:       cfg.Config,
		serviceName:  cfg.ServiceName,
		state:        domain.StateClosed,
		lastChange:   time.Now(),
		eventBuilder: cfg.EventBuilder,
	}
}

// Execute runs the operation with circuit breaker protection.
func (b *Breaker) Execute(ctx context.Context, operation func() error) error {
	if !b.allowRequest() {
		return domain.NewCircuitOpenError(b.serviceName)
	}

	err := operation()
	if err != nil {
		b.RecordFailure()
		return err
	}

	b.RecordSuccess()
	return nil
}

// allowRequest checks if a request should be allowed.
func (b *Breaker) allowRequest() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case domain.StateClosed:
		return true

	case domain.StateOpen:
		if time.Since(b.openedAt) >= b.config.Timeout {
			b.transitionTo(domain.StateHalfOpen)
			return true
		}
		return false

	case domain.StateHalfOpen:
		return true

	default:
		return false
	}
}

// GetState returns current circuit state.
func (b *Breaker) GetState() domain.CircuitState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// GetFullState returns the complete circuit breaker state.
func (b *Breaker) GetFullState() domain.CircuitBreakerState {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return domain.CircuitBreakerState{
		ServiceName:     b.serviceName,
		State:           b.state,
		FailureCount:    b.failureCount,
		SuccessCount:    b.successCount,
		LastFailureTime: b.lastFailure,
		LastStateChange: b.lastChange,
		Version:         b.version,
	}
}

// RecordSuccess records a successful operation.
func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case domain.StateClosed:
		b.failureCount = 0

	case domain.StateHalfOpen:
		b.successCount++
		if b.successCount >= b.config.SuccessThreshold {
			b.transitionTo(domain.StateClosed)
		}
	}
}

// RecordFailure records a failed operation.
func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	b.lastFailure = &now

	switch b.state {
	case domain.StateClosed:
		b.failureCount++
		if b.failureCount >= b.config.FailureThreshold {
			b.transitionTo(domain.StateOpen)
		}

	case domain.StateHalfOpen:
		b.transitionTo(domain.StateOpen)
	}
}

// Reset forces circuit to closed state.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.transitionTo(domain.StateClosed)
	b.failureCount = 0
	b.successCount = 0
}

// transitionTo changes the circuit state. Must be called with lock held.
func (b *Breaker) transitionTo(newState domain.CircuitState) {
	if b.state == newState {
		return
	}

	prevState := b.state
	b.state = newState
	b.lastChange = time.Now()
	b.version++

	if newState == domain.StateOpen {
		b.openedAt = time.Now()
	}

	if newState == domain.StateClosed {
		b.failureCount = 0
		b.successCount = 0
	}

	if newState == domain.StateHalfOpen {
		b.successCount = 0
	}

	b.emitStateChangeEvent(prevState, newState)
}

// emitStateChangeEvent emits a state change event using EventBuilder.
func (b *Breaker) emitStateChangeEvent(prevState, newState domain.CircuitState) {
	if b.eventBuilder == nil {
		return
	}

	b.eventBuilder.Emit(domain.EventCircuitStateChange, map[string]any{
		"previous_state": prevState.String(),
		"new_state":      newState.String(),
		"failure_count":  b.failureCount,
		"success_count":  b.successCount,
	})
}
