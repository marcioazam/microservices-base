// Package queue provides generic Queue and Stack data structures.
package queue

import (
	"github.com/auth-platform/libs/go/functional/option"
	"sync"
)

// Queue is a FIFO data structure.
type Queue[T any] struct {
	items []T
}

// NewQueue creates a new Queue.
func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{items: make([]T, 0)}
}

// Enqueue adds an item to the back.
func (q *Queue[T]) Enqueue(item T) {
	q.items = append(q.items, item)
}

// Dequeue removes and returns the front item.
func (q *Queue[T]) Dequeue() option.Option[T] {
	if len(q.items) == 0 {
		return option.None[T]()
	}
	item := q.items[0]
	q.items = q.items[1:]
	return option.Some(item)
}

// Peek returns the front item without removing.
func (q *Queue[T]) Peek() option.Option[T] {
	if len(q.items) == 0 {
		return option.None[T]()
	}
	return option.Some(q.items[0])
}

// Len returns the number of items.
func (q *Queue[T]) Len() int {
	return len(q.items)
}

// IsEmpty returns true if the queue is empty.
func (q *Queue[T]) IsEmpty() bool {
	return len(q.items) == 0
}

// Clear removes all items.
func (q *Queue[T]) Clear() {
	q.items = make([]T, 0)
}

// ToSlice returns all items as a slice.
func (q *Queue[T]) ToSlice() []T {
	result := make([]T, len(q.items))
	copy(result, q.items)
	return result
}

// Stack is a LIFO data structure.
type Stack[T any] struct {
	items []T
}

// NewStack creates a new Stack.
func NewStack[T any]() *Stack[T] {
	return &Stack[T]{items: make([]T, 0)}
}

// Push adds an item to the top.
func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

// Pop removes and returns the top item.
func (s *Stack[T]) Pop() option.Option[T] {
	if len(s.items) == 0 {
		return option.None[T]()
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return option.Some(item)
}

// Peek returns the top item without removing.
func (s *Stack[T]) Peek() option.Option[T] {
	if len(s.items) == 0 {
		return option.None[T]()
	}
	return option.Some(s.items[len(s.items)-1])
}

// Len returns the number of items.
func (s *Stack[T]) Len() int {
	return len(s.items)
}

// IsEmpty returns true if the stack is empty.
func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}

// Clear removes all items.
func (s *Stack[T]) Clear() {
	s.items = make([]T, 0)
}

// ToSlice returns all items as a slice (bottom to top).
func (s *Stack[T]) ToSlice() []T {
	result := make([]T, len(s.items))
	copy(result, s.items)
	return result
}

// ConcurrentQueue is a thread-safe FIFO queue.
type ConcurrentQueue[T any] struct {
	mu    sync.Mutex
	items []T
}

// NewConcurrentQueue creates a new ConcurrentQueue.
func NewConcurrentQueue[T any]() *ConcurrentQueue[T] {
	return &ConcurrentQueue[T]{items: make([]T, 0)}
}

// Enqueue adds an item to the back.
func (q *ConcurrentQueue[T]) Enqueue(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, item)
}

// Dequeue removes and returns the front item.
func (q *ConcurrentQueue[T]) Dequeue() option.Option[T] {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return option.None[T]()
	}
	item := q.items[0]
	q.items = q.items[1:]
	return option.Some(item)
}

// Peek returns the front item without removing.
func (q *ConcurrentQueue[T]) Peek() option.Option[T] {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return option.None[T]()
	}
	return option.Some(q.items[0])
}

// Len returns the number of items.
func (q *ConcurrentQueue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// IsEmpty returns true if the queue is empty.
func (q *ConcurrentQueue[T]) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items) == 0
}

// ConcurrentStack is a thread-safe LIFO stack.
type ConcurrentStack[T any] struct {
	mu    sync.Mutex
	items []T
}

// NewConcurrentStack creates a new ConcurrentStack.
func NewConcurrentStack[T any]() *ConcurrentStack[T] {
	return &ConcurrentStack[T]{items: make([]T, 0)}
}

// Push adds an item to the top.
func (s *ConcurrentStack[T]) Push(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
}

// Pop removes and returns the top item.
func (s *ConcurrentStack[T]) Pop() option.Option[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items) == 0 {
		return option.None[T]()
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return option.Some(item)
}

// Peek returns the top item without removing.
func (s *ConcurrentStack[T]) Peek() option.Option[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items) == 0 {
		return option.None[T]()
	}
	return option.Some(s.items[len(s.items)-1])
}

// Len returns the number of items.
func (s *ConcurrentStack[T]) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items)
}

// IsEmpty returns true if the stack is empty.
func (s *ConcurrentStack[T]) IsEmpty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items) == 0
}
