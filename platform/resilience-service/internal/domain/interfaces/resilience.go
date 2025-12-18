// Package interfaces defines core domain interfaces with no external dependencies.
package interfaces

import (
	"context"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
)

// ResiliencePolicy represents a complete resilience configuration.
type ResiliencePolicy interface {
	Name() string
	Version() int
	Execute(ctx context.Context, fn func() error) error
	Validate() error
}

// PolicyRepository manages policy persistence and retrieval.
type PolicyRepository interface {
	Get(ctx context.Context, name string) (*entities.Policy, error)
	Save(ctx context.Context, policy *entities.Policy) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]*entities.Policy, error)
	Watch(ctx context.Context) (<-chan valueobjects.PolicyEvent, error)
}

// ResilienceExecutor applies resilience patterns to operations.
type ResilienceExecutor interface {
	Execute(ctx context.Context, policyName string, operation func() error) error
	ExecuteWithResult(ctx context.Context, policyName string, operation func() (any, error)) (any, error)
}

// HealthChecker provides health status for components.
type HealthChecker interface {
	Check(ctx context.Context) valueobjects.HealthStatus
	Name() string
}

// EventEmitter publishes domain events.
type EventEmitter interface {
	Emit(ctx context.Context, event valueobjects.DomainEvent) error
}

// PolicyValidator validates policy configurations.
type PolicyValidator interface {
	Validate(policy *entities.Policy) error
	ValidateCircuitBreaker(config *entities.CircuitBreakerConfig) error
	ValidateRetry(config *entities.RetryConfig) error
	ValidateTimeout(config *entities.TimeoutConfig) error
	ValidateRateLimit(config *entities.RateLimitConfig) error
	ValidateBulkhead(config *entities.BulkheadConfig) error
}

// MetricsRecorder records resilience execution metrics.
type MetricsRecorder interface {
	RecordExecution(ctx context.Context, metrics valueobjects.ExecutionMetrics)
	RecordCircuitState(ctx context.Context, policyName string, state string)
	RecordRetryAttempt(ctx context.Context, policyName string, attempt int)
	RecordRateLimit(ctx context.Context, policyName string, limited bool)
	RecordBulkheadQueue(ctx context.Context, policyName string, queued bool)
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