package collections

import (
	"iter"
	"sync"

	"github.com/authcorp/libs/go/src/functional"
)

// Queue is a thread-safe FIFO queue.
type Queue[T any] struct {
	items []T
	mu    sync.RWMutex
}

// NewQueue creates a new empty queue.
func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{items: make([]T, 0)}
}

// Enqueue adds an item to the back.
func (q *Queue[T]) Enqueue(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, item)
}

// Dequeue removes and returns the front item.
func (q *Queue[T]) Dequeue() functional.Option[T] {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return functional.None[T]()
	}
	item := q.items[0]
	q.items = q.items[1:]
	return functional.Some(item)
}

// Peek returns the front item without removing.
func (q *Queue[T]) Peek() functional.Option[T] {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if len(q.items) == 0 {
		return functional.None[T]()
	}
	return functional.Some(q.items[0])
}

// Size returns the number of items.
func (q *Queue[T]) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items)
}

// IsEmpty returns true if queue is empty.
func (q *Queue[T]) IsEmpty() bool {
	return q.Size() == 0
}

// Clear removes all items.
func (q *Queue[T]) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = make([]T, 0)
}

// ToSlice returns items as a slice.
func (q *Queue[T]) ToSlice() []T {
	q.mu.RLock()
	defer q.mu.RUnlock()
	result := make([]T, len(q.items))
	copy(result, q.items)
	return result
}

// All returns a Go 1.23+ iterator over queue elements (FIFO order).
func (q *Queue[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		q.mu.RLock()
		defer q.mu.RUnlock()
		for _, item := range q.items {
			if !yield(item) {
				return
			}
		}
	}
}

// Collect returns all elements as a slice.
func (q *Queue[T]) Collect() []T {
	return q.ToSlice()
}

// Stack is a thread-safe LIFO stack.
type Stack[T any] struct {
	items []T
	mu    sync.RWMutex
}

// NewStack creates a new empty stack.
func NewStack[T any]() *Stack[T] {
	return &Stack[T]{items: make([]T, 0)}
}

// Push adds an item to the top.
func (s *Stack[T]) Push(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
}

// Pop removes and returns the top item.
func (s *Stack[T]) Pop() functional.Option[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items) == 0 {
		return functional.None[T]()
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return functional.Some(item)
}

// Peek returns the top item without removing.
func (s *Stack[T]) Peek() functional.Option[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.items) == 0 {
		return functional.None[T]()
	}
	return functional.Some(s.items[len(s.items)-1])
}

// Size returns the number of items.
func (s *Stack[T]) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// IsEmpty returns true if stack is empty.
func (s *Stack[T]) IsEmpty() bool {
	return s.Size() == 0
}

// Clear removes all items.
func (s *Stack[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make([]T, 0)
}
