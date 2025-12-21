package patterns_test

import (
	"context"
	"testing"

	"github.com/authcorp/libs/go/src/functional"
	"github.com/authcorp/libs/go/src/patterns"
	"pgregory.net/rapid"
)

// Test entity for property tests
type TestEntity struct {
	ID   string
	Name string
	Age  int
}

func extractID(e TestEntity) string {
	return e.ID
}

// Property: InMemoryRepository Get returns None for non-existent ID
func TestInMemoryRepositoryGetNonExistent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := patterns.NewInMemoryRepository(extractID)
		id := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "id")

		result := repo.Get(context.Background(), id)

		if result.IsSome() {
			t.Errorf("Get(%s) should return None for non-existent ID", id)
		}
	})
}

// Property: InMemoryRepository Save then Get returns same entity
func TestInMemoryRepositorySaveGet(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := patterns.NewInMemoryRepository(extractID)
		id := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "id")
		name := rapid.StringMatching(`[A-Za-z]{3,20}`).Draw(t, "name")
		age := rapid.IntRange(0, 150).Draw(t, "age")

		entity := TestEntity{ID: id, Name: name, Age: age}

		// Save
		saveResult := repo.Save(context.Background(), entity)
		if saveResult.IsErr() {
			t.Fatalf("Save failed: %v", saveResult.UnwrapErr())
		}

		// Get
		getResult := repo.Get(context.Background(), id)
		if !getResult.IsSome() {
			t.Fatalf("Get(%s) should return Some after Save", id)
		}

		retrieved := getResult.Unwrap()
		if retrieved.ID != entity.ID {
			t.Errorf("ID = %s, want %s", retrieved.ID, entity.ID)
		}
		if retrieved.Name != entity.Name {
			t.Errorf("Name = %s, want %s", retrieved.Name, entity.Name)
		}
		if retrieved.Age != entity.Age {
			t.Errorf("Age = %d, want %d", retrieved.Age, entity.Age)
		}
	})
}

// Property: InMemoryRepository Delete removes entity
func TestInMemoryRepositoryDelete(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := patterns.NewInMemoryRepository(extractID)
		id := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "id")

		entity := TestEntity{ID: id, Name: "Test", Age: 25}

		// Save then Delete
		repo.Save(context.Background(), entity)
		err := repo.Delete(context.Background(), id)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Get should return None
		result := repo.Get(context.Background(), id)
		if result.IsSome() {
			t.Errorf("Get(%s) should return None after Delete", id)
		}
	})
}

// Property: InMemoryRepository Exists is consistent with Get
func TestInMemoryRepositoryExistsConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := patterns.NewInMemoryRepository(extractID)
		id := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "id")
		shouldExist := rapid.Bool().Draw(t, "should_exist")

		if shouldExist {
			entity := TestEntity{ID: id, Name: "Test", Age: 25}
			repo.Save(context.Background(), entity)
		}

		exists := repo.Exists(context.Background(), id)
		getResult := repo.Get(context.Background(), id)

		if exists != getResult.IsSome() {
			t.Errorf("Exists(%s) = %v, but Get.IsSome() = %v", id, exists, getResult.IsSome())
		}
	})
}

// Property: InMemoryRepository List returns all saved entities
func TestInMemoryRepositoryList(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := patterns.NewInMemoryRepository(extractID)
		count := rapid.IntRange(0, 20).Draw(t, "count")

		// Save entities
		for i := 0; i < count; i++ {
			id := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "id")
			entity := TestEntity{ID: id, Name: "Test", Age: i}
			repo.Save(context.Background(), entity)
		}

		// List
		result := repo.List(context.Background())
		if result.IsErr() {
			t.Fatalf("List failed: %v", result.UnwrapErr())
		}

		entities := result.Unwrap()
		if len(entities) != repo.Size() {
			t.Errorf("List returned %d entities, but Size() = %d", len(entities), repo.Size())
		}
	})
}

// Property: InMemoryRepository Clear removes all entities
func TestInMemoryRepositoryClear(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := patterns.NewInMemoryRepository(extractID)
		count := rapid.IntRange(1, 20).Draw(t, "count")

		// Save entities
		for i := 0; i < count; i++ {
			id := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "id")
			entity := TestEntity{ID: id, Name: "Test", Age: i}
			repo.Save(context.Background(), entity)
		}

		// Clear
		repo.Clear()

		if repo.Size() != 0 {
			t.Errorf("Size() = %d after Clear, want 0", repo.Size())
		}
	})
}

// Property: InMemoryRepository Size is consistent
func TestInMemoryRepositorySizeConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := patterns.NewInMemoryRepository(extractID)
		operations := rapid.IntRange(1, 50).Draw(t, "operations")

		expectedSize := 0
		ids := make(map[string]bool)

		for i := 0; i < operations; i++ {
			op := rapid.IntRange(0, 2).Draw(t, "op")
			id := rapid.StringMatching(`[a-z0-9]{4}`).Draw(t, "id")

			switch op {
			case 0: // Save
				entity := TestEntity{ID: id, Name: "Test", Age: i}
				repo.Save(context.Background(), entity)
				if !ids[id] {
					expectedSize++
					ids[id] = true
				}
			case 1: // Delete
				repo.Delete(context.Background(), id)
				if ids[id] {
					expectedSize--
					delete(ids, id)
				}
			}
		}

		if repo.Size() != expectedSize {
			t.Errorf("Size() = %d, want %d", repo.Size(), expectedSize)
		}
	})
}

// Property: Page HasNext/HasPrev are consistent
func TestPageNavigation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		pageSize := rapid.IntRange(1, 100).Draw(t, "page_size")
		totalItems := rapid.Int64Range(0, 1000).Draw(t, "total_items")
		page := rapid.IntRange(1, 20).Draw(t, "page")

		items := make([]TestEntity, 0)
		p := patterns.NewPage(items, page, pageSize, totalItems)

		// HasPrev should be true if page > 1
		if p.HasPrev() != (page > 1) {
			t.Errorf("HasPrev() = %v, want %v (page=%d)", p.HasPrev(), page > 1, page)
		}

		// HasNext should be true if page < totalPages
		if p.HasNext() != (page < p.TotalPages) {
			t.Errorf("HasNext() = %v, want %v (page=%d, totalPages=%d)", p.HasNext(), page < p.TotalPages, page, p.TotalPages)
		}
	})
}

// Property: Page TotalPages calculation is correct
func TestPageTotalPagesCalculation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		pageSize := rapid.IntRange(1, 100).Draw(t, "page_size")
		totalItems := rapid.Int64Range(0, 1000).Draw(t, "total_items")

		items := make([]TestEntity, 0)
		p := patterns.NewPage(items, 1, pageSize, totalItems)

		expectedTotalPages := int(totalItems) / pageSize
		if int(totalItems)%pageSize > 0 {
			expectedTotalPages++
		}

		if p.TotalPages != expectedTotalPages {
			t.Errorf("TotalPages = %d, want %d (totalItems=%d, pageSize=%d)",
				p.TotalPages, expectedTotalPages, totalItems, pageSize)
		}
	})
}

// Property: Page IsEmpty is consistent with Items length
func TestPageIsEmpty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		itemCount := rapid.IntRange(0, 10).Draw(t, "item_count")

		items := make([]TestEntity, itemCount)
		p := patterns.NewPage(items, 1, 10, int64(itemCount))

		if p.IsEmpty() != (len(items) == 0) {
			t.Errorf("IsEmpty() = %v, but len(Items) = %d", p.IsEmpty(), len(items))
		}
	})
}

// Mock cache for testing CachedRepository
type mockCache[K comparable, V any] struct {
	data map[K]V
}

func newMockCache[K comparable, V any]() *mockCache[K, V] {
	return &mockCache[K, V]{data: make(map[K]V)}
}

func (c *mockCache[K, V]) Get(key K) functional.Option[V] {
	if v, ok := c.data[key]; ok {
		return functional.Some(v)
	}
	return functional.None[V]()
}

func (c *mockCache[K, V]) Put(key K, value V) {
	c.data[key] = value
}

func (c *mockCache[K, V]) Remove(key K) bool {
	_, ok := c.data[key]
	delete(c.data, key)
	return ok
}

func (c *mockCache[K, V]) Clear() {
	c.data = make(map[K]V)
}

func (c *mockCache[K, V]) Size() int {
	return len(c.data)
}

// Property: CachedRepository Get uses cache
func TestCachedRepositoryUsesCache(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		inner := patterns.NewInMemoryRepository(extractID)
		cache := newMockCache[string, TestEntity]()
		cached := patterns.NewCachedRepository(inner, cache, extractID)

		id := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "id")
		entity := TestEntity{ID: id, Name: "Test", Age: 25}

		// Save to inner
		inner.Save(context.Background(), entity)

		// First Get should populate cache
		result1 := cached.Get(context.Background(), id)
		if !result1.IsSome() {
			t.Fatalf("First Get should return Some")
		}

		// Cache should now have the entity
		if cache.Size() != 1 {
			t.Errorf("Cache size = %d, want 1", cache.Size())
		}

		// Second Get should use cache
		result2 := cached.Get(context.Background(), id)
		if !result2.IsSome() {
			t.Fatalf("Second Get should return Some")
		}
	})
}

// Property: CachedRepository Save updates cache
func TestCachedRepositorySaveUpdatesCache(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		inner := patterns.NewInMemoryRepository(extractID)
		cache := newMockCache[string, TestEntity]()
		cached := patterns.NewCachedRepository(inner, cache, extractID)

		id := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "id")
		entity := TestEntity{ID: id, Name: "Test", Age: 25}

		// Save through cached repository
		cached.Save(context.Background(), entity)

		// Cache should have the entity
		cacheResult := cache.Get(id)
		if !cacheResult.IsSome() {
			t.Error("Cache should have entity after Save")
		}
	})
}

// Property: CachedRepository Delete invalidates cache
func TestCachedRepositoryDeleteInvalidatesCache(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		inner := patterns.NewInMemoryRepository(extractID)
		cache := newMockCache[string, TestEntity]()
		cached := patterns.NewCachedRepository(inner, cache, extractID)

		id := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "id")
		entity := TestEntity{ID: id, Name: "Test", Age: 25}

		// Save then Delete
		cached.Save(context.Background(), entity)
		cached.Delete(context.Background(), id)

		// Cache should not have the entity
		cacheResult := cache.Get(id)
		if cacheResult.IsSome() {
			t.Error("Cache should not have entity after Delete")
		}
	})
}
