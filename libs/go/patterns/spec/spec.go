// Package spec provides a generic specification pattern implementation.
package spec

import "github.com/auth-platform/libs/go/functional/option"

// Spec represents a specification that can be satisfied by a value.
type Spec[T any] struct {
	predicate func(T) bool
}

// New creates a new Spec from a predicate.
func New[T any](predicate func(T) bool) *Spec[T] {
	return &Spec[T]{predicate: predicate}
}

// IsSatisfiedBy returns true if the value satisfies the specification.
func (s *Spec[T]) IsSatisfiedBy(value T) bool {
	return s.predicate(value)
}

// And creates a specification that requires both specs to be satisfied.
func (s *Spec[T]) And(other *Spec[T]) *Spec[T] {
	return &Spec[T]{
		predicate: func(v T) bool {
			return s.predicate(v) && other.predicate(v)
		},
	}
}

// Or creates a specification that requires either spec to be satisfied.
func (s *Spec[T]) Or(other *Spec[T]) *Spec[T] {
	return &Spec[T]{
		predicate: func(v T) bool {
			return s.predicate(v) || other.predicate(v)
		},
	}
}

// Not creates a specification that negates this spec.
func (s *Spec[T]) Not() *Spec[T] {
	return &Spec[T]{
		predicate: func(v T) bool {
			return !s.predicate(v)
		},
	}
}

// Filter returns items that satisfy the specification.
func (s *Spec[T]) Filter(items []T) []T {
	result := make([]T, 0)
	for _, item := range items {
		if s.predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// FindFirst returns the first item that satisfies the specification.
func (s *Spec[T]) FindFirst(items []T) option.Option[T] {
	for _, item := range items {
		if s.predicate(item) {
			return option.Some(item)
		}
	}
	return option.None[T]()
}

// FindAll returns all items that satisfy the specification.
func (s *Spec[T]) FindAll(items []T) []T {
	return s.Filter(items)
}

// Count returns the number of items that satisfy the specification.
func (s *Spec[T]) Count(items []T) int {
	count := 0
	for _, item := range items {
		if s.predicate(item) {
			count++
		}
	}
	return count
}

// Any returns true if any item satisfies the specification.
func (s *Spec[T]) Any(items []T) bool {
	for _, item := range items {
		if s.predicate(item) {
			return true
		}
	}
	return false
}

// All returns true if all items satisfy the specification.
func (s *Spec[T]) All(items []T) bool {
	for _, item := range items {
		if !s.predicate(item) {
			return false
		}
	}
	return true
}

// None returns true if no items satisfy the specification.
func (s *Spec[T]) None(items []T) bool {
	return !s.Any(items)
}

// True creates a specification that is always satisfied.
func True[T any]() *Spec[T] {
	return &Spec[T]{predicate: func(T) bool { return true }}
}

// False creates a specification that is never satisfied.
func False[T any]() *Spec[T] {
	return &Spec[T]{predicate: func(T) bool { return false }}
}

// Equals creates a specification for equality.
func Equals[T comparable](value T) *Spec[T] {
	return &Spec[T]{predicate: func(v T) bool { return v == value }}
}

// NotEquals creates a specification for inequality.
func NotEquals[T comparable](value T) *Spec[T] {
	return &Spec[T]{predicate: func(v T) bool { return v != value }}
}

// GreaterThan creates a specification for greater than comparison.
func GreaterThan[T interface{ ~int | ~int64 | ~float64 | ~string }](value T) *Spec[T] {
	return &Spec[T]{predicate: func(v T) bool { return v > value }}
}

// LessThan creates a specification for less than comparison.
func LessThan[T interface{ ~int | ~int64 | ~float64 | ~string }](value T) *Spec[T] {
	return &Spec[T]{predicate: func(v T) bool { return v < value }}
}

// Between creates a specification for range check.
func Between[T interface{ ~int | ~int64 | ~float64 | ~string }](min, max T) *Spec[T] {
	return &Spec[T]{predicate: func(v T) bool { return v >= min && v <= max }}
}

// In creates a specification for set membership.
func In[T comparable](values ...T) *Spec[T] {
	set := make(map[T]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return &Spec[T]{predicate: func(v T) bool {
		_, ok := set[v]
		return ok
	}}
}
