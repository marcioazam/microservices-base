package patterns

// Specification pattern for composable business rules.

// Spec represents a specification that can be evaluated.
type Spec[T any] interface {
	IsSatisfiedBy(T) bool
}

// SpecFunc is a function-based specification.
type SpecFunc[T any] func(T) bool

// IsSatisfiedBy implements Spec interface.
func (f SpecFunc[T]) IsSatisfiedBy(t T) bool {
	return f(t)
}

// And creates a specification that requires both specs to be satisfied.
func And[T any](left, right Spec[T]) Spec[T] {
	return SpecFunc[T](func(t T) bool {
		return left.IsSatisfiedBy(t) && right.IsSatisfiedBy(t)
	})
}

// Or creates a specification that requires either spec to be satisfied.
func Or[T any](left, right Spec[T]) Spec[T] {
	return SpecFunc[T](func(t T) bool {
		return left.IsSatisfiedBy(t) || right.IsSatisfiedBy(t)
	})
}

// Not creates a specification that negates the given spec.
func Not[T any](spec Spec[T]) Spec[T] {
	return SpecFunc[T](func(t T) bool {
		return !spec.IsSatisfiedBy(t)
	})
}

// All creates a specification that requires all specs to be satisfied.
func All[T any](specs ...Spec[T]) Spec[T] {
	return SpecFunc[T](func(t T) bool {
		for _, spec := range specs {
			if !spec.IsSatisfiedBy(t) {
				return false
			}
		}
		return true
	})
}

// Any creates a specification that requires any spec to be satisfied.
func Any[T any](specs ...Spec[T]) Spec[T] {
	return SpecFunc[T](func(t T) bool {
		for _, spec := range specs {
			if spec.IsSatisfiedBy(t) {
				return true
			}
		}
		return false
	})
}

// None creates a specification that requires no specs to be satisfied.
func None[T any](specs ...Spec[T]) Spec[T] {
	return Not(Any(specs...))
}

// Filter returns items that satisfy the specification.
func Filter[T any](items []T, spec Spec[T]) []T {
	var result []T
	for _, item := range items {
		if spec.IsSatisfiedBy(item) {
			result = append(result, item)
		}
	}
	return result
}

// FindFirst returns the first item that satisfies the specification.
func FindFirst[T any](items []T, spec Spec[T]) (T, bool) {
	for _, item := range items {
		if spec.IsSatisfiedBy(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// Count returns the number of items that satisfy the specification.
func Count[T any](items []T, spec Spec[T]) int {
	count := 0
	for _, item := range items {
		if spec.IsSatisfiedBy(item) {
			count++
		}
	}
	return count
}

// Partition splits items into those that satisfy and don't satisfy the spec.
func Partition[T any](items []T, spec Spec[T]) (satisfied, unsatisfied []T) {
	for _, item := range items {
		if spec.IsSatisfiedBy(item) {
			satisfied = append(satisfied, item)
		} else {
			unsatisfied = append(unsatisfied, item)
		}
	}
	return
}
