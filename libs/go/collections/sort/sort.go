// Package sort provides generic sorting utilities for slices.
package sort

import (
	"sort"
)

// Sort sorts a slice in-place using the provided less function.
func Sort[T any](slice []T, less func(a, b T) bool) {
	sort.Slice(slice, func(i, j int) bool {
		return less(slice[i], slice[j])
	})
}

// SortStable sorts a slice in-place using stable sort.
// Stable sort preserves the original order of equal elements.
func SortStable[T any](slice []T, less func(a, b T) bool) {
	sort.SliceStable(slice, func(i, j int) bool {
		return less(slice[i], slice[j])
	})
}

// Sorted returns a sorted copy of the slice.
func Sorted[T any](slice []T, less func(a, b T) bool) []T {
	result := make([]T, len(slice))
	copy(result, slice)
	Sort(result, less)
	return result
}

// SortedStable returns a stable sorted copy of the slice.
func SortedStable[T any](slice []T, less func(a, b T) bool) []T {
	result := make([]T, len(slice))
	copy(result, slice)
	SortStable(result, less)
	return result
}

// SortBy sorts a slice by a key extracted from each element.
func SortBy[T any, K Ordered](slice []T, keyFn func(T) K) {
	Sort(slice, func(a, b T) bool {
		return keyFn(a) < keyFn(b)
	})
}

// SortByDesc sorts a slice by a key in descending order.
func SortByDesc[T any, K Ordered](slice []T, keyFn func(T) K) {
	Sort(slice, func(a, b T) bool {
		return keyFn(a) > keyFn(b)
	})
}

// SortByMultiple sorts a slice using multiple key functions.
// Earlier keys have higher priority.
func SortByMultiple[T any](slice []T, comparators ...func(a, b T) int) {
	Sort(slice, func(a, b T) bool {
		for _, cmp := range comparators {
			result := cmp(a, b)
			if result != 0 {
				return result < 0
			}
		}
		return false
	})
}

// IsSorted checks if a slice is sorted according to the less function.
func IsSorted[T any](slice []T, less func(a, b T) bool) bool {
	for i := 1; i < len(slice); i++ {
		if less(slice[i], slice[i-1]) {
			return false
		}
	}
	return true
}

// IsSortedBy checks if a slice is sorted by a key.
func IsSortedBy[T any, K Ordered](slice []T, keyFn func(T) K) bool {
	return IsSorted(slice, func(a, b T) bool {
		return keyFn(a) < keyFn(b)
	})
}

// Reverse reverses a slice in-place.
func Reverse[T any](slice []T) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// Reversed returns a reversed copy of the slice.
func Reversed[T any](slice []T) []T {
	result := make([]T, len(slice))
	for i, v := range slice {
		result[len(slice)-1-i] = v
	}
	return result
}

// Min returns the minimum element in a slice.
func Min[T Ordered](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	min := slice[0]
	for _, v := range slice[1:] {
		if v < min {
			min = v
		}
	}
	return min, true
}

// Max returns the maximum element in a slice.
func Max[T Ordered](slice []T) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	max := slice[0]
	for _, v := range slice[1:] {
		if v > max {
			max = v
		}
	}
	return max, true
}

// MinBy returns the minimum element by a key function.
func MinBy[T any, K Ordered](slice []T, keyFn func(T) K) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	min := slice[0]
	minKey := keyFn(min)
	for _, v := range slice[1:] {
		key := keyFn(v)
		if key < minKey {
			min = v
			minKey = key
		}
	}
	return min, true
}

// MaxBy returns the maximum element by a key function.
func MaxBy[T any, K Ordered](slice []T, keyFn func(T) K) (T, bool) {
	if len(slice) == 0 {
		var zero T
		return zero, false
	}
	max := slice[0]
	maxKey := keyFn(max)
	for _, v := range slice[1:] {
		key := keyFn(v)
		if key > maxKey {
			max = v
			maxKey = key
		}
	}
	return max, true
}

// TopN returns the top N elements (largest first).
func TopN[T any](slice []T, n int, less func(a, b T) bool) []T {
	if n <= 0 {
		return nil
	}
	if n >= len(slice) {
		result := Sorted(slice, less)
		Reverse(result)
		return result
	}

	// Use partial sort for efficiency
	result := make([]T, len(slice))
	copy(result, slice)
	Sort(result, func(a, b T) bool { return less(b, a) }) // Reverse order
	return result[:n]
}

// BottomN returns the bottom N elements (smallest first).
func BottomN[T any](slice []T, n int, less func(a, b T) bool) []T {
	if n <= 0 {
		return nil
	}
	if n >= len(slice) {
		return Sorted(slice, less)
	}

	result := make([]T, len(slice))
	copy(result, slice)
	Sort(result, less)
	return result[:n]
}

// Ordered is a constraint for types that support ordering.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

// Compare returns a comparator function for ordered types.
func Compare[T Ordered]() func(a, b T) int {
	return func(a, b T) int {
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
}

// CompareBy returns a comparator that extracts a key.
func CompareBy[T any, K Ordered](keyFn func(T) K) func(a, b T) int {
	return func(a, b T) int {
		ka, kb := keyFn(a), keyFn(b)
		if ka < kb {
			return -1
		}
		if ka > kb {
			return 1
		}
		return 0
	}
}

// CompareDesc returns a descending comparator.
func CompareDesc[T Ordered]() func(a, b T) int {
	return func(a, b T) int {
		if a > b {
			return -1
		}
		if a < b {
			return 1
		}
		return 0
	}
}

// Shuffle randomly shuffles a slice in-place.
func Shuffle[T any](slice []T, randFn func(n int) int) {
	for i := len(slice) - 1; i > 0; i-- {
		j := randFn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// BinarySearch searches for a value in a sorted slice.
// Returns the index and true if found, or insertion point and false if not.
func BinarySearch[T Ordered](slice []T, target T) (int, bool) {
	lo, hi := 0, len(slice)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if slice[mid] < target {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(slice) && slice[lo] == target {
		return lo, true
	}
	return lo, false
}

// BinarySearchBy searches using a key function.
func BinarySearchBy[T any, K Ordered](slice []T, target K, keyFn func(T) K) (int, bool) {
	lo, hi := 0, len(slice)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if keyFn(slice[mid]) < target {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(slice) && keyFn(slice[lo]) == target {
		return lo, true
	}
	return lo, false
}
