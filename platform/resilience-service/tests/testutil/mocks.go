// Package testutil provides shared test utilities and mocks.
package testutil

import (
	"context"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// MockMetricsRecorder provides a mock implementation of MetricsRecorder.
type MockMetricsRecorder struct{}

func (m *MockMetricsRecorder) RecordExecution(ctx context.Context, metrics valueobjects.ExecutionMetrics) {}
func (m *MockMetricsRecorder) RecordCircuitState(ctx context.Context, policyName string, state string) {}
func (m *MockMetricsRecorder) RecordRetryAttempt(ctx context.Context, policyName string, attempt int) {}
func (m *MockMetricsRecorder) RecordRateLimit(ctx context.Context, policyName string, limited bool) {}
func (m *MockMetricsRecorder) RecordBulkheadQueue(ctx context.Context, policyName string, queued bool) {}

// MockHealthChecker provides a mock implementation of HealthChecker.
type MockHealthChecker struct {
	CheckerName   string
	Status valueobjects.HealthStatus
}

func (m *MockHealthChecker) Check(ctx context.Context) valueobjects.HealthStatus {
	return m.Status
}

func (m *MockHealthChecker) Name() string {
	return m.CheckerName
}

// GetMockTracer returns a noop tracer for testing.
func GetMockTracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}

// MockEventEmitter provides a mock implementation of EventEmitter.
type MockEventEmitter struct{}

func (m *MockEventEmitter) Emit(ctx context.Context, event valueobjects.DomainEvent) error {
	return nil
}

// MockPolicyRepository provides a mock implementation of PolicyRepository.
type MockPolicyRepository struct{}

func (m *MockPolicyRepository) Get(ctx context.Context, name string) (*entities.Policy, error) {
	return nil, nil
}

func (m *MockPolicyRepository) Save(ctx context.Context, policy *entities.Policy) error {
	return nil
}

func (m *MockPolicyRepository) Delete(ctx context.Context, name string) error {
	return nil
}

func (m *MockPolicyRepository) List(ctx context.Context) ([]*entities.Policy, error) {
	return nil, nil
}

func (m *MockPolicyRepository) Watch(ctx context.Context) (<-chan valueobjects.PolicyEvent, error) {
	ch := make(chan valueobjects.PolicyEvent)
	close(ch)
	return ch, nil
}

// MockPolicyValidator provides a mock implementation of PolicyValidator.
type MockPolicyValidator struct{}

func (m *MockPolicyValidator) Validate(policy *entities.Policy) error { return nil }
func (m *MockPolicyValidator) ValidateCircuitBreaker(config *entities.CircuitBreakerConfig) error { return nil }
func (m *MockPolicyValidator) ValidateRetry(config *entities.RetryConfig) error { return nil }
func (m *MockPolicyValidator) ValidateTimeout(config *entities.TimeoutConfig) error { return nil }
func (m *MockPolicyValidator) ValidateRateLimit(config *entities.RateLimitConfig) error { return nil }
func (m *MockPolicyValidator) ValidateBulkhead(config *entities.BulkheadConfig) error { return nil }