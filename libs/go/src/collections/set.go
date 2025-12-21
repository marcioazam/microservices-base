package collections

import (
	"iter"
	"sync"
)

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

// All returns a Go 1.23+ iterator over the set elements.
func (s *Set[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for item := range s.items {
			if !yield(item) {
				return
			}
		}
	}
}

// Collect returns all elements as a slice (alias for ToSlice).
func (s *Set[T]) Collect() []T {
	return s.ToSlice()
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

// IsSuperset returns true if s contains all elements of other.
func (s *Set[T]) IsSuperset(other *Set[T]) bool {
	return other.IsSubset(s)
}

// SymmetricDifference returns elements in either set but not both.
func (s *Set[T]) SymmetricDifference(other *Set[T]) *Set[T] {
	s.mu.RLock()
	other.mu.RLock()
	defer s.mu.RUnlock()
	defer other.mu.RUnlock()

	result := NewSet[T]()
	for item := range s.items {
		if _, exists := other.items[item]; !exists {
			result.items[item] = struct{}{}
		}
	}
	for item := range other.items {
		if _, exists := s.items[item]; !exists {
			result.items[item] = struct{}{}
		}
	}
	return result
}

// Clone returns a copy of the set.
func (s *Set[T]) Clone() *Set[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := NewSet[T]()
	for item := range s.items {
		result.items[item] = struct{}{}
	}
	return result
}

// Filter returns a new set with elements that satisfy the predicate.
func (s *Set[T]) Filter(predicate func(T) bool) *Set[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := NewSet[T]()
	for item := range s.items {
		if predicate(item) {
			result.items[item] = struct{}{}
		}
	}
	return result
}

// SetMap applies fn to each element and returns a new set with the results.
func SetMap[T, U comparable](s *Set[T], fn func(T) U) *Set[U] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := NewSet[U]()
	for item := range s.items {
		result.items[fn(item)] = struct{}{}
	}
	return result
}

// SetOf creates a new Set from the given elements (variadic constructor).
func SetOf[T comparable](elements ...T) *Set[T] {
	s := NewSet[T]()
	for _, e := range elements {
		s.items[e] = struct{}{}
	}
	return s
}

// ForEach applies fn to each element in the set.
func (s *Set[T]) ForEach(fn func(T)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for item := range s.items {
		fn(item)
	}
}
