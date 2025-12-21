// Package repositories provides cached repository implementation using LRU cache.
package repositories

import (
	"context"
	"log/slog"
	"time"

	"github.com/authcorp/libs/go/src/collections"
	"github.com/authcorp/libs/go/src/functional"
	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/interfaces"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
)

// CachedPolicyRepository wraps a repository with LRU caching.
type CachedPolicyRepository struct {
	inner   interfaces.PolicyRepository
	cache   *collections.LRUCache[string, *entities.Policy]
	logger  *slog.Logger
	eventCh chan valueobjects.PolicyEvent
}

// CachedRepositoryConfig configures the cached repository.
type CachedRepositoryConfig struct {
	CacheSize int
	TTL       time.Duration
}

// DefaultCachedRepositoryConfig returns sensible defaults.
func DefaultCachedRepositoryConfig() CachedRepositoryConfig {
	return CachedRepositoryConfig{
		CacheSize: 1000,
		TTL:       5 * time.Minute,
	}
}

// NewCachedPolicyRepository creates a cached policy repository.
func NewCachedPolicyRepository(
	inner interfaces.PolicyRepository,
	config CachedRepositoryConfig,
	logger *slog.Logger,
) *CachedPolicyRepository {
	cache := collections.NewLRUCache[string, *entities.Policy](config.CacheSize).
		WithTTL(config.TTL).
		WithEvictCallback(func(key string, value *entities.Policy) {
			logger.Debug("policy evicted from cache",
				slog.String("policy_name", key),
				slog.Int("version", value.Version()))
		})

	return &CachedPolicyRepository{
		inner:   inner,
		cache:   cache,
		logger:  logger,
		eventCh: make(chan valueobjects.PolicyEvent, 100),
	}
}

// Get retrieves a policy, checking cache first.
func (r *CachedPolicyRepository) Get(ctx context.Context, name string) functional.Option[*entities.Policy] {
	// Check cache first
	if opt := r.cache.Get(name); opt.IsSome() {
		r.logger.DebugContext(ctx, "cache hit",
			slog.String("policy_name", name))
		return opt
	}

	r.logger.DebugContext(ctx, "cache miss",
		slog.String("policy_name", name))

	// Fallback to inner repository
	opt := r.inner.Get(ctx, name)
	if opt.IsSome() {
		r.cache.Put(name, opt.Unwrap())
	}
	return opt
}

// Save persists a policy and updates cache.
func (r *CachedPolicyRepository) Save(ctx context.Context, policy *entities.Policy) functional.Result[*entities.Policy] {
	result := r.inner.Save(ctx, policy)
	if result.IsOk() {
		r.cache.Put(policy.Name(), result.Unwrap())
		r.logger.DebugContext(ctx, "policy cached after save",
			slog.String("policy_name", policy.Name()),
			slog.Int("version", policy.Version()))
	}
	return result
}

// Delete removes a policy and invalidates cache.
func (r *CachedPolicyRepository) Delete(ctx context.Context, name string) error {
	r.cache.Remove(name)
	r.logger.DebugContext(ctx, "policy removed from cache",
		slog.String("policy_name", name))
	return r.inner.Delete(ctx, name)
}

// List returns all policies (bypasses cache for consistency).
func (r *CachedPolicyRepository) List(ctx context.Context) functional.Result[[]*entities.Policy] {
	return r.inner.List(ctx)
}

// Exists checks if a policy exists.
func (r *CachedPolicyRepository) Exists(ctx context.Context, name string) bool {
	if r.cache.Get(name).IsSome() {
		return true
	}
	return r.inner.Exists(ctx, name)
}

// Watch returns a channel for policy change events.
func (r *CachedPolicyRepository) Watch(ctx context.Context) (<-chan valueobjects.PolicyEvent, error) {
	return r.inner.Watch(ctx)
}

// Invalidate removes an entry from cache.
func (r *CachedPolicyRepository) Invalidate(name string) {
	r.cache.Remove(name)
}

// InvalidateAll clears the entire cache.
func (r *CachedPolicyRepository) InvalidateAll() {
	r.cache.Clear()
}

// Stats returns cache statistics.
func (r *CachedPolicyRepository) Stats() collections.Stats {
	return r.cache.Stats()
}

// Cleanup removes expired entries from cache.
func (r *CachedPolicyRepository) Cleanup() int {
	return r.cache.Cleanup()
}

// Ensure CachedPolicyRepository implements PolicyRepository.
var _ interfaces.PolicyRepository = (*CachedPolicyRepository)(nil)
