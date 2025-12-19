// Package iterator provides a generic iterator type for lazy sequence processing.
package iterator

import "github.com/auth-platform/libs/go/functional/option"

// Iterator represents a lazy sequence that can be iterated.
type Iterator[T any] struct {
	items []T
	index int
}

// FromSlice creates an iterator from a slice.
func FromSlice[T any](items []T) *Iterator[T] {
	return &Iterator[T]{items: items, index: 0}
}

// Of creates an iterator from values.
func Of[T any](items ...T) *Iterator[T] {
	return FromSlice(items)
}

// HasNext returns true if there are more elements.
func (it *Iterator[T]) HasNext() bool {
	return it.index < len(it.items)
}

// Next returns the next element and advances the iterator.
func (it *Iterator[T]) Next() option.Option[T] {
	if !it.HasNext() {
		return option.None[T]()
	}
	item := it.items[it.index]
	it.index++
	return option.Some(item)
}

// Peek returns the next element without advancing.
func (it *Iterator[T]) Peek() option.Option[T] {
	if !it.HasNext() {
		return option.None[T]()
	}
	return option.Some(it.items[it.index])
}

// Reset resets the iterator to the beginning.
func (it *Iterator[T]) Reset() {
	it.index = 0
}

// Remaining returns the number of remaining elements.
func (it *Iterator[T]) Remaining() int {
	return len(it.items) - it.index
}

// Collect consumes the iterator and returns remaining elements as a slice.
func (it *Iterator[T]) Collect() []T {
	result := make([]T, 0, it.Remaining())
	for it.HasNext() {
		result = append(result, it.Next().Unwrap())
	}
	return result
}

// Map transforms elements using the given function.
func Map[T, U any](it *Iterator[T], fn func(T) U) *Iterator[U] {
	result := make([]U, 0, it.Remaining())
	for it.HasNext() {
		result = append(result, fn(it.Next().Unwrap()))
	}
	return FromSlice(result)
}

// Filter keeps elements matching the predicate.
func Filter[T any](it *Iterator[T], predicate func(T) bool) *Iterator[T] {
	result := make([]T, 0)
	for it.HasNext() {
		item := it.Next().Unwrap()
		if predicate(item) {
			result = append(result, item)
		}
	}
	return FromSlice(result)
}

// Take returns an iterator with at most n elements.
func Take[T any](it *Iterator[T], n int) *Iterator[T] {
	result := make([]T, 0, n)
	for i := 0; i < n && it.HasNext(); i++ {
		result = append(result, it.Next().Unwrap())
	}
	return FromSlice(result)
}

// Skip skips n elements and returns the rest.
func Skip[T any](it *Iterator[T], n int) *Iterator[T] {
	for i := 0; i < n && it.HasNext(); i++ {
		it.Next()
	}
	return FromSlice(it.Collect())
}

// ForEach applies fn to each remaining element.
func (it *Iterator[T]) ForEach(fn func(T)) {
	for it.HasNext() {
		fn(it.Next().Unwrap())
	}
}

// Find returns the first element matching the predicate.
func (it *Iterator[T]) Find(predicate func(T) bool) option.Option[T] {
	for it.HasNext() {
		item := it.Next().Unwrap()
		if predicate(item) {
			return option.Some(item)
		}
	}
	return option.None[T]()
}

// Any returns true if any element matches the predicate.
func (it *Iterator[T]) Any(predicate func(T) bool) bool {
	for it.HasNext() {
		if predicate(it.Next().Unwrap()) {
			return true
		}
	}
	return false
}

// All returns true if all elements match the predicate.
func (it *Iterator[T]) All(predicate func(T) bool) bool {
	for it.HasNext() {
		if !predicate(it.Next().Unwrap()) {
			return false
		}
	}
	return true
}

// Count returns the number of remaining elements (consumes iterator).
func (it *Iterator[T]) Count() int {
	count := 0
	for it.HasNext() {
		it.Next()
		count++
	}
	return count
}

// Reduce folds elements using the given function.
func Reduce[T any](it *Iterator[T], initial T, fn func(T, T) T) T {
	result := initial
	for it.HasNext() {
		result = fn(result, it.Next().Unwrap())
	}
	return result
}

// Fold folds elements to a different type.
func Fold[T, U any](it *Iterator[T], initial U, fn func(U, T) U) U {
	result := initial
	for it.HasNext() {
		result = fn(result, it.Next().Unwrap())
	}
	return result
}

// Zip combines two iterators into pairs.
func Zip[T, U any](it1 *Iterator[T], it2 *Iterator[U]) *Iterator[struct{ First T; Second U }] {
	result := make([]struct{ First T; Second U }, 0)
	for it1.HasNext() && it2.HasNext() {
		result = append(result, struct{ First T; Second U }{First: it1.Next().Unwrap(), Second: it2.Next().Unwrap()})
	}
	return FromSlice(result)
}

// Chain concatenates two iterators.
func Chain[T any](it1, it2 *Iterator[T]) *Iterator[T] {
	result := make([]T, 0, it1.Remaining()+it2.Remaining())
	result = append(result, it1.Collect()...)
	result = append(result, it2.Collect()...)
	return FromSlice(result)
}
