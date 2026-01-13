// Package patterns provides cached repository wrapper.
package patterns

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/authcorp/libs/go/src/functional"
)

// Serializer defines serialization for cache storage.
type Serializer[T any] interface {
	Serialize(T) ([]byte, error)
	Deserialize([]byte) (T, error)
}

// JSONSerializer provides JSON serialization.
type JSONSerializer[T any] struct{}

// Serialize converts entity to JSON bytes.
func (s JSONSerializer[T]) Serialize(entity T) ([]byte, error) {
	return json.Marshal(entity)
}

// Deserialize converts JSON bytes to entity.
func (s JSONSerializer[T]) Deserialize(data []byte) (T, error) {
	var entity T
	err := json.Unmarshal(data, &entity)
	return entity, err
}

// CacheClient defines the interface for distributed cache operations.
// This interface is implemented by libs/go/src/cache.Client
type CacheClient interface {
	Get(ctx context.Context, key string) functional.Result[[]byte]
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
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

// CachedRepository wraps a repository with distributed caching via cache-service.
type CachedRepository[T any, ID comparable] struct {
	inner      Repository[T, ID]
	cache      CacheClient
	serializer Serializer[T]
	keyPrefix  string
	ttl        time.Duration
	extractor  IDExtractor[T, ID]
	stats      CacheStats
}

// CachedRepositoryConfig configures the cached repository.
type CachedRepositoryConfig struct {
	KeyPrefix string
	TTL       time.Duration
}

// DefaultCachedRepositoryConfig returns sensible defaults.
func DefaultCachedRepositoryConfig() CachedRepositoryConfig {
	return CachedRepositoryConfig{
		KeyPrefix: "repo",
		TTL:       5 * time.Minute,
	}
}

// NewCachedRepository creates a cached repository using cache-service client.
func NewCachedRepository[T any, ID comparable](
	inner Repository[T, ID],
	cache CacheClient,
	serializer Serializer[T],
	extractor IDExtractor[T, ID],
	config CachedRepositoryConfig,
) *CachedRepository[T, ID] {
	return &CachedRepository[T, ID]{
		inner:      inner,
		cache:      cache,
		serializer: serializer,
		keyPrefix:  config.KeyPrefix,
		ttl:        config.TTL,
		extractor:  extractor,
	}
}

// NewCachedRepositoryWithJSON creates a cached repository with JSON serialization.
func NewCachedRepositoryWithJSON[T any, ID comparable](
	inner Repository[T, ID],
	cache CacheClient,
	extractor IDExtractor[T, ID],
	config CachedRepositoryConfig,
) *CachedRepository[T, ID] {
	return NewCachedRepository(inner, cache, JSONSerializer[T]{}, extractor, config)
}

// Get retrieves an entity, checking cache first.
func (r *CachedRepository[T, ID]) Get(ctx context.Context, id ID) functional.Option[T] {
	key := r.buildKey(id)

	// Check cache first
	result := r.cache.Get(ctx, key)
	if result.IsOk() {
		data := result.Unwrap()
		if entity, err := r.serializer.Deserialize(data); err == nil {
			r.stats.Hits++
			return functional.Some(entity)
		}
	}

	r.stats.Misses++

	// Fallback to inner repository
	opt := r.inner.Get(ctx, id)
	if opt.IsSome() {
		entity := opt.Unwrap()
		if data, err := r.serializer.Serialize(entity); err == nil {
			r.cache.Set(ctx, key, data, r.ttl)
		}
	}
	return opt
}

// Save persists an entity and updates cache.
func (r *CachedRepository[T, ID]) Save(ctx context.Context, entity T) functional.Result[T] {
	result := r.inner.Save(ctx, entity)
	if result.IsOk() {
		saved := result.Unwrap()
		id := r.extractor(saved)
		key := r.buildKey(id)
		if data, err := r.serializer.Serialize(saved); err == nil {
			r.cache.Set(ctx, key, data, r.ttl)
		}
	}
	return result
}

// Delete removes an entity and invalidates cache.
func (r *CachedRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	key := r.buildKey(id)
	r.cache.Delete(ctx, key)
	return r.inner.Delete(ctx, id)
}

// List returns all entities (bypasses cache).
func (r *CachedRepository[T, ID]) List(ctx context.Context) functional.Result[[]T] {
	return r.inner.List(ctx)
}

// Exists checks if an entity exists.
func (r *CachedRepository[T, ID]) Exists(ctx context.Context, id ID) bool {
	key := r.buildKey(id)
	if r.cache.Get(ctx, key).IsOk() {
		return true
	}
	return r.inner.Exists(ctx, id)
}

// Invalidate removes an entry from cache.
func (r *CachedRepository[T, ID]) Invalidate(ctx context.Context, id ID) {
	key := r.buildKey(id)
	r.cache.Delete(ctx, key)
}

// Stats returns cache statistics.
func (r *CachedRepository[T, ID]) Stats() CacheStats {
	stats := r.stats
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}
	return stats
}

func (r *CachedRepository[T, ID]) buildKey(id ID) string {
	return fmt.Sprintf("%s:%v", r.keyPrefix, id)
}

// Ensure CachedRepository implements Repository.
var _ Repository[any, string] = (*CachedRepository[any, string])(nil)
