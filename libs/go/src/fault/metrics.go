// Package fault provides execution metrics for resilience patterns.
package fault

import (
	"context"
	"time"
)

// ExecutionMetrics captures resilience execution statistics.
// This is the shared metrics type used across all resilience implementations.
type ExecutionMetrics struct {
	PolicyName     string        `json:"policy_name"`
	ExecutionTime  time.Duration `json:"execution_time"`
	Success        bool          `json:"success"`
	CircuitState   string        `json:"circuit_state,omitempty"`
	RetryAttempts  int           `json:"retry_attempts,omitempty"`
	RateLimited    bool          `json:"rate_limited,omitempty"`
	BulkheadQueued bool          `json:"bulkhead_queued,omitempty"`
	Timestamp      time.Time     `json:"timestamp"`
	CorrelationID  string        `json:"correlation_id,omitempty"`
	TraceID        string        `json:"trace_id,omitempty"`
}

// NewExecutionMetrics creates new execution metrics with required fields.
func NewExecutionMetrics(policyName string, executionTime time.Duration, success bool) ExecutionMetrics {
	return ExecutionMetrics{
		PolicyName:    policyName,
		ExecutionTime: executionTime,
		Success:       success,
		Timestamp:     time.Now().UTC(),
	}
}

// WithCircuitState adds circuit breaker state to metrics.
func (e ExecutionMetrics) WithCircuitState(state string) ExecutionMetrics {
	e.CircuitState = state
	return e
}

// WithRetryAttempts adds retry attempts count to metrics.
func (e ExecutionMetrics) WithRetryAttempts(attempts int) ExecutionMetrics {
	e.RetryAttempts = attempts
	return e
}

// WithRateLimit adds rate limiting information to metrics.
func (e ExecutionMetrics) WithRateLimit(limited bool) ExecutionMetrics {
	e.RateLimited = limited
	return e
}

// WithBulkheadQueue adds bulkhead queue information to metrics.
func (e ExecutionMetrics) WithBulkheadQueue(queued bool) ExecutionMetrics {
	e.BulkheadQueued = queued
	return e
}

// WithCorrelationID adds correlation ID for distributed tracing.
func (e ExecutionMetrics) WithCorrelationID(id string) ExecutionMetrics {
	e.CorrelationID = id
	return e
}

// WithTraceID adds trace ID for distributed tracing.
func (e ExecutionMetrics) WithTraceID(id string) ExecutionMetrics {
	e.TraceID = id
	return e
}

// IsSuccessful returns true if the execution was successful.
func (e ExecutionMetrics) IsSuccessful() bool {
	return e.Success
}

// WasRetried returns true if the operation was retried.
func (e ExecutionMetrics) WasRetried() bool {
	return e.RetryAttempts > 0
}

// WasRateLimited returns true if the operation was rate limited.
func (e ExecutionMetrics) WasRateLimited() bool {
	return e.RateLimited
}

// MetricsRecorder records execution metrics with type safety.
// Implementations should be thread-safe.
type MetricsRecorder interface {
	// RecordExecution records metrics for a completed execution.
	RecordExecution(ctx context.Context, metrics ExecutionMetrics)

	// RecordCircuitState records a circuit breaker state change.
	RecordCircuitState(ctx context.Context, policyName string, state string)

	// RecordRetryAttempt records a retry attempt.
	RecordRetryAttempt(ctx context.Context, policyName string, attempt int)

	// RecordRateLimit records a rate limit event.
	RecordRateLimit(ctx context.Context, policyName string, limited bool)

	// RecordBulkheadQueue records a bulkhead queue event.
	RecordBulkheadQueue(ctx context.Context, policyName string, queued bool)
}

// NoOpMetricsRecorder is a metrics recorder that does nothing.
// Useful for testing or when metrics are disabled.
type NoOpMetricsRecorder struct{}

// RecordExecution does nothing.
func (n NoOpMetricsRecorder) RecordExecution(ctx context.Context, metrics ExecutionMetrics) {}

// RecordCircuitState does nothing.
func (n NoOpMetricsRecorder) RecordCircuitState(ctx context.Context, policyName string, state string) {
}

// RecordRetryAttempt does nothing.
func (n NoOpMetricsRecorder) RecordRetryAttempt(ctx context.Context, policyName string, attempt int) {
}

// RecordRateLimit does nothing.
func (n NoOpMetricsRecorder) RecordRateLimit(ctx context.Context, policyName string, limited bool) {}

// RecordBulkheadQueue does nothing.
func (n NoOpMetricsRecorder) RecordBulkheadQueue(ctx context.Context, policyName string, queued bool) {
}

// Ensure NoOpMetricsRecorder implements MetricsRecorder.
var _ MetricsRecorder = NoOpMetricsRecorder{}
