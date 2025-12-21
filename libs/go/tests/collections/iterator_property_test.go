// Feature: go-libs-state-of-art-2025, Property 7: Iterator Correctness
// Validates: Requirements 5.3, 8.5
package collections_test

import (
	"testing"

	"github.com/authcorp/libs/go/src/collections"
	"github.com/authcorp/libs/go/src/functional"
	"pgregory.net/rapid"
)

// TestSetIteratorYieldsAllElements verifies Set.All() yields exactly n elements.
func TestSetIteratorYieldsAllElements(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numElements := rapid.IntRange(0, 50).Draw(t, "numElements")
		set := collections.NewSet[int]()

		for i := 0; i < numElements; i++ {
			set.Add(i)
		}

		// Count elements from iterator
		count := 0
		for range set.All() {
			count++
		}

		if count != set.Size() {
			t.Fatalf("iterator yielded %d elements, set has %d", count, set.Size())
		}
	})
}

// TestSetIteratorCollect verifies Collect returns all elements.
func TestSetIteratorCollect(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		elements := rapid.SliceOfN(rapid.Int(), 0, 30).Draw(t, "elements")
		set := collections.NewSet[int]()

		for _, e := range elements {
			set.Add(e)
		}

		collected := set.Collect()

		if len(collected) != set.Size() {
			t.Fatalf("Collect returned %d elements, set has %d", len(collected), set.Size())
		}

		// Verify all collected elements are in set
		for _, e := range collected {
			if !set.Contains(e) {
				t.Fatalf("collected element %d not in set", e)
			}
		}
	})
}

// TestQueueIteratorFIFOOrder verifies Queue.All() yields elements in FIFO order.
func TestQueueIteratorFIFOOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numElements := rapid.IntRange(1, 50).Draw(t, "numElements")
		queue := collections.NewQueue[int]()

		for i := 0; i < numElements; i++ {
			queue.Enqueue(i)
		}

		// Verify FIFO order
		expected := 0
		for v := range queue.All() {
			if v != expected {
				t.Fatalf("expected %d at position %d, got %d", expected, expected, v)
			}
			expected++
		}

		if expected != numElements {
			t.Fatalf("iterator yielded %d elements, expected %d", expected, numElements)
		}
	})
}

// TestQueueIteratorCollect verifies Collect returns elements in FIFO order.
func TestQueueIteratorCollect(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		elements := rapid.SliceOfN(rapid.Int(), 0, 30).Draw(t, "elements")
		queue := collections.NewQueue[int]()

		for _, e := range elements {
			queue.Enqueue(e)
		}

		collected := queue.Collect()

		if len(collected) != len(elements) {
			t.Fatalf("Collect returned %d elements, expected %d", len(collected), len(elements))
		}

		// Verify order matches
		for i, e := range elements {
			if collected[i] != e {
				t.Fatalf("element at %d: got %d, expected %d", i, collected[i], e)
			}
		}
	})
}

// TestPriorityQueueIteratorPriorityOrder verifies PriorityQueue.All() yields in priority order.
func TestPriorityQueueIteratorPriorityOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		elements := rapid.SliceOfN(rapid.IntRange(0, 100), 1, 20).Draw(t, "elements")

		// Min-heap: smaller values have higher priority
		pq := collections.NewPriorityQueue[int](func(a, b int) bool {
			return a < b
		})

		for _, e := range elements {
			pq.Push(e)
		}

		// Verify priority order (ascending for min-heap)
		collected := pq.Collect()
		for i := 1; i < len(collected); i++ {
			if collected[i] < collected[i-1] {
				t.Fatalf("priority order violated: %d < %d at position %d", collected[i], collected[i-1], i)
			}
		}
	})
}

// TestLRUCacheIteratorYieldsAllEntries verifies LRUCache.All() yields all entries.
func TestLRUCacheIteratorYieldsAllEntries(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		capacity := rapid.IntRange(5, 30).Draw(t, "capacity")
		numPuts := rapid.IntRange(1, capacity).Draw(t, "numPuts")

		cache := collections.NewLRUCache[int, string](capacity)
		expected := make(map[int]string)

		for i := 0; i < numPuts; i++ {
			key := i
			value := rapid.String().Draw(t, "value")
			cache.Put(key, value)
			expected[key] = value
		}

		// Collect via iterator
		collected := make(map[int]string)
		for k, v := range cache.All() {
			collected[k] = v
		}

		if len(collected) != len(expected) {
			t.Fatalf("iterator yielded %d entries, expected %d", len(collected), len(expected))
		}

		for k, v := range expected {
			if collected[k] != v {
				t.Fatalf("entry %d: got %q, expected %q", k, collected[k], v)
			}
		}
	})
}

// TestOptionIteratorYieldsCorrectly verifies Option.All() yields 0 or 1 element.
func TestOptionIteratorYieldsCorrectly(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hasSome := rapid.Bool().Draw(t, "hasSome")
		value := rapid.Int().Draw(t, "value")

		var opt functional.Option[int]
		if hasSome {
			opt = functional.Some(value)
		} else {
			opt = functional.None[int]()
		}

		count := 0
		var yielded int
		for v := range opt.All() {
			count++
			yielded = v
		}

		if hasSome {
			if count != 1 {
				t.Fatalf("Some should yield 1 element, got %d", count)
			}
			if yielded != value {
				t.Fatalf("yielded %d, expected %d", yielded, value)
			}
		} else {
			if count != 0 {
				t.Fatalf("None should yield 0 elements, got %d", count)
			}
		}
	})
}

// TestResultIteratorYieldsCorrectly verifies Result.All() yields 0 or 1 element.
func TestResultIteratorYieldsCorrectly(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		isOk := rapid.Bool().Draw(t, "isOk")
		value := rapid.Int().Draw(t, "value")

		var result functional.Result[int]
		if isOk {
			result = functional.Ok(value)
		} else {
			result = functional.Err[int](functional.NewError("test error"))
		}

		count := 0
		var yielded int
		for v := range result.All() {
			count++
			yielded = v
		}

		if isOk {
			if count != 1 {
				t.Fatalf("Ok should yield 1 element, got %d", count)
			}
			if yielded != value {
				t.Fatalf("yielded %d, expected %d", yielded, value)
			}
		} else {
			if count != 0 {
				t.Fatalf("Err should yield 0 elements, got %d", count)
			}
		}
	})
}

// TestIteratorReusability verifies iterators can be called multiple times.
func TestIteratorReusability(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numElements := rapid.IntRange(1, 20).Draw(t, "numElements")
		set := collections.NewSet[int]()

		for i := 0; i < numElements; i++ {
			set.Add(i)
		}

		// First iteration
		count1 := 0
		for range set.All() {
			count1++
		}

		// Second iteration
		count2 := 0
		for range set.All() {
			count2++
		}

		if count1 != count2 {
			t.Fatalf("iterator not reusable: first=%d, second=%d", count1, count2)
		}
	})
}

// TestIteratorEarlyTermination verifies iterators handle early termination.
func TestIteratorEarlyTermination(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numElements := rapid.IntRange(5, 30).Draw(t, "numElements")
		stopAt := rapid.IntRange(1, numElements-1).Draw(t, "stopAt")

		queue := collections.NewQueue[int]()
		for i := 0; i < numElements; i++ {
			queue.Enqueue(i)
		}

		count := 0
		for range queue.All() {
			count++
			if count >= stopAt {
				break
			}
		}

		if count != stopAt {
			t.Fatalf("early termination failed: stopped at %d, expected %d", count, stopAt)
		}

		// Queue should still be intact
		if queue.Size() != numElements {
			t.Fatalf("queue modified during iteration: size=%d, expected=%d", queue.Size(), numElements)
		}
	})
}
