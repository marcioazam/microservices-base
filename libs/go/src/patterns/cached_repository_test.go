package patterns

import (
	"context"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/functional"
	"pgregory.net/rapid"
)

// mockCacheClient implements CacheClient for testing.
type mockCacheClient struct {
	data map[string][]byte
}

func newMockCacheClient() *mockCacheClient {
	return &mockCacheClient{data: make(map[string][]byte)}
}

func (m *mockCacheClient) Get(ctx context.Context, key string) functional.Result[[]byte] {
	if data, ok := m.data[key]; ok {
		return functional.Ok(data)
	}
	return functional.Err[[]byte](nil)
}

func (m *mockCacheClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCacheClient) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

// mockRepository implements Repository for testing.
type mockRepository[T any, ID comparable] struct {
	data      map[ID]T
	extractor IDExtractor[T, ID]
}

func newMockRepository[T any, ID comparable](extractor IDExtractor[T, ID]) *mockRepository[T, ID] {
	return &mockRepository[T, ID]{
		data:      make(map[ID]T),
		extractor: extractor,
	}
}

func (m *mockRepository[T, ID]) Get(ctx context.Context, id ID) functional.Option[T] {
	if entity, ok := m.data[id]; ok {
		return functional.Some(entity)
	}
	return functional.None[T]()
}

func (m *mockRepository[T, ID]) Save(ctx context.Context, entity T) functional.Result[T] {
	id := m.extractor(entity)
	m.data[id] = entity
	return functional.Ok(entity)
}

func (m *mockRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	delete(m.data, id)
	return nil
}

func (m *mockRepository[T, ID]) List(ctx context.Context) functional.Result[[]T] {
	result := make([]T, 0, len(m.data))
	for _, v := range m.data {
		result = append(result, v)
	}
	return functional.Ok(result)
}

func (m *mockRepository[T, ID]) Exists(ctx context.Context, id ID) bool {
	_, ok := m.data[id]
	return ok
}

// TestEntity for testing.
type TestEntity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Property test: Get after Save returns same value
// Validates: Requirements 3.2
func TestProperty_CachedRepository_GetAfterSave(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "id")
		name := rapid.StringMatching(`[a-zA-Z ]{1,50}`).Draw(t, "name")

		entity := TestEntity{ID: id, Name: name}

		extractor := func(e TestEntity) string { return e.ID }
		innerRepo := newMockRepository(extractor)
		cache := newMockCacheClient()

		repo := NewCachedRepositoryWithJSON(
			innerRepo,
			cache,
			extractor,
			CachedRepositoryConfig{KeyPrefix: "test", TTL: 5 * time.Minute},
		)

		ctx := context.Background()

		// Save entity
		result := repo.Save(ctx, entity)
		if !result.IsOk() {
			t.Fatalf("Save failed: %v", result.UnwrapErr())
		}

		// Get entity
		opt := repo.Get(ctx, id)
		if !opt.IsSome() {
			t.Fatal("Get returned None after Save")
		}

		retrieved := opt.Unwrap()
		if retrieved.ID != entity.ID || retrieved.Name != entity.Name {
			t.Errorf("Entity mismatch: got %+v, want %+v", retrieved, entity)
		}
	})
}

// Unit test for cache hit
func TestCachedRepository_CacheHit(t *testing.T) {
	extractor := func(e TestEntity) string { return e.ID }
	innerRepo := newMockRepository(extractor)
	cache := newMockCacheClient()

	repo := NewCachedRepositoryWithJSON(
		innerRepo,
		cache,
		extractor,
		CachedRepositoryConfig{KeyPrefix: "test", TTL: 5 * time.Minute},
	)

	ctx := context.Background()
	entity := TestEntity{ID: "123", Name: "Test"}

	// Save to populate cache
	repo.Save(ctx, entity)

	// Clear inner repo to verify cache hit
	delete(innerRepo.data, "123")

	// Get should hit cache
	opt := repo.Get(ctx, "123")
	if !opt.IsSome() {
		t.Fatal("Expected cache hit")
	}

	stats := repo.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
}

// Unit test for cache miss
func TestCachedRepository_CacheMiss(t *testing.T) {
	extractor := func(e TestEntity) string { return e.ID }
	innerRepo := newMockRepository(extractor)
	cache := newMockCacheClient()

	repo := NewCachedRepositoryWithJSON(
		innerRepo,
		cache,
		extractor,
		CachedRepositoryConfig{KeyPrefix: "test", TTL: 5 * time.Minute},
	)

	ctx := context.Background()

	// Add directly to inner repo (bypassing cache)
	innerRepo.data["123"] = TestEntity{ID: "123", Name: "Test"}

	// Get should miss cache but find in repo
	opt := repo.Get(ctx, "123")
	if !opt.IsSome() {
		t.Fatal("Expected to find entity")
	}

	stats := repo.Stats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
}

// Unit test for delete
func TestCachedRepository_Delete(t *testing.T) {
	extractor := func(e TestEntity) string { return e.ID }
	innerRepo := newMockRepository(extractor)
	cache := newMockCacheClient()

	repo := NewCachedRepositoryWithJSON(
		innerRepo,
		cache,
		extractor,
		CachedRepositoryConfig{KeyPrefix: "test", TTL: 5 * time.Minute},
	)

	ctx := context.Background()
	entity := TestEntity{ID: "123", Name: "Test"}

	// Save
	repo.Save(ctx, entity)

	// Delete
	err := repo.Delete(ctx, "123")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Get should return None
	opt := repo.Get(ctx, "123")
	if opt.IsSome() {
		t.Error("Expected None after delete")
	}
}
