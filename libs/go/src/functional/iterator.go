package functional

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

// TakeWhile yields elements while predicate is true.
func TakeWhile[T any](iter Iterator[T], pred func(T) bool) Iterator[T] {
	return func(yield func(T) bool) {
		iter(func(t T) bool {
			if !pred(t) {
				return false
			}
			return yield(t)
		})
	}
}

// SkipWhile skips elements while predicate is true.
func SkipWhile[T any](iter Iterator[T], pred func(T) bool) Iterator[T] {
	return func(yield func(T) bool) {
		skipping := true
		iter(func(t T) bool {
			if skipping && pred(t) {
				return true
			}
			skipping = false
			return yield(t)
		})
	}
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
func Find[T any](iter Iterator[T], pred func(T) bool) Option[T] {
	var result Option[T] = None[T]()
	iter(func(t T) bool {
		if pred(t) {
			result = Some(t)
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

// Enumerate adds index to each element.
func Enumerate[T any](iter Iterator[T]) Iterator[Pair[int, T]] {
	return func(yield func(Pair[int, T]) bool) {
		i := 0
		iter(func(t T) bool {
			result := yield(NewPair(i, t))
			i++
			return result
		})
	}
}

// Flatten flattens nested iterators.
func Flatten[T any](iter Iterator[Iterator[T]]) Iterator[T] {
	return func(yield func(T) bool) {
		iter(func(inner Iterator[T]) bool {
			shouldContinue := true
			inner(func(t T) bool {
				if !yield(t) {
					shouldContinue = false
					return false
				}
				return true
			})
			return shouldContinue
		})
	}
}

// FlatMap maps and flattens in one operation.
func FlatMap[T, U any](iter Iterator[T], fn func(T) Iterator[U]) Iterator[U] {
	return Flatten(Map(iter, fn))
}
