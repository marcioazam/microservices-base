// Package timeout implements timeout management with context cancellation.
package timeout

import (
	"context"
	"time"

	liberror "github.com/auth-platform/libs/go/error"
	"github.com/auth-platform/libs/go/resilience"
)

// Manager implements the TimeoutManager interface.
type Manager struct {
	config        resilience.TimeoutConfig
	serviceName   string
	eventEmitter  resilience.EventEmitter
	correlationFn func() string
}

// Config holds timeout manager creation options.
type Config struct {
	ServiceName   string
	Config        resilience.TimeoutConfig
	EventEmitter  resilience.EventEmitter
	CorrelationFn func() string
}

// New creates a new timeout manager.
func New(cfg Config) *Manager {
	return &Manager{
		config:        cfg.Config,
		serviceName:   cfg.ServiceName,
		eventEmitter:  cfg.EventEmitter,
		correlationFn: resilience.EnsureCorrelationFunc(cfg.CorrelationFn),
	}
}

// Execute runs operation with timeout enforcement.
func (m *Manager) Execute(ctx context.Context, operation string, fn func(ctx context.Context) error) error {
	timeout := m.GetTimeout(operation)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		done <- fn(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		m.emitTimeoutEvent(operation, timeout)
		return liberror.NewTimeoutError(m.serviceName, timeout)
	}
}

// GetTimeout returns the effective timeout for an operation.
func (m *Manager) GetTimeout(operation string) time.Duration {
	// Check for operation-specific timeout
	if m.config.PerOp != nil {
		if timeout, ok := m.config.PerOp[operation]; ok {
			return timeout
		}
	}

	// Return default timeout
	return m.config.Default
}

// WithTimeout returns a context with the appropriate timeout.
func (m *Manager) WithTimeout(ctx context.Context, operation string) (context.Context, context.CancelFunc) {
	timeout := m.GetTimeout(operation)
	return context.WithTimeout(ctx, timeout)
}

// emitTimeoutEvent emits a timeout event.
func (m *Manager) emitTimeoutEvent(operation string, timeout time.Duration) {
	event := resilience.NewEvent(resilience.EventTimeout, m.serviceName).
		WithCorrelationID(m.correlationFn()).
		WithMetadata("operation", operation).
		WithMetadata("timeout", timeout.String())

	resilience.EmitEvent(m.eventEmitter, *event)
}
