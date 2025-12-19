// Package pqueue provides a generic priority queue.
package pqueue

import (
	"container/heap"

	"github.com/auth-platform/libs/go/functional/option"
)

// PriorityQueue is a generic priority queue.
type PriorityQueue[T any] struct {
	items *heapItems[T]
	less  func(a, b T) bool
}

// New creates a new PriorityQueue with a comparison function.
// less(a, b) should return true if a has higher priority than b.
func New[T any](less func(a, b T) bool) *PriorityQueue[T] {
	h := &heapItems[T]{less: less}
	heap.Init(h)
	return &PriorityQueue[T]{items: h, less: less}
}

// Push adds an item to the queue.
func (pq *PriorityQueue[T]) Push(item T) {
	heap.Push(pq.items, item)
}

// Pop removes and returns the highest priority item.
func (pq *PriorityQueue[T]) Pop() option.Option[T] {
	if pq.items.Len() == 0 {
		return option.None[T]()
	}
	item := heap.Pop(pq.items).(T)
	return option.Some(item)
}

// Peek returns the highest priority item without removing.
func (pq *PriorityQueue[T]) Peek() option.Option[T] {
	if pq.items.Len() == 0 {
		return option.None[T]()
	}
	return option.Some(pq.items.items[0])
}

// Len returns the number of items.
func (pq *PriorityQueue[T]) Len() int {
	return pq.items.Len()
}

// IsEmpty returns true if the queue is empty.
func (pq *PriorityQueue[T]) IsEmpty() bool {
	return pq.items.Len() == 0
}

// Clear removes all items.
func (pq *PriorityQueue[T]) Clear() {
	pq.items.items = make([]T, 0)
}

// heapItems implements heap.Interface.
type heapItems[T any] struct {
	items []T
	less  func(a, b T) bool
}

func (h *heapItems[T]) Len() int           { return len(h.items) }
func (h *heapItems[T]) Less(i, j int) bool { return h.less(h.items[i], h.items[j]) }
func (h *heapItems[T]) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }

func (h *heapItems[T]) Push(x interface{}) {
	h.items = append(h.items, x.(T))
}

func (h *heapItems[T]) Pop() interface{} {
	old := h.items
	n := len(old)
	item := old[n-1]
	h.items = old[0 : n-1]
	return item
}

// MinHeap creates a min-heap priority queue for ordered types.
func MinHeap[T interface{ ~int | ~int64 | ~float64 | ~string }]() *PriorityQueue[T] {
	return New(func(a, b T) bool { return a < b })
}

// MaxHeap creates a max-heap priority queue for ordered types.
func MaxHeap[T interface{ ~int | ~int64 | ~float64 | ~string }]() *PriorityQueue[T] {
	return New(func(a, b T) bool { return a > b })
}

// PriorityItem wraps a value with a priority.
type PriorityItem[T any] struct {
	Value    T
	Priority int
}

// NewWithPriority creates a priority queue using PriorityItem.
func NewWithPriority[T any]() *PriorityQueue[PriorityItem[T]] {
	return New(func(a, b PriorityItem[T]) bool {
		return a.Priority > b.Priority // Higher priority first
	})
}
