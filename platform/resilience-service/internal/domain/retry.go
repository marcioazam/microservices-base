package domain

import (
	"context"
	"time"
)

// RetryConfig defines retry behavior.
type RetryConfig struct {
	MaxAttempts     int           `json:"max_attempts" yaml:"maxAttempts"`
	BaseDelay       time.Duration `json:"base_delay" yaml:"baseDelay"`
	MaxDelay        time.Duration `json:"max_delay" yaml:"maxDelay"`
	Multiplier      float64       `json:"multiplier" yaml:"multiplier"`
	JitterPercent   float64       `json:"jitter_percent" yaml:"jitterPercent"`
	RetryableErrors []string      `json:"retryable_errors" yaml:"retryableErrors"`
}

// RetryHandler manages retry logic with backoff.
type RetryHandler interface {
	// Execute runs operation with retry policy.
	Execute(ctx context.Context, operation func() error) error

	// ExecuteWithCircuitBreaker runs operation with retry and circuit breaker.
	ExecuteWithCircuitBreaker(ctx context.Context, cb CircuitBreaker, operation func() error) error

	// CalculateDelay returns next retry delay for given attempt.
	CalculateDelay(attempt int) time.Duration
}

// RetryAttemptEvent represents a retry attempt for observability.
type RetryAttemptEvent struct {
	ServiceName   string        `json:"service_name"`
	Attempt       int           `json:"attempt"`
	MaxAttempts   int           `json:"max_attempts"`
	Delay         time.Duration `json:"delay"`
	Error         string        `json:"error,omitempty"`
	CorrelationID string        `json:"correlation_id"`
	Timestamp     time.Time     `json:"timestamp"`
}
