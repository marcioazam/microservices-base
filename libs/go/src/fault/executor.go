// Package fault provides generic resilience patterns with type safety.
package fault

import (
	"context"
	"time"

	"github.com/authcorp/libs/go/src/functional"
)

// ResilienceExecutor applies resilience patterns to operations with type safety.
// This is the generic interface that all resilience executors should implement.
type ResilienceExecutor[T any] interface {
	// Execute runs an operation with resilience patterns applied.
	// Returns error if the operation fails after all resilience attempts.
	Execute(ctx context.Context, policyName string, op func() error) error

	// ExecuteWithResult runs an operation returning a typed result.
	// Returns Result[T] for type-safe error handling.
	ExecuteWithResult(ctx context.Context, policyName string, op func() (T, error)) functional.Result[T]

	// RegisterPolicy registers a resilience policy configuration.
	RegisterPolicy(policy PolicyConfig) error

	// UnregisterPolicy removes a policy from the executor.
	UnregisterPolicy(policyName string)

	// GetPolicyNames returns all registered policy names.
	GetPolicyNames() []string
}

// PolicyConfig defines a resilience policy configuration.
type PolicyConfig struct {
	Name           string
	CircuitBreaker *CircuitBreakerPolicyConfig
	Retry          *RetryPolicyConfig
	Timeout        *TimeoutPolicyConfig
	RateLimit      *RateLimitPolicyConfig
	Bulkhead       *BulkheadPolicyConfig
}

// CircuitBreakerPolicyConfig defines circuit breaker parameters.
type CircuitBreakerPolicyConfig struct {
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
	HalfOpenMaxCalls int
}

// RetryPolicyConfig defines retry behavior parameters.
type RetryPolicyConfig struct {
	MaxAttempts   int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	Multiplier    float64
	JitterPercent float64
}

// TimeoutPolicyConfig defines timeout parameters.
type TimeoutPolicyConfig struct {
	Default time.Duration
	Max     time.Duration
}

// RateLimitPolicyConfig defines rate limiting parameters.
type RateLimitPolicyConfig struct {
	Algorithm string
	Limit     int
	Window    time.Duration
	BurstSize int
}

// BulkheadPolicyConfig defines bulkhead isolation parameters.
type BulkheadPolicyConfig struct {
	MaxConcurrent int
	MaxQueue      int
	QueueTimeout  time.Duration
}

// ExecutorConfig configures the resilience executor behavior.
type ExecutorConfig struct {
	DefaultTimeout time.Duration
	MetricsEnabled bool
	TracingEnabled bool
	LoggerEnabled  bool
}

// DefaultExecutorConfig returns sensible defaults for executor configuration.
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		DefaultTimeout: 30 * time.Second,
		MetricsEnabled: true,
		TracingEnabled: true,
		LoggerEnabled:  true,
	}
}

// ExecuteFunc is a helper to execute a function with resilience using Result type.
func ExecuteFunc[T any](
	ctx context.Context,
	executor ResilienceExecutor[T],
	policyName string,
	op func() (T, error),
) functional.Result[T] {
	return executor.ExecuteWithResult(ctx, policyName, op)
}

// ExecuteSimple is a helper to execute a simple function with resilience.
func ExecuteSimple[T any](
	ctx context.Context,
	executor ResilienceExecutor[T],
	policyName string,
	op func() error,
) error {
	return executor.Execute(ctx, policyName, op)
}
