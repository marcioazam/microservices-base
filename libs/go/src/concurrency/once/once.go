// Package once provides a generic once-with-result type for lazy initialization.
package once

import (
	"sync"
	"sync/atomic"
)

// Once is a generic once-with-result that lazily initializes a value.
type Once[T any] struct {
	done  uint32
	mu    sync.Mutex
	value T
	err   error
	fn    func() (T, error)
}

// New creates a new Once with the given initialization function.
func New[T any](fn func() (T, error)) *Once[T] {
	return &Once[T]{fn: fn}
}

// NewSimple creates a new Once with a simple initialization function that cannot fail.
func NewSimple[T any](fn func() T) *Once[T] {
	return &Once[T]{
		fn: func() (T, error) {
			return fn(), nil
		},
	}
}

// Get returns the value, initializing it if necessary.
// If initialization fails, the error is returned and subsequent calls will retry.
func (o *Once[T]) Get() (T, error) {
	if atomic.LoadUint32(&o.done) == 1 {
		return o.value, o.err
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.done == 0 {
		o.value, o.err = o.fn()
		if o.err == nil {
			atomic.StoreUint32(&o.done, 1)
		}
	}

	return o.value, o.err
}

// MustGet returns the value, panicking if initialization fails.
func (o *Once[T]) MustGet() T {
	v, err := o.Get()
	if err != nil {
		panic(err)
	}
	return v
}

// IsDone returns true if the value has been successfully initialized.
func (o *Once[T]) IsDone() bool {
	return atomic.LoadUint32(&o.done) == 1
}

// Reset resets the Once so that the next Get call will re-initialize.
func (o *Once[T]) Reset() {
	o.mu.Lock()
	defer o.mu.Unlock()
	atomic.StoreUint32(&o.done, 0)
	var zero T
	o.value = zero
	o.err = nil
}

// ResetWithFn resets the Once with a new initialization function.
func (o *Once[T]) ResetWithFn(fn func() (T, error)) {
	o.mu.Lock()
	defer o.mu.Unlock()
	atomic.StoreUint32(&o.done, 0)
	var zero T
	o.value = zero
	o.err = nil
	o.fn = fn
}

// Lazy is a simpler version of Once that cannot fail.
type Lazy[T any] struct {
	once        sync.Once
	value       T
	fn          func() T
	initialized uint32
}

// NewLazy creates a new Lazy with the given initialization function.
func NewLazy[T any](fn func() T) *Lazy[T] {
	return &Lazy[T]{fn: fn}
}

// Get returns the value, initializing it if necessary.
func (l *Lazy[T]) Get() T {
	l.once.Do(func() {
		l.value = l.fn()
		atomic.StoreUint32(&l.initialized, 1)
	})
	return l.value
}

// IsInitialized returns true if the value has been initialized.
func (l *Lazy[T]) IsInitialized() bool {
	return atomic.LoadUint32(&l.initialized) == 1
}
