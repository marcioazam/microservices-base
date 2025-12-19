package collections

import "sync"

// Set is a generic set implementation.
type Set[T comparable] struct {
	items map[T]struct{}
	mu    sync.RWMutex
}

// NewSet creates a new empty set.
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{items: make(map[T]struct{})}
}

// SetFrom creates a set from a slice.
func SetFrom[T comparable](items []T) *Set[T] {
	s := NewSet[T]()
	for _, item := range items {
		s.Add(item)
	}
	return s
}

// Add adds an item to the set.
func (s *Set[T]) Add(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item] = struct{}{}
}

// Remove removes an item from the set.
func (s *Set[T]) Remove(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, item)
}

// Contains checks if item is in the set.
func (s *Set[T]) Contains(item T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.items[item]
	return ok
}

// Size returns the number of items.
func (s *Set[T]) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// IsEmpty returns true if set is empty.
func (s *Set[T]) IsEmpty() bool {
	return s.Size() == 0
}

// Clear removes all items.
func (s *Set[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[T]struct{})
}

// ToSlice returns items as a slice.
func (s *Set[T]) ToSlice() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]T, 0, len(s.items))
	for item := range s.items {
		result = append(result, item)
	}
	return result
}

// Iterator returns an iterator over the set.
func (s *Set[T]) Iterator() Iterator[T] {
	return FromSlice(s.ToSlice())
}

// Union returns a new set with items from both sets.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	for item := range s.items {
		result.Add(item)
	}
	for item := range other.items {
		result.Add(item)
	}
	return result
}

// Intersection returns a new set with items in both sets.
func (s *Set[T]) Intersection(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	s.mu.RLock()
	defer s.mu.RUnlock()
	for item := range s.items {
		if other.Contains(item) {
			result.Add(item)
		}
	}
	return result
}

// Difference returns a new set with items in s but not in other.
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	result := NewSet[T]()
	s.mu.RLock()
	defer s.mu.RUnlock()
	for item := range s.items {
		if !other.Contains(item) {
			result.Add(item)
		}
	}
	return result
}

// IsSubset checks if s is a subset of other.
func (s *Set[T]) IsSubset(other *Set[T]) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for item := range s.items {
		if !other.Contains(item) {
			return false
		}
	}
	return true
}

// Equals checks if two sets are equal.
func (s *Set[T]) Equals(other *Set[T]) bool {
	if s.Size() != other.Size() {
		return false
	}
	return s.IsSubset(other)
}
