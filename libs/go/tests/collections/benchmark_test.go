package collections_test

import (
	"testing"

	"github.com/authcorp/libs/go/src/collections"
)

func BenchmarkLRUCache_Get(b *testing.B) {
	cache := collections.NewLRUCache[int, string](1000)
	for i := 0; i < 1000; i++ {
		cache.Put(i, "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(i % 1000)
	}
}

func BenchmarkLRUCache_Put(b *testing.B) {
	cache := collections.NewLRUCache[int, string](1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put(i%1000, "value")
	}
}

func BenchmarkLRUCache_GetOrCompute(b *testing.B) {
	cache := collections.NewLRUCache[int, string](1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetOrCompute(i%1000, func() string {
			return "computed"
		})
	}
}

func BenchmarkSet_Add(b *testing.B) {
	set := collections.NewSet[int]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Add(i)
	}
}

func BenchmarkSet_Contains(b *testing.B) {
	set := collections.NewSet[int]()
	for i := 0; i < 1000; i++ {
		set.Add(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Contains(i % 1000)
	}
}

func BenchmarkIterator_Map(b *testing.B) {
	slice := make([]int, 1000)
	for i := range slice {
		slice[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter := collections.FromSlice(slice)
		mapped := collections.Map(iter, func(x int) int { return x * 2 })
		collections.Collect(mapped)
	}
}

func BenchmarkIterator_Filter(b *testing.B) {
	slice := make([]int, 1000)
	for i := range slice {
		slice[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter := collections.FromSlice(slice)
		filtered := collections.Filter(iter, func(x int) bool { return x%2 == 0 })
		collections.Collect(filtered)
	}
}

func BenchmarkQueue_EnqueueDequeue(b *testing.B) {
	queue := collections.NewQueue[int]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.Enqueue(i)
		queue.Dequeue()
	}
}

func BenchmarkPriorityQueue_PushPop(b *testing.B) {
	pq := collections.NewPriorityQueue(func(a, b int) bool { return a < b })

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pq.Push(i)
		pq.Pop()
	}
}
