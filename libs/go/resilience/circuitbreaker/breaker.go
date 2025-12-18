// Package circuitbreaker implements the circuit breaker pattern.
package circuitbreaker

import (
	"context"
	"sync"
	"time"

	liberror "github.com/auth-platform/libs/go/error"
	"github.com/auth-platform/libs/go/resilience"
)

// Breaker implements the CircuitBreaker interface.
type Breaker struct {
	mu            sync.RWMutex
	config        resilience.CircuitBreakerConfig
	serviceName   string
	state         resilience.CircuitState
	failureCount  int
	successCount  int
	lastFailure   *time.Time
	lastChange    time.Time
	version       int64
	openedAt      time.Time
	eventEmitter  resilience.EventEmitter
	correlationFn func() string
}

// Config holds circuit breaker creation options.
type Config struct {
	ServiceName   string
	Config        resilience.CircuitBreakerConfig
	EventEmitter  resilience.EventEmitter
	CorrelationFn func() string
}

// New creates a new circuit breaker.
func New(cfg Config) *Breaker {
	return &Breaker{
		config:        cfg.Config,
		serviceName:   cfg.ServiceName,
		state:         resilience.StateClosed,
		lastChange:    resilience.NowUTC(),
		eventEmitter:  cfg.EventEmitter,
		correlationFn: resilience.EnsureCorrelationFunc(cfg.CorrelationFn),
	}
}

// Execute runs the operation with circuit breaker protection.
func (b *Breaker) Execute(ctx context.Context, operation func() error) error {
	if !b.allowRequest() {
		return liberror.NewCircuitOpenError(b.serviceName)
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
	case resilience.StateClosed:
		return true

	case resilience.StateOpen:
		if time.Since(b.openedAt) >= b.config.Timeout {
			b.transitionTo(resilience.StateHalfOpen)
			return true
		}
		return false

	case resilience.StateHalfOpen:
		return true

	default:
		return false
	}
}

// GetState returns current circuit state.
func (b *Breaker) GetState() resilience.CircuitState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// GetFullState returns the complete circuit breaker state.
func (b *Breaker) GetFullState() resilience.CircuitBreakerState {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return resilience.CircuitBreakerState{
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
	case resilience.StateClosed:
		b.failureCount = 0

	case resilience.StateHalfOpen:
		b.successCount++
		if b.successCount >= b.config.SuccessThreshold {
			b.transitionTo(resilience.StateClosed)
		}
	}
}

// RecordFailure records a failed operation.
func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := resilience.NowUTC()
	b.lastFailure = &now

	switch b.state {
	case resilience.StateClosed:
		b.failureCount++
		if b.failureCount >= b.config.FailureThreshold {
			b.transitionTo(resilience.StateOpen)
		}

	case resilience.StateHalfOpen:
		b.transitionTo(resilience.StateOpen)
	}
}

// Reset forces circuit to closed state.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.transitionTo(resilience.StateClosed)
	b.failureCount = 0
	b.successCount = 0
}

// transitionTo changes the circuit state. Must be called with lock held.
func (b *Breaker) transitionTo(newState resilience.CircuitState) {
	if b.state == newState {
		return
	}

	prevState := b.state
	b.state = newState
	b.lastChange = resilience.NowUTC()
	b.version++

	if newState == resilience.StateOpen {
		b.openedAt = resilience.NowUTC()
	}

	if newState == resilience.StateClosed {
		b.failureCount = 0
		b.successCount = 0
	}

	if newState == resilience.StateHalfOpen {
		b.successCount = 0
	}

	b.emitStateChangeEvent(prevState, newState)
}

// emitStateChangeEvent emits a state change event.
func (b *Breaker) emitStateChangeEvent(prevState, newState resilience.CircuitState) {
	event := resilience.Event{
		ID:            resilience.GenerateEventID(),
		Type:          resilience.EventCircuitStateChange,
		ServiceName:   b.serviceName,
		Timestamp:     resilience.NowUTC(),
		CorrelationID: b.correlationFn(),
		Metadata: map[string]any{
			"previous_state": prevState.String(),
			"new_state":      newState.String(),
			"failure_count":  b.failureCount,
			"success_count":  b.successCount,
		},
	}

	resilience.EmitEvent(b.eventEmitter, event)
}
