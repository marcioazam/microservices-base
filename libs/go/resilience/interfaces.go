package resilience

import (
	"context"
	"time"
)

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

// RateLimitDecision represents allow/deny decision.
type RateLimitDecision struct {
	Allowed    bool
	Remaining  int
	Limit      int
	ResetAt    time.Time
	RetryAfter time.Duration
}

// RateLimitHeaders contains rate limit response headers.
type RateLimitHeaders struct {
	Limit     int   `json:"X-RateLimit-Limit"`
	Remaining int   `json:"X-RateLimit-Remaining"`
	Reset     int64 `json:"X-RateLimit-Reset"`
}

// RateLimiter controls request throughput.
type RateLimiter interface {
	// Allow checks if request should be allowed.
	Allow(ctx context.Context, key string) (RateLimitDecision, error)

	// GetHeaders returns rate limit headers for response.
	GetHeaders(ctx context.Context, key string) (RateLimitHeaders, error)
}

// RateLimitEvent represents a rate limit hit for observability.
type RateLimitEvent struct {
	Key           string        `json:"key"`
	Allowed       bool          `json:"allowed"`
	Remaining     int           `json:"remaining"`
	RetryAfter    time.Duration `json:"retry_after,omitempty"`
	CorrelationID string        `json:"correlation_id"`
	Timestamp     time.Time     `json:"timestamp"`
}

// BulkheadMetrics reports bulkhead utilization.
type BulkheadMetrics struct {
	ActiveCount   int
	QueuedCount   int
	RejectedCount int64
}

// Bulkhead provides isolation through concurrency limits.
type Bulkhead interface {
	// Acquire attempts to acquire a permit.
	Acquire(ctx context.Context) error

	// Release returns a permit.
	Release()

	// GetMetrics returns current utilization.
	GetMetrics() BulkheadMetrics
}

// BulkheadManager manages multiple bulkhead partitions.
type BulkheadManager interface {
	// GetBulkhead returns the bulkhead for a partition.
	GetBulkhead(partition string) Bulkhead

	// GetAllMetrics returns metrics for all partitions.
	GetAllMetrics() map[string]BulkheadMetrics
}

// BulkheadRejectionEvent represents a bulkhead rejection for observability.
type BulkheadRejectionEvent struct {
	Partition     string    `json:"partition"`
	ActiveCount   int       `json:"active_count"`
	QueuedCount   int       `json:"queued_count"`
	CorrelationID string    `json:"correlation_id"`
	Timestamp     time.Time `json:"timestamp"`
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
