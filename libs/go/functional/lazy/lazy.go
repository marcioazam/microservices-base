// Package lazy provides generic lazy initialization types.
package lazy

import (
	"sync"
	"sync/atomic"
)

// Lazy is a generic lazy value that is initialized on first access.
type Lazy[T any] struct {
	once  sync.Once
	value T
	fn    func() T
	done  uint32
}

// New creates a new Lazy with the given initialization function.
func New[T any](fn func() T) *Lazy[T] {
	return &Lazy[T]{fn: fn}
}

// Get returns the value, initializing it if necessary.
func (l *Lazy[T]) Get() T {
	l.once.Do(func() {
		l.value = l.fn()
		atomic.StoreUint32(&l.done, 1)
	})
	return l.value
}

// IsInitialized returns true if the value has been initialized.
func (l *Lazy[T]) IsInitialized() bool {
	return atomic.LoadUint32(&l.done) == 1
}

// LazyWithError is a lazy value that can fail during initialization.
type LazyWithError[T any] struct {
	mu    sync.Mutex
	value T
	err   error
	fn    func() (T, error)
	done  bool
}

// NewWithError creates a new LazyWithError with the given initialization function.
func NewWithError[T any](fn func() (T, error)) *LazyWithError[T] {
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

// Memoize creates a memoized version of a function.
func Memoize[T any](fn func() T) func() T {
	lazy := New(fn)
	return func() T {
		return lazy.Get()
	}
}

// MemoizeWithError creates a memoized version of a fallible function.
func MemoizeWithError[T any](fn func() (T, error)) func() (T, error) {
	lazy := NewWithError(fn)
	return func() (T, error) {
		return lazy.Get()
	}
}

// Value creates a lazy value from a constant.
func Value[T any](value T) *Lazy[T] {
	l := &Lazy[T]{
		value: value,
		done:  1,
		fn:    func() T { return value }, // Provide a no-op function
	}
	// Mark the once as done by calling Do with an empty function
	l.once.Do(func() {})
	return l
}
