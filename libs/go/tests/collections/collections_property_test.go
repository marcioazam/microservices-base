package collections_test

import (
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/collections"
	"pgregory.net/rapid"
)

// Property 6: Collection Map Identity
// Mapping with identity function preserves all elements.
func TestProperty_CollectionMapIdentity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		slice := rapid.SliceOf(rapid.Int()).Draw(t, "slice")
		
		iter := collections.FromSlice(slice)
		mapped := collections.Map(iter, func(x int) int { return x })
		collected := collections.Collect(mapped)
		
		if len(collected) != len(slice) {
			t.Fatalf("Length mismatch: expected %d, got %d", len(slice), len(collected))
		}
		for i, v := range collected {
			if v != slice[i] {
				t.Fatalf("Value mismatch at %d: expected %d, got %d", i, slice[i], v)
			}
		}
	})
}

// Property 7: Collection IsEmpty Invariant
// IsEmpty is true iff Size is 0.
func TestProperty_CollectionIsEmptyInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		items := rapid.SliceOf(rapid.Int()).Draw(t, "items")
		
		set := collections.SetFrom(items)
		isEmpty := set.IsEmpty()
		size := set.Size()
		
		if isEmpty != (size == 0) {
			t.Fatalf("IsEmpty invariant violated: IsEmpty=%v, Size=%d", isEmpty, size)
		}
	})
}

// Property 8: Collection Contains After Add
// After Add(x), Contains(x) is true.
func TestProperty_CollectionContainsAfterAdd(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		item := rapid.Int().Draw(t, "item")
		
		set := collections.NewSet[int]()
		set.Add(item)
		
		if !set.Contains(item) {
			t.Fatalf("Contains should be true after Add")
		}
	})
}

// Property 13: TTL Cache Expiry
// Items expire after TTL.
func TestProperty_TTLCacheExpiry(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.Int().Draw(t, "key")
		value := rapid.String().Draw(t, "value")
		
		cache := collections.NewLRUCache[int, string](10).WithTTL(time.Millisecond * 10)
		cache.Put(key, value)
		
		// Should exist immediately
		if opt := cache.Get(key); !opt.IsSome() {
			t.Fatalf("Item should exist immediately after Put")
		}
		
		// Wait for expiry
		time.Sleep(time.Millisecond * 20)
		
		// Should be expired
		if opt := cache.Get(key); opt.IsSome() {
			t.Fatalf("Item should be expired after TTL")
		}
	})
}

// Property 14: LRU Cache Eviction Order
// Least recently used items are evicted first.
func TestProperty_LRUCacheEvictionOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(2, 10).Draw(t, "capacity")
		
		var evicted []int
		cache := collections.NewLRUCache[int, string](capacity).
			WithEvictCallback(func(k int, _ string) {
				evicted = append(evicted, k)
			})
		
		// Fill cache
		for i := 0; i < capacity; i++ {
			cache.Put(i, "value")
		}
		
		// Access first item to make it recently used
		cache.Get(0)
		
		// Add one more to trigger eviction
		cache.Put(capacity, "new")
		
		// Item 1 should be evicted (least recently used after 0 was accessed)
		if len(evicted) != 1 {
			t.Fatalf("Expected 1 eviction, got %d", len(evicted))
		}
		if evicted[0] != 1 {
			t.Fatalf("Expected item 1 to be evicted, got %d", evicted[0])
		}
	})
}

// Property 15: Cache GetOrCompute Behavior
// GetOrCompute returns cached value or computes new one.
func TestProperty_CacheGetOrCompute(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.Int().Draw(t, "key")
		value := rapid.String().Draw(t, "value")
		
		cache := collections.NewLRUCache[int, string](10)
		computeCount := 0
		
		compute := func() string {
			computeCount++
			return value
		}
		
		// First call should compute
		result1 := cache.GetOrCompute(key, compute)
		if result1 != value {
			t.Fatalf("Expected %s, got %s", value, result1)
		}
		if computeCount != 1 {
			t.Fatalf("Expected 1 compute, got %d", computeCount)
		}
		
		// Second call should use cache
		result2 := cache.GetOrCompute(key, compute)
		if result2 != value {
			t.Fatalf("Expected %s, got %s", value, result2)
		}
		if computeCount != 1 {
			t.Fatalf("Expected still 1 compute, got %d", computeCount)
		}
	})
}

// Property 16: Cache Eviction Callback
// Eviction callback is called with correct key/value.
func TestProperty_CacheEvictionCallback(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.Int().Draw(t, "key")
		value := rapid.String().Draw(t, "value")
		
		var evictedKey int
		var evictedValue string
		
		cache := collections.NewLRUCache[int, string](1).
			WithEvictCallback(func(k int, v string) {
				evictedKey = k
				evictedValue = v
			})
		
		cache.Put(key, value)
		cache.Put(key+1, "other") // Triggers eviction
		
		if evictedKey != key {
			t.Fatalf("Expected evicted key %d, got %d", key, evictedKey)
		}
		if evictedValue != value {
			t.Fatalf("Expected evicted value %s, got %s", value, evictedValue)
		}
	})
}

// Property: Set Union contains all elements
func TestProperty_SetUnion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		items1 := rapid.SliceOf(rapid.Int()).Draw(t, "items1")
		items2 := rapid.SliceOf(rapid.Int()).Draw(t, "items2")
		
		set1 := collections.SetFrom(items1)
		set2 := collections.SetFrom(items2)
		union := set1.Union(set2)
		
		for _, item := range items1 {
			if !union.Contains(item) {
				t.Fatalf("Union should contain all items from set1")
			}
		}
		for _, item := range items2 {
			if !union.Contains(item) {
				t.Fatalf("Union should contain all items from set2")
			}
		}
	})
}

// Property: Set Intersection contains only common elements
func TestProperty_SetIntersection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		items1 := rapid.SliceOf(rapid.Int()).Draw(t, "items1")
		items2 := rapid.SliceOf(rapid.Int()).Draw(t, "items2")
		
		set1 := collections.SetFrom(items1)
		set2 := collections.SetFrom(items2)
		intersection := set1.Intersection(set2)
		
		for _, item := range intersection.ToSlice() {
			if !set1.Contains(item) || !set2.Contains(item) {
				t.Fatalf("Intersection should only contain common elements")
			}
		}
	})
}

// Property: Queue FIFO order
func TestProperty_QueueFIFO(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		items := rapid.SliceOfN(rapid.Int(), 1, 20).Draw(t, "items")
		
		queue := collections.NewQueue[int]()
		for _, item := range items {
			queue.Enqueue(item)
		}
		
		for _, expected := range items {
			opt := queue.Dequeue()
			if !opt.IsSome() {
				t.Fatalf("Queue should not be empty")
			}
			if opt.Unwrap() != expected {
				t.Fatalf("FIFO order violated: expected %d, got %d", expected, opt.Unwrap())
			}
		}
	})
}

// Property: Stack LIFO order
func TestProperty_StackLIFO(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		items := rapid.SliceOfN(rapid.Int(), 1, 20).Draw(t, "items")
		
		stack := collections.NewStack[int]()
		for _, item := range items {
			stack.Push(item)
		}
		
		// Reverse order for LIFO
		for i := len(items) - 1; i >= 0; i-- {
			opt := stack.Pop()
			if !opt.IsSome() {
				t.Fatalf("Stack should not be empty")
			}
			if opt.Unwrap() != items[i] {
				t.Fatalf("LIFO order violated: expected %d, got %d", items[i], opt.Unwrap())
			}
		}
	})
}
