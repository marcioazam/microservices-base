// Package interfaces defines core domain interfaces with no external dependencies.
package interfaces

import (
	"context"

	"github.com/authcorp/libs/go/src/functional"
	"github.com/authcorp/libs/go/src/fault"
	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
)

// PolicyRepository manages policy persistence and retrieval with type-safe returns.
type PolicyRepository interface {
	// Get retrieves a policy by name, returning Option for type-safe null handling.
	Get(ctx context.Context, name string) functional.Option[*entities.Policy]

	// Save persists a policy and returns Result for error handling.
	Save(ctx context.Context, policy *entities.Policy) functional.Result[*entities.Policy]

	// Delete removes a policy by name.
	Delete(ctx context.Context, name string) error

	// List returns all policies.
	List(ctx context.Context) functional.Result[[]*entities.Policy]

	// Exists checks if a policy exists.
	Exists(ctx context.Context, name string) bool

	// Watch returns a channel for policy change events.
	Watch(ctx context.Context) (<-chan valueobjects.PolicyEvent, error)
}

// ResilienceExecutor applies resilience patterns to operations.
// Extends the generic executor from libs/go with service-specific methods.
type ResilienceExecutor interface {
	// Execute runs an operation with resilience patterns.
	Execute(ctx context.Context, policyName string, operation func() error) error

	// ExecuteWithResult runs an operation returning a typed result.
	ExecuteWithResult(ctx context.Context, policyName string, operation func() (any, error)) functional.Result[any]

	// RegisterPolicy registers a policy with the executor.
	RegisterPolicy(policy *entities.Policy) error

	// UnregisterPolicy removes a policy from the executor.
	UnregisterPolicy(policyName string)

	// GetPolicyNames returns all registered policy names.
	GetPolicyNames() []string
}

// TypedResilienceExecutor is a generic executor for type-safe operations.
type TypedResilienceExecutor[T any] interface {
	fault.ResilienceExecutor[T]
}

// HealthChecker provides health status for components.
type HealthChecker interface {
	Check(ctx context.Context) valueobjects.HealthStatus
	Name() string
}

// EventEmitter publishes domain events with type safety.
type EventEmitter interface {
	Emit(ctx context.Context, event valueobjects.DomainEvent) error
	EmitPolicyEvent(ctx context.Context, event valueobjects.PolicyEvent) error
}

// PolicyValidator validates policy configurations.
type PolicyValidator interface {
	Validate(policy *entities.Policy) functional.Result[*entities.Policy]
	ValidateCircuitBreaker(config *entities.CircuitBreakerConfig) functional.Result[*entities.CircuitBreakerConfig]
	ValidateRetry(config *entities.RetryConfig) functional.Result[*entities.RetryConfig]
	ValidateTimeout(config *entities.TimeoutConfig) functional.Result[*entities.TimeoutConfig]
	ValidateRateLimit(config *entities.RateLimitConfig) functional.Result[*entities.RateLimitConfig]
	ValidateBulkhead(config *entities.BulkheadConfig) functional.Result[*entities.BulkheadConfig]
}

// MetricsRecorder records resilience execution metrics.
// Extends the shared MetricsRecorder from libs/go.
type MetricsRecorder interface {
	fault.MetricsRecorder

	// RecordCacheStats records cache statistics.
	RecordCacheStats(ctx context.Context, hits, misses, evictions int64)
}

// Logger provides structured logging interface.
type Logger interface {
	Debug(ctx context.Context, msg string, fields map[string]any)
	Info(ctx context.Context, msg string, fields map[string]any)
	Warn(ctx context.Context, msg string, fields map[string]any)
	Error(ctx context.Context, msg string, err error, fields map[string]any)
}

// Tracer provides distributed tracing interface.
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, func())
	AddEvent(ctx context.Context, name string, attributes map[string]any)
	RecordError(ctx context.Context, err error)
}
