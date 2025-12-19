package functional

import "sync"

// Lazy represents a lazily evaluated value.
type Lazy[T any] struct {
	compute func() T
	value   T
	once    sync.Once
	done    bool
}

// NewLazy creates a new lazy value.
func NewLazy[T any](compute func() T) *Lazy[T] {
	return &Lazy[T]{compute: compute}
}

// Get returns the value, computing it if necessary.
func (l *Lazy[T]) Get() T {
	l.once.Do(func() {
		l.value = l.compute()
		l.done = true
	})
	return l.value
}

// IsEvaluated returns true if the value has been computed.
func (l *Lazy[T]) IsEvaluated() bool {
	return l.done
}

// MapLazy applies a function to a lazy value.
func MapLazy[T, U any](l *Lazy[T], fn func(T) U) *Lazy[U] {
	return NewLazy(func() U {
		return fn(l.Get())
	})
}

// FlatMapLazy applies a function that returns a Lazy.
func FlatMapLazy[T, U any](l *Lazy[T], fn func(T) *Lazy[U]) *Lazy[U] {
	return NewLazy(func() U {
		return fn(l.Get()).Get()
	})
}

// Thunk represents a deferred computation without memoization.
type Thunk[T any] func() T

// Force evaluates the thunk.
func (t Thunk[T]) Force() T {
	return t()
}

// MapThunk applies a function to a thunk.
func MapThunk[T, U any](t Thunk[T], fn func(T) U) Thunk[U] {
	return func() U {
		return fn(t())
	}
}

// Memoize converts a Thunk to a Lazy value.
func (t Thunk[T]) Memoize() *Lazy[T] {
	return NewLazy(func() T { return t() })
}
