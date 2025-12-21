package integration_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/functional"
	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/repositories"
)

// mockPolicyRepository is a simple in-memory repository for testing.
type mockPolicyRepository struct {
	policies map[string]*entities.Policy
	getCalls int
}

func newMockPolicyRepository() *mockPolicyRepository {
	return &mockPolicyRepository{
		policies: make(map[string]*entities.Policy),
	}
}

func (m *mockPolicyRepository) Get(ctx context.Context, name string) functional.Option[*entities.Policy] {
	m.getCalls++
	if p, ok := m.policies[name]; ok {
		return functional.Some(p)
	}
	return functional.None[*entities.Policy]()
}

func (m *mockPolicyRepository) Save(ctx context.Context, policy *entities.Policy) functional.Result[*entities.Policy] {
	m.policies[policy.Name()] = policy
	return functional.Ok(policy)
}

func (m *mockPolicyRepository) Delete(ctx context.Context, name string) error {
	delete(m.policies, name)
	return nil
}

func (m *mockPolicyRepository) List(ctx context.Context) functional.Result[[]*entities.Policy] {
	policies := make([]*entities.Policy, 0, len(m.policies))
	for _, p := range m.policies {
		policies = append(policies, p)
	}
	return functional.Ok(policies)
}

func (m *mockPolicyRepository) Exists(ctx context.Context, name string) bool {
	_, ok := m.policies[name]
	return ok
}

func (m *mockPolicyRepository) Watch(ctx context.Context) (<-chan valueobjects.PolicyEvent, error) {
	return make(chan valueobjects.PolicyEvent), nil
}

func TestCachedRepositoryCacheHit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	inner := newMockPolicyRepository()

	config := repositories.CachedRepositoryConfig{
		CacheSize: 100,
		TTL:       5 * time.Minute,
	}

	cached := repositories.NewCachedPolicyRepository(inner, config, logger)

	policy, _ := entities.NewPolicy("test-policy")
	cbConfig := &entities.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ProbeCount:       2,
	}
	policy.SetCircuitBreaker(cbConfig)

	ctx := context.Background()
	cached.Save(ctx, policy)

	inner.getCalls = 0

	result1 := cached.Get(ctx, "test-policy")
	if !result1.IsSome() {
		t.Fatal("Expected policy to be found")
	}

	result2 := cached.Get(ctx, "test-policy")
	if !result2.IsSome() {
		t.Fatal("Expected policy to be found")
	}

	if inner.getCalls != 0 {
		t.Errorf("Expected 0 inner Get calls, got %d", inner.getCalls)
	}

	stats := cached.Stats()
	if stats.Hits < 2 {
		t.Errorf("Expected at least 2 cache hits, got %d", stats.Hits)
	}
}

func TestCachedRepositoryCacheMiss(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	inner := newMockPolicyRepository()

	config := repositories.CachedRepositoryConfig{
		CacheSize: 100,
		TTL:       5 * time.Minute,
	}

	cached := repositories.NewCachedPolicyRepository(inner, config, logger)

	policy, _ := entities.NewPolicy("test-policy")
	cbConfig := &entities.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ProbeCount:       2,
	}
	policy.SetCircuitBreaker(cbConfig)
	inner.policies["test-policy"] = policy

	ctx := context.Background()

	result1 := cached.Get(ctx, "test-policy")
	if !result1.IsSome() {
		t.Fatal("Expected policy to be found")
	}

	if inner.getCalls != 1 {
		t.Errorf("Expected 1 inner Get call, got %d", inner.getCalls)
	}

	result2 := cached.Get(ctx, "test-policy")
	if !result2.IsSome() {
		t.Fatal("Expected policy to be found")
	}

	if inner.getCalls != 1 {
		t.Errorf("Expected 1 inner Get call, got %d", inner.getCalls)
	}
}

func TestCachedRepositoryInvalidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	inner := newMockPolicyRepository()

	config := repositories.CachedRepositoryConfig{
		CacheSize: 100,
		TTL:       5 * time.Minute,
	}

	cached := repositories.NewCachedPolicyRepository(inner, config, logger)

	policy, _ := entities.NewPolicy("test-policy")
	cbConfig := &entities.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ProbeCount:       2,
	}
	policy.SetCircuitBreaker(cbConfig)

	ctx := context.Background()
	cached.Save(ctx, policy)

	cached.Invalidate("test-policy")

	inner.getCalls = 0

	cached.Get(ctx, "test-policy")

	if inner.getCalls != 1 {
		t.Errorf("Expected 1 inner Get call after invalidation, got %d", inner.getCalls)
	}
}

func TestCachedRepositoryDelete(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	inner := newMockPolicyRepository()

	config := repositories.CachedRepositoryConfig{
		CacheSize: 100,
		TTL:       5 * time.Minute,
	}

	cached := repositories.NewCachedPolicyRepository(inner, config, logger)

	policy, _ := entities.NewPolicy("test-policy")
	cbConfig := &entities.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ProbeCount:       2,
	}
	policy.SetCircuitBreaker(cbConfig)

	ctx := context.Background()
	cached.Save(ctx, policy)

	err := cached.Delete(ctx, "test-policy")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	result := cached.Get(ctx, "test-policy")
	if result.IsSome() {
		t.Error("Expected policy to be deleted")
	}
}

func TestCachedRepositoryStats(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	inner := newMockPolicyRepository()

	config := repositories.CachedRepositoryConfig{
		CacheSize: 100,
		TTL:       5 * time.Minute,
	}

	cached := repositories.NewCachedPolicyRepository(inner, config, logger)

	policy, _ := entities.NewPolicy("test-policy")
	cbConfig := &entities.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ProbeCount:       2,
	}
	policy.SetCircuitBreaker(cbConfig)

	ctx := context.Background()
	cached.Save(ctx, policy)

	for i := 0; i < 10; i++ {
		cached.Get(ctx, "test-policy")
	}

	stats := cached.Stats()

	if stats.Hits < 10 {
		t.Errorf("Expected at least 10 hits, got %d", stats.Hits)
	}

	if stats.Size != 1 {
		t.Errorf("Expected size 1, got %d", stats.Size)
	}
}
