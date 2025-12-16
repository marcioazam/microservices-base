package domain

import (
	"context"
	"time"
)

// TimeoutConfig defines timeout behavior.
type TimeoutConfig struct {
	Default time.Duration            `json:"default" yaml:"default"`
	Max     time.Duration            `json:"max" yaml:"max"`
	PerOp   map[string]time.Duration `json:"per_operation" yaml:"perOperation"`
}

// TimeoutManager manages timeout enforcement.
type TimeoutManager interface {
	// Execute runs operation with timeout enforcement.
	Execute(ctx context.Context, operation string, fn func(ctx context.Context) error) error

	// GetTimeout returns the effective timeout for an operation.
	GetTimeout(operation string) time.Duration

	// WithTimeout returns a context with the appropriate timeout.
	WithTimeout(ctx context.Context, operation string) (context.Context, context.CancelFunc)
}

// TimeoutEvent represents a timeout occurrence for observability.
type TimeoutEvent struct {
	ServiceName   string        `json:"service_name"`
	Operation     string        `json:"operation"`
	Timeout       time.Duration `json:"timeout"`
	CorrelationID string        `json:"correlation_id"`
	Timestamp     time.Time     `json:"timestamp"`
}
