// Package stream provides a generic lazy stream type for functional data processing.
package stream

import (
	"sort"

	"github.com/auth-platform/libs/go/functional/option"
)

// Stream represents a lazy sequence of elements.
type Stream[T any] struct {
	source func() []T
}

// Of creates a stream from values.
func Of[T any](items ...T) Stream[T] {
	return Stream[T]{source: func() []T { return items }}
}

// FromSlice creates a stream from a slice.
func FromSlice[T any](items []T) Stream[T] {
	return Stream[T]{source: func() []T { return items }}
}

// Empty creates an empty stream.
func Empty[T any]() Stream[T] {
	return Stream[T]{source: func() []T { return nil }}
}

// Generate creates a stream by repeatedly calling a generator function.
func Generate[T any](n int, gen func(int) T) Stream[T] {
	return Stream[T]{source: func() []T {
		result := make([]T, n)
		for i := 0; i < n; i++ {
			result[i] = gen(i)
		}
		return result
	}}
}

// Map transforms elements using the given function.
func (s Stream[T]) Map(fn func(T) T) Stream[T] {
	return Stream[T]{source: func() []T {
		items := s.source()
		result := make([]T, len(items))
		for i, item := range items {
			result[i] = fn(item)
		}
		return result
	}}
}

// MapTo transforms elements to a different type.
func MapTo[T, U any](s Stream[T], fn func(T) U) Stream[U] {
	return Stream[U]{source: func() []U {
		items := s.source()
		result := make([]U, len(items))
		for i, item := range items {
			result[i] = fn(item)
		}
		return result
	}}
}

// Filter keeps elements matching the predicate.
func (s Stream[T]) Filter(predicate func(T) bool) Stream[T] {
	return Stream[T]{source: func() []T {
		items := s.source()
		result := make([]T, 0)
		for _, item := range items {
			if predicate(item) {
				result = append(result, item)
			}
		}
		return result
	}}
}

// FlatMap transforms and flattens elements.
func FlatMap[T, U any](s Stream[T], fn func(T) Stream[U]) Stream[U] {
	return Stream[U]{source: func() []U {
		items := s.source()
		result := make([]U, 0)
		for _, item := range items {
			result = append(result, fn(item).source()...)
		}
		return result
	}}
}

// Reduce folds elements using the given function.
func (s Stream[T]) Reduce(initial T, fn func(T, T) T) T {
	items := s.source()
	result := initial
	for _, item := range items {
		result = fn(result, item)
	}
	return result
}

// ReduceTo folds elements to a different type.
func ReduceTo[T, U any](s Stream[T], initial U, fn func(U, T) U) U {
	items := s.source()
	result := initial
	for _, item := range items {
		result = fn(result, item)
	}
	return result
}

// Collect materializes the stream into a slice.
func (s Stream[T]) Collect() []T {
	return s.source()
}

// FindFirst returns the first element, if any.
func (s Stream[T]) FindFirst() option.Option[T] {
	items := s.source()
	if len(items) == 0 {
		return option.None[T]()
	}
	return option.Some(items[0])
}

// FindLast returns the last element, if any.
func (s Stream[T]) FindLast() option.Option[T] {
	items := s.source()
	if len(items) == 0 {
		return option.None[T]()
	}
	return option.Some(items[len(items)-1])
}

// Find returns the first element matching the predicate.
func (s Stream[T]) Find(predicate func(T) bool) option.Option[T] {
	items := s.source()
	for _, item := range items {
		if predicate(item) {
			return option.Some(item)
		}
	}
	return option.None[T]()
}

// AnyMatch returns true if any element matches the predicate.
func (s Stream[T]) AnyMatch(predicate func(T) bool) bool {
	items := s.source()
	for _, item := range items {
		if predicate(item) {
			return true
		}
	}
	return false
}

// AllMatch returns true if all elements match the predicate.
func (s Stream[T]) AllMatch(predicate func(T) bool) bool {
	items := s.source()
	for _, item := range items {
		if !predicate(item) {
			return false
		}
	}
	return true
}

// NoneMatch returns true if no elements match the predicate.
func (s Stream[T]) NoneMatch(predicate func(T) bool) bool {
	return !s.AnyMatch(predicate)
}

// Count returns the number of elements.
func (s Stream[T]) Count() int {
	return len(s.source())
}

// Sorted returns a sorted stream using the given less function.
func (s Stream[T]) Sorted(less func(T, T) bool) Stream[T] {
	return Stream[T]{source: func() []T {
		items := s.source()
		result := make([]T, len(items))
		copy(result, items)
		sort.Slice(result, func(i, j int) bool {
			return less(result[i], result[j])
		})
		return result
	}}
}

// Distinct removes duplicates (requires comparable type).
func Distinct[T comparable](s Stream[T]) Stream[T] {
	return Stream[T]{source: func() []T {
		items := s.source()
		seen := make(map[T]struct{})
		result := make([]T, 0)
		for _, item := range items {
			if _, ok := seen[item]; !ok {
				seen[item] = struct{}{}
				result = append(result, item)
			}
		}
		return result
	}}
}

// Limit takes the first n elements.
func (s Stream[T]) Limit(n int) Stream[T] {
	return Stream[T]{source: func() []T {
		items := s.source()
		if n >= len(items) {
			return items
		}
		return items[:n]
	}}
}

// Skip skips the first n elements.
func (s Stream[T]) Skip(n int) Stream[T] {
	return Stream[T]{source: func() []T {
		items := s.source()
		if n >= len(items) {
			return nil
		}
		return items[n:]
	}}
}

// GroupBy groups elements by key.
func GroupBy[T any, K comparable](s Stream[T], keyFn func(T) K) map[K][]T {
	items := s.source()
	result := make(map[K][]T)
	for _, item := range items {
		key := keyFn(item)
		result[key] = append(result[key], item)
	}
	return result
}

// Partition splits elements into two groups based on predicate.
func (s Stream[T]) Partition(predicate func(T) bool) ([]T, []T) {
	items := s.source()
	matching := make([]T, 0)
	notMatching := make([]T, 0)
	for _, item := range items {
		if predicate(item) {
			matching = append(matching, item)
		} else {
			notMatching = append(notMatching, item)
		}
	}
	return matching, notMatching
}

// ForEach applies fn to each element.
func (s Stream[T]) ForEach(fn func(T)) {
	items := s.source()
	for _, item := range items {
		fn(item)
	}
}

// Peek applies fn to each element and returns the stream unchanged.
func (s Stream[T]) Peek(fn func(T)) Stream[T] {
	return Stream[T]{source: func() []T {
		items := s.source()
		for _, item := range items {
			fn(item)
		}
		return items
	}}
}

// Concat concatenates two streams.
func Concat[T any](s1, s2 Stream[T]) Stream[T] {
	return Stream[T]{source: func() []T {
		items1 := s1.source()
		items2 := s2.source()
		result := make([]T, 0, len(items1)+len(items2))
		result = append(result, items1...)
		result = append(result, items2...)
		return result
	}}
}

// Reverse returns a stream with elements in reverse order.
func (s Stream[T]) Reverse() Stream[T] {
	return Stream[T]{source: func() []T {
		items := s.source()
		result := make([]T, len(items))
		for i, item := range items {
			result[len(items)-1-i] = item
		}
		return result
	}}
}
