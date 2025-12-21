package functional

import (
	"sync"
	"sync/atomic"
)

// Lazy represents a lazily evaluated value with thread-safe memoization.
type Lazy[T any] struct {
	compute func() T
	value   T
	once    sync.Once
	done    uint32
}

// NewLazy creates a new lazy value.
func NewLazy[T any](compute func() T) *Lazy[T] {
	return &Lazy[T]{compute: compute}
}

// Get returns the value, computing it if necessary.
func (l *Lazy[T]) Get() T {
	l.once.Do(func() {
		l.value = l.compute()
		atomic.StoreUint32(&l.done, 1)
	})
	return l.value
}

// IsEvaluated returns true if the value has been computed.
func (l *Lazy[T]) IsEvaluated() bool {
	return atomic.LoadUint32(&l.done) == 1
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

// LazyValue creates a lazy value from a constant (already evaluated).
func LazyValue[T any](value T) *Lazy[T] {
	l := &Lazy[T]{
		value:   value,
		done:    1,
		compute: func() T { return value },
	}
	l.once.Do(func() {})
	return l
}

// LazyWithError is a lazy value that can fail during initialization.
type LazyWithError[T any] struct {
	mu    sync.Mutex
	value T
	err   error
	fn    func() (T, error)
	done  bool
}

// NewLazyWithError creates a new LazyWithError with the given initialization function.
func NewLazyWithError[T any](fn func() (T, error)) *LazyWithError[T] {
	return &LazyWithError[T]{fn: fn}
}

// Get returns the value, initializing it if necessary.
// If initialization fails, the error is returned and subsequent calls will retry.
func (l *LazyWithError[T]) Get() (T, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.done {
		return l.value, l.err
	}

	l.value, l.err = l.fn()
	if l.err == nil {
		l.done = true
	}

	return l.value, l.err
}

// MustGet returns the value, panicking if initialization fails.
func (l *LazyWithError[T]) MustGet() T {
	v, err := l.Get()
	if err != nil {
		panic(err)
	}
	return v
}

// IsInitialized returns true if the value has been successfully initialized.
func (l *LazyWithError[T]) IsInitialized() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.done
}

// Reset resets the lazy value so it will be re-initialized on next Get.
func (l *LazyWithError[T]) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.done = false
	var zero T
	l.value = zero
	l.err = nil
}

// ResetWithFn resets the lazy value with a new initialization function.
func (l *LazyWithError[T]) ResetWithFn(fn func() (T, error)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.done = false
	var zero T
	l.value = zero
	l.err = nil
	l.fn = fn
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

// MemoizeFunc creates a memoized version of a function.
func MemoizeFunc[T any](fn func() T) func() T {
	lazy := NewLazy(fn)
	return func() T {
		return lazy.Get()
	}
}

// MemoizeFuncWithError creates a memoized version of a fallible function.
func MemoizeFuncWithError[T any](fn func() (T, error)) func() (T, error) {
	lazy := NewLazyWithError(fn)
	return func() (T, error) {
		return lazy.Get()
	}
}
