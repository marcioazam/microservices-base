// Package slices provides generic slice utility functions.
package slices

import "github.com/authcorp/libs/go/src/functional"

// Map applies fn to each element of the slice, returning a new slice.
func Map[T, U any](slice []T, fn func(T) U) []U {
	result := make([]U, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

// Filter returns a new slice containing only elements that satisfy the predicate.
func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0)
	for _, v := range slice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

// Reduce folds the slice into a single value using the accumulator function.
func Reduce[T, U any](slice []T, initial U, fn func(U, T) U) U {
	acc := initial
	for _, v := range slice {
		acc = fn(acc, v)
	}
	return acc
}

// Find returns the first element that satisfies the predicate.
func Find[T any](slice []T, predicate func(T) bool) functional.Option[T] {
	for _, v := range slice {
		if predicate(v) {
			return functional.Some(v)
		}
	}
	return functional.None[T]()
}

// Any returns true if any element satisfies the predicate.
func Any[T any](slice []T, predicate func(T) bool) bool {
	for _, v := range slice {
		if predicate(v) {
			return true
		}
	}
	return false
}

// All returns true if all elements satisfy the predicate.
func All[T any](slice []T, predicate func(T) bool) bool {
	for _, v := range slice {
		if !predicate(v) {
			return false
		}
	}
	return true
}


// GroupBy groups elements by a key function.
func GroupBy[T any, K comparable](slice []T, keyFn func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, v := range slice {
		key := keyFn(v)
		result[key] = append(result[key], v)
	}
	return result
}

// Partition splits the slice into two: elements that satisfy the predicate and those that don't.
func Partition[T any](slice []T, predicate func(T) bool) ([]T, []T) {
	matching := make([]T, 0)
	notMatching := make([]T, 0)
	for _, v := range slice {
		if predicate(v) {
			matching = append(matching, v)
		} else {
			notMatching = append(notMatching, v)
		}
	}
	return matching, notMatching
}

// Chunk splits the slice into chunks of the given size.
func Chunk[T any](slice []T, size int) [][]T {
	if size <= 0 {
		return nil
	}
	result := make([][]T, 0, (len(slice)+size-1)/size)
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		result = append(result, slice[i:end])
	}
	return result
}

// Flatten flattens a slice of slices into a single slice.
func Flatten[T any](slices [][]T) []T {
	total := 0
	for _, s := range slices {
		total += len(s)
	}
	result := make([]T, 0, total)
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// Contains returns true if the slice contains the element.
func Contains[T comparable](slice []T, elem T) bool {
	for _, v := range slice {
		if v == elem {
			return true
		}
	}
	return false
}

// IndexOf returns the index of the first occurrence of elem, or -1 if not found.
func IndexOf[T comparable](slice []T, elem T) int {
	for i, v := range slice {
		if v == elem {
			return i
		}
	}
	return -1
}

// Unique returns a new slice with duplicate elements removed.
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0)
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// Reverse returns a new slice with elements in reverse order.
func Reverse[T any](slice []T) []T {
	result := make([]T, len(slice))
	for i, v := range slice {
		result[len(slice)-1-i] = v
	}
	return result
}

// First returns the first element of the slice.
func First[T any](slice []T) functional.Option[T] {
	if len(slice) == 0 {
		return functional.None[T]()
	}
	return functional.Some(slice[0])
}

// Last returns the last element of the slice.
func Last[T any](slice []T) functional.Option[T] {
	if len(slice) == 0 {
		return functional.None[T]()
	}
	return functional.Some(slice[len(slice)-1])
}

// Take returns the first n elements of the slice.
func Take[T any](slice []T, n int) []T {
	if n <= 0 {
		return []T{}
	}
	if n >= len(slice) {
		return slice
	}
	return slice[:n]
}

// Drop returns the slice without the first n elements.
func Drop[T any](slice []T, n int) []T {
	if n <= 0 {
		return slice
	}
	if n >= len(slice) {
		return []T{}
	}
	return slice[n:]
}

// FlatMap applies fn to each element and flattens the result.
func FlatMap[T, U any](slice []T, fn func(T) []U) []U {
	result := make([]U, 0)
	for _, v := range slice {
		result = append(result, fn(v)...)
	}
	return result
}

// ForEach applies fn to each element.
func ForEach[T any](slice []T, fn func(T)) {
	for _, v := range slice {
		fn(v)
	}
}

// Count returns the number of elements that satisfy the predicate.
func Count[T any](slice []T, predicate func(T) bool) int {
	count := 0
	for _, v := range slice {
		if predicate(v) {
			count++
		}
	}
	return count
}
