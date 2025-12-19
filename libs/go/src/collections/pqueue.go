package collections

import (
	"container/heap"
	"sync"

	"github.com/authcorp/libs/go/src/functional"
)

// PriorityQueue is a thread-safe priority queue.
type PriorityQueue[T any] struct {
	items    *pqHeap[T]
	less     func(a, b T) bool
	mu       sync.RWMutex
}

type pqHeap[T any] struct {
	items []T
	less  func(a, b T) bool
}

func (h *pqHeap[T]) Len() int           { return len(h.items) }
func (h *pqHeap[T]) Less(i, j int) bool { return h.less(h.items[i], h.items[j]) }
func (h *pqHeap[T]) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }

func (h *pqHeap[T]) Push(x any) {
	h.items = append(h.items, x.(T))
}

func (h *pqHeap[T]) Pop() any {
	old := h.items
	n := len(old)
	x := old[n-1]
	h.items = old[0 : n-1]
	return x
}

// NewPriorityQueue creates a new priority queue.
func NewPriorityQueue[T any](less func(a, b T) bool) *PriorityQueue[T] {
	h := &pqHeap[T]{
		items: make([]T, 0),
		less:  less,
	}
	heap.Init(h)
	return &PriorityQueue[T]{
		items: h,
		less:  less,
	}
}

// Push adds an item to the queue.
func (pq *PriorityQueue[T]) Push(item T) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	heap.Push(pq.items, item)
}

// Pop removes and returns the highest priority item.
func (pq *PriorityQueue[T]) Pop() functional.Option[T] {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if pq.items.Len() == 0 {
		return functional.None[T]()
	}
	item := heap.Pop(pq.items).(T)
	return functional.Some(item)
}

// Peek returns the highest priority item without removing.
func (pq *PriorityQueue[T]) Peek() functional.Option[T] {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	if pq.items.Len() == 0 {
		return functional.None[T]()
	}
	return functional.Some(pq.items.items[0])
}

// Size returns the number of items.
func (pq *PriorityQueue[T]) Size() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return pq.items.Len()
}

// IsEmpty returns true if queue is empty.
func (pq *PriorityQueue[T]) IsEmpty() bool {
	return pq.Size() == 0
}

// Clear removes all items.
func (pq *PriorityQueue[T]) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.items.items = make([]T, 0)
}

// ToSlice returns items as a slice (not in priority order).
func (pq *PriorityQueue[T]) ToSlice() []T {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	result := make([]T, len(pq.items.items))
	copy(result, pq.items.items)
	return result
}
