package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/functional"
	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
)

// Mock implementations for testing

type mockRepository struct {
	policies map[string]*entities.Policy
}

func newMockRepository() *mockRepository {
	return &mockRepository{policies: make(map[string]*entities.Policy)}
}

func (m *mockRepository) Get(ctx context.Context, name string) functional.Option[*entities.Policy] {
	if p, ok := m.policies[name]; ok {
		return functional.Some(p)
	}
	return functional.None[*entities.Policy]()
}

func (m *mockRepository) Save(ctx context.Context, policy *entities.Policy) functional.Result[*entities.Policy] {
	m.policies[policy.Name()] = policy
	return functional.Ok(policy)
}

func (m *mockRepository) Delete(ctx context.Context, name string) error {
	delete(m.policies, name)
	return nil
}

func (m *mockRepository) List(ctx context.Context) functional.Result[[]*entities.Policy] {
	policies := make([]*entities.Policy, 0, len(m.policies))
	for _, p := range m.policies {
		policies = append(policies, p)
	}
	return functional.Ok(policies)
}

func (m *mockRepository) Exists(ctx context.Context, name string) bool {
	_, ok := m.policies[name]
	return ok
}

func (m *mockRepository) Watch(ctx context.Context) (<-chan valueobjects.PolicyEvent, error) {
	return make(chan valueobjects.PolicyEvent), nil
}

type mockValidator struct{}

func (m *mockValidator) Validate(policy *entities.Policy) functional.Result[*entities.Policy] {
	return functional.Ok(policy)
}

func (m *mockValidator) ValidateCircuitBreaker(config *entities.CircuitBreakerConfig) functional.Result[*entities.CircuitBreakerConfig] {
	return functional.Ok(config)
}

func (m *mockValidator) ValidateRetry(config *entities.RetryConfig) functional.Result[*entities.RetryConfig] {
	return functional.Ok(config)
}

func (m *mockValidator) ValidateTimeout(config *entities.TimeoutConfig) functional.Result[*entities.TimeoutConfig] {
	return functional.Ok(config)
}

func (m *mockValidator) ValidateRateLimit(config *entities.RateLimitConfig) functional.Result[*entities.RateLimitConfig] {
	return functional.Ok(config)
}

func (m *mockValidator) ValidateBulkhead(config *entities.BulkheadConfig) functional.Result[*entities.BulkheadConfig] {
	return functional.Ok(config)
}

type mockEmitter struct {
	events []valueobjects.PolicyEvent
}

func (m *mockEmitter) Emit(ctx context.Context, event valueobjects.DomainEvent) error {
	return nil
}

func (m *mockEmitter) EmitPolicyEvent(ctx context.Context, event valueobjects.PolicyEvent) error {
	m.events = append(m.events, event)
	return nil
}

func TestPolicyEntityWithOptionTypes(t *testing.T) {
	policy, err := entities.NewPolicy("test-policy")
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	// Initially all configs should be None
	if policy.CircuitBreaker().IsSome() {
		t.Error("CircuitBreaker should be None initially")
	}
	if policy.Retry().IsSome() {
		t.Error("Retry should be None initially")
	}
	if policy.Timeout().IsSome() {
		t.Error("Timeout should be None initially")
	}
	if policy.RateLimit().IsSome() {
		t.Error("RateLimit should be None initially")
	}
	if policy.Bulkhead().IsSome() {
		t.Error("Bulkhead should be None initially")
	}
}

func TestPolicySetCircuitBreakerReturnsResult(t *testing.T) {
	policy, _ := entities.NewPolicy("test-policy")

	cbConfig := &entities.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ProbeCount:       2,
	}

	result := policy.SetCircuitBreaker(cbConfig)

	if result.IsErr() {
		t.Fatalf("SetCircuitBreaker failed: %v", result.UnwrapErr())
	}

	if !policy.CircuitBreaker().IsSome() {
		t.Error("CircuitBreaker should be Some after setting")
	}

	cb := policy.CircuitBreaker().Unwrap()
	if cb.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %d, want 5", cb.FailureThreshold)
	}
}

func TestPolicySetRetryReturnsResult(t *testing.T) {
	policy, _ := entities.NewPolicy("test-policy")

	retryConfig := &entities.RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		Multiplier:    2.0,
		JitterPercent: 0.1,
	}

	result := policy.SetRetry(retryConfig)

	if result.IsErr() {
		t.Fatalf("SetRetry failed: %v", result.UnwrapErr())
	}

	if !policy.Retry().IsSome() {
		t.Error("Retry should be Some after setting")
	}
}

func TestPolicyValidateResultReturnsResult(t *testing.T) {
	policy, _ := entities.NewPolicy("test-policy")

	// Policy without any pattern should fail validation
	result := policy.ValidateResult()

	if result.IsOk() {
		t.Error("Validation should fail for policy without patterns")
	}

	// Add a pattern
	cbConfig := &entities.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ProbeCount:       2,
	}
	policy.SetCircuitBreaker(cbConfig)

	result = policy.ValidateResult()

	if result.IsErr() {
		t.Errorf("Validation should pass: %v", result.UnwrapErr())
	}
}

func TestPolicyHasAnyPattern(t *testing.T) {
	policy, _ := entities.NewPolicy("test-policy")

	if policy.HasAnyPattern() {
		t.Error("HasAnyPattern should be false initially")
	}

	cbConfig := &entities.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ProbeCount:       2,
	}
	policy.SetCircuitBreaker(cbConfig)

	if !policy.HasAnyPattern() {
		t.Error("HasAnyPattern should be true after setting circuit breaker")
	}
}

func TestPolicyClone(t *testing.T) {
	policy, _ := entities.NewPolicy("test-policy")

	cbConfig := &entities.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ProbeCount:       2,
	}
	policy.SetCircuitBreaker(cbConfig)

	clone := policy.Clone()

	if clone.Name() != policy.Name() {
		t.Errorf("Clone name = %s, want %s", clone.Name(), policy.Name())
	}

	if !clone.CircuitBreaker().IsSome() {
		t.Error("Clone should have CircuitBreaker")
	}

	// Modify original, clone should not change
	policy.IncrementVersion()

	if clone.Version() == policy.Version() {
		t.Error("Clone version should not change when original changes")
	}
}

func TestRepositoryGetReturnsOption(t *testing.T) {
	repo := newMockRepository()
	ctx := context.Background()

	// Get non-existent policy
	opt := repo.Get(ctx, "non-existent")
	if opt.IsSome() {
		t.Error("Get should return None for non-existent policy")
	}

	// Save and get
	policy, _ := entities.NewPolicy("test-policy")
	cbConfig := &entities.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ProbeCount:       2,
	}
	policy.SetCircuitBreaker(cbConfig)
	repo.Save(ctx, policy)

	opt = repo.Get(ctx, "test-policy")
	if !opt.IsSome() {
		t.Error("Get should return Some for existing policy")
	}
}

func TestRepositorySaveReturnsResult(t *testing.T) {
	repo := newMockRepository()
	ctx := context.Background()

	policy, _ := entities.NewPolicy("test-policy")
	cbConfig := &entities.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ProbeCount:       2,
	}
	policy.SetCircuitBreaker(cbConfig)

	result := repo.Save(ctx, policy)

	if result.IsErr() {
		t.Fatalf("Save failed: %v", result.UnwrapErr())
	}

	saved := result.Unwrap()
	if saved.Name() != "test-policy" {
		t.Errorf("Saved policy name = %s, want test-policy", saved.Name())
	}
}

func TestRepositoryListReturnsResult(t *testing.T) {
	repo := newMockRepository()
	ctx := context.Background()

	// Empty list
	result := repo.List(ctx)
	if result.IsErr() {
		t.Fatalf("List failed: %v", result.UnwrapErr())
	}

	if len(result.Unwrap()) != 0 {
		t.Errorf("Expected empty list, got %d items", len(result.Unwrap()))
	}

	// Add policies
	for i := 0; i < 3; i++ {
		policy, _ := entities.NewPolicy("policy-" + string(rune('a'+i)))
		cbConfig := &entities.CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 3,
			Timeout:          30 * time.Second,
			ProbeCount:       2,
		}
		policy.SetCircuitBreaker(cbConfig)
		repo.Save(ctx, policy)
	}

	result = repo.List(ctx)
	if result.IsErr() {
		t.Fatalf("List failed: %v", result.UnwrapErr())
	}

	if len(result.Unwrap()) != 3 {
		t.Errorf("Expected 3 policies, got %d", len(result.Unwrap()))
	}
}
