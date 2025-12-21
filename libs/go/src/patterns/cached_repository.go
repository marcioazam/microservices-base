// Package patterns provides cached repository wrapper.
package patterns

import (
	"context"
	"time"

	"github.com/authcorp/libs/go/src/functional"
)

// Cache defines the interface for a generic cache.
type Cache[K comparable, V any] interface {
	Get(key K) functional.Option[V]
	Put(key K, value V)
	Remove(key K) bool
	Clear()
	Size() int
}

// CacheStats holds cache statistics.
type CacheStats struct {
	Hits       int64   `json:"hits"`
	Misses     int64   `json:"misses"`
	Evictions  int64   `json:"evictions"`
	Size       int     `json:"size"`
	Capacity   int     `json:"capacity"`
	HitRate    float64 `json:"hit_rate"`
}

// CachedRepository wraps a repository with caching.
type CachedRepository[T any, ID comparable] struct {
	inner     Repository[T, ID]
	cache     Cache[ID, T]
	extractor IDExtractor[T, ID]
	ttl       time.Duration
	stats     CacheStats
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

// NewCachedRepository creates a cached repository wrapper.
func NewCachedRepository[T any, ID comparable](
	inner Repository[T, ID],
	cache Cache[ID, T],
	extractor IDExtractor[T, ID],
) *CachedRepository[T, ID] {
	return &CachedRepository[T, ID]{
		inner:     inner,
		cache:     cache,
		extractor: extractor,
	}
}

// Get retrieves an entity, checking cache first.
func (r *CachedRepository[T, ID]) Get(ctx context.Context, id ID) functional.Option[T] {
	// Check cache first
	if opt := r.cache.Get(id); opt.IsSome() {
		r.stats.Hits++
		return opt
	}

	r.stats.Misses++

	// Fallback to inner repository
	opt := r.inner.Get(ctx, id)
	if opt.IsSome() {
		r.cache.Put(id, opt.Unwrap())
	}
	return opt
}

// Save persists an entity and updates cache.
func (r *CachedRepository[T, ID]) Save(ctx context.Context, entity T) functional.Result[T] {
	result := r.inner.Save(ctx, entity)
	if result.IsOk() {
		id := r.extractor(entity)
		r.cache.Put(id, result.Unwrap())
	}
	return result
}

// Delete removes an entity and invalidates cache.
func (r *CachedRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	r.cache.Remove(id)
	return r.inner.Delete(ctx, id)
}

// List returns all entities (bypasses cache).
func (r *CachedRepository[T, ID]) List(ctx context.Context) functional.Result[[]T] {
	return r.inner.List(ctx)
}

// Exists checks if an entity exists.
func (r *CachedRepository[T, ID]) Exists(ctx context.Context, id ID) bool {
	if r.cache.Get(id).IsSome() {
		return true
	}
	return r.inner.Exists(ctx, id)
}

// Invalidate removes an entry from cache.
func (r *CachedRepository[T, ID]) Invalidate(id ID) {
	r.cache.Remove(id)
}

// InvalidateAll clears the entire cache.
func (r *CachedRepository[T, ID]) InvalidateAll() {
	r.cache.Clear()
}

// Stats returns cache statistics.
func (r *CachedRepository[T, ID]) Stats() CacheStats {
	stats := r.stats
	stats.Size = r.cache.Size()
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}
	return stats
}

// Ensure CachedRepository implements Repository.
var _ Repository[any, string] = (*CachedRepository[any, string])(nil)
