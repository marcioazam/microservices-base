package collections

import "github.com/authcorp/libs/go/src/functional"

// Iterator provides lazy iteration using Go 1.23+ range functions.
type Iterator[T any] func(yield func(T) bool)

// FromSlice creates an iterator from a slice.
func FromSlice[T any](slice []T) Iterator[T] {
	return func(yield func(T) bool) {
		for _, v := range slice {
			if !yield(v) {
				return
			}
		}
	}
}

// FromMap creates an iterator from a map.
func FromMap[K comparable, V any](m map[K]V) Iterator[functional.Pair[K, V]] {
	return func(yield func(functional.Pair[K, V]) bool) {
		for k, v := range m {
			if !yield(functional.NewPair(k, v)) {
				return
			}
		}
	}
}

// Map transforms iterator elements.
func Map[T, U any](iter Iterator[T], fn func(T) U) Iterator[U] {
	return func(yield func(U) bool) {
		iter(func(t T) bool {
			return yield(fn(t))
		})
	}
}

// Filter keeps elements matching predicate.
func Filter[T any](iter Iterator[T], pred func(T) bool) Iterator[T] {
	return func(yield func(T) bool) {
		iter(func(t T) bool {
			if pred(t) {
				return yield(t)
			}
			return true
		})
	}
}

// Reduce accumulates iterator values.
func Reduce[T, U any](iter Iterator[T], initial U, fn func(U, T) U) U {
	acc := initial
	iter(func(t T) bool {
		acc = fn(acc, t)
		return true
	})
	return acc
}

// ForEach applies a function to each element.
func ForEach[T any](iter Iterator[T], fn func(T)) {
	iter(func(t T) bool {
		fn(t)
		return true
	})
}

// Collect materializes iterator to slice.
func Collect[T any](iter Iterator[T]) []T {
	var result []T
	iter(func(t T) bool {
		result = append(result, t)
		return true
	})
	return result
}

// Take limits iterator to n elements.
func Take[T any](iter Iterator[T], n int) Iterator[T] {
	return func(yield func(T) bool) {
		count := 0
		iter(func(t T) bool {
			if count >= n {
				return false
			}
			count++
			return yield(t)
		})
	}
}

// Skip skips the first n elements.
func Skip[T any](iter Iterator[T], n int) Iterator[T] {
	return func(yield func(T) bool) {
		count := 0
		iter(func(t T) bool {
			if count < n {
				count++
				return true
			}
			return yield(t)
		})
	}
}

// Any returns true if any element matches predicate.
func Any[T any](iter Iterator[T], pred func(T) bool) bool {
	found := false
	iter(func(t T) bool {
		if pred(t) {
			found = true
			return false
		}
		return true
	})
	return found
}

// All returns true if all elements match predicate.
func All[T any](iter Iterator[T], pred func(T) bool) bool {
	allMatch := true
	iter(func(t T) bool {
		if !pred(t) {
			allMatch = false
			return false
		}
		return true
	})
	return allMatch
}

// Find returns the first element matching predicate.
func Find[T any](iter Iterator[T], pred func(T) bool) functional.Option[T] {
	var result functional.Option[T] = functional.None[T]()
	iter(func(t T) bool {
		if pred(t) {
			result = functional.Some(t)
			return false
		}
		return true
	})
	return result
}

// Count returns the number of elements.
func Count[T any](iter Iterator[T]) int {
	count := 0
	iter(func(_ T) bool {
		count++
		return true
	})
	return count
}

// Chain concatenates two iterators.
func Chain[T any](first, second Iterator[T]) Iterator[T] {
	return func(yield func(T) bool) {
		first(yield)
		second(yield)
	}
}

// Zip combines two iterators element-wise.
func Zip[T, U any](first Iterator[T], second Iterator[U]) Iterator[functional.Pair[T, U]] {
	return func(yield func(functional.Pair[T, U]) bool) {
		firstSlice := Collect(first)
		secondSlice := Collect(second)
		minLen := len(firstSlice)
		if len(secondSlice) < minLen {
			minLen = len(secondSlice)
		}
		for i := 0; i < minLen; i++ {
			if !yield(functional.NewPair(firstSlice[i], secondSlice[i])) {
				return
			}
		}
	}
}
