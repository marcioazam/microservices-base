// Package set provides a generic Set[T] type for unique element collections.
package set

import "sync"

// Set is a generic set type that stores unique elements.
type Set[T comparable] struct {
	mu    sync.RWMutex
	items map[T]struct{}
}

// New creates a new empty Set.
func New[T comparable]() *Set[T] {
	return &Set[T]{items: make(map[T]struct{})}
}

// Of creates a new Set from the given elements.
func Of[T comparable](elements ...T) *Set[T] {
	s := New[T]()
	for _, e := range elements {
		s.items[e] = struct{}{}
	}
	return s
}

// Add adds an element to the set. Returns true if element was added (not already present).
func (s *Set[T]) Add(elem T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[elem]; exists {
		return false
	}
	s.items[elem] = struct{}{}
	return true
}

// Remove removes an element from the set. Returns true if element was removed.
func (s *Set[T]) Remove(elem T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[elem]; !exists {
		return false
	}
	delete(s.items, elem)
	return true
}

// Contains returns true if the set contains the element.
func (s *Set[T]) Contains(elem T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.items[elem]
	return exists
}

// Len returns the number of elements in the set.
func (s *Set[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// IsEmpty returns true if the set is empty.
func (s *Set[T]) IsEmpty() bool {
	return s.Len() == 0
}

// Clear removes all elements from the set.
func (s *Set[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[T]struct{})
}

// ToSlice returns all elements as a slice.
func (s *Set[T]) ToSlice() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]T, 0, len(s.items))
	for elem := range s.items {
		result = append(result, elem)
	}
	return result
}

// ForEach applies fn to each element in the set.
func (s *Set[T]) ForEach(fn func(T)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for elem := range s.items {
		fn(elem)
	}
}

// Union returns a new set containing all elements from both sets.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	s.mu.RLock()
	other.mu.RLock()
	defer s.mu.RUnlock()
	defer other.mu.RUnlock()

	result := New[T]()
	for elem := range s.items {
		result.items[elem] = struct{}{}
	}
	for elem := range other.items {
		result.items[elem] = struct{}{}
	}
	return result
}

// Intersection returns a new set containing elements present in both sets.
func (s *Set[T]) Intersection(other *Set[T]) *Set[T] {
	s.mu.RLock()
	other.mu.RLock()
	defer s.mu.RUnlock()
	defer other.mu.RUnlock()

	result := New[T]()
	// Iterate over the smaller set for efficiency
	smaller, larger := s.items, other.items
	if len(s.items) > len(other.items) {
		smaller, larger = other.items, s.items
	}
	for elem := range smaller {
		if _, exists := larger[elem]; exists {
			result.items[elem] = struct{}{}
		}
	}
	return result
}

// Difference returns a new set containing elements in s but not in other.
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	s.mu.RLock()
	other.mu.RLock()
	defer s.mu.RUnlock()
	defer other.mu.RUnlock()

	result := New[T]()
	for elem := range s.items {
		if _, exists := other.items[elem]; !exists {
			result.items[elem] = struct{}{}
		}
	}
	return result
}

// SymmetricDifference returns elements in either set but not both.
func (s *Set[T]) SymmetricDifference(other *Set[T]) *Set[T] {
	s.mu.RLock()
	other.mu.RLock()
	defer s.mu.RUnlock()
	defer other.mu.RUnlock()

	result := New[T]()
	for elem := range s.items {
		if _, exists := other.items[elem]; !exists {
			result.items[elem] = struct{}{}
		}
	}
	for elem := range other.items {
		if _, exists := s.items[elem]; !exists {
			result.items[elem] = struct{}{}
		}
	}
	return result
}

// IsSubset returns true if all elements of s are in other.
func (s *Set[T]) IsSubset(other *Set[T]) bool {
	s.mu.RLock()
	other.mu.RLock()
	defer s.mu.RUnlock()
	defer other.mu.RUnlock()

	for elem := range s.items {
		if _, exists := other.items[elem]; !exists {
			return false
		}
	}
	return true
}

// IsSuperset returns true if s contains all elements of other.
func (s *Set[T]) IsSuperset(other *Set[T]) bool {
	return other.IsSubset(s)
}

// Equal returns true if both sets contain the same elements.
func (s *Set[T]) Equal(other *Set[T]) bool {
	s.mu.RLock()
	other.mu.RLock()
	defer s.mu.RUnlock()
	defer other.mu.RUnlock()

	if len(s.items) != len(other.items) {
		return false
	}
	for elem := range s.items {
		if _, exists := other.items[elem]; !exists {
			return false
		}
	}
	return true
}

// Clone returns a copy of the set.
func (s *Set[T]) Clone() *Set[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := New[T]()
	for elem := range s.items {
		result.items[elem] = struct{}{}
	}
	return result
}

// Filter returns a new set with elements that satisfy the predicate.
func (s *Set[T]) Filter(predicate func(T) bool) *Set[T] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := New[T]()
	for elem := range s.items {
		if predicate(elem) {
			result.items[elem] = struct{}{}
		}
	}
	return result
}

// Map applies fn to each element and returns a new set with the results.
func Map[T, U comparable](s *Set[T], fn func(T) U) *Set[U] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := New[U]()
	for elem := range s.items {
		result.items[fn(elem)] = struct{}{}
	}
	return result
}

// FromSlice creates a set from a slice.
func FromSlice[T comparable](slice []T) *Set[T] {
	s := New[T]()
	for _, elem := range slice {
		s.items[elem] = struct{}{}
	}
	return s
}
