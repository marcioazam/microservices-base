// Package atomic provides generic atomic operations for any type.
package atomic

import (
	"sync"
)

// Value is a generic atomic value holder.
type Value[T any] struct {
	mu    sync.RWMutex
	value T
}

// New creates a new atomic value with the given initial value.
func New[T any](initial T) *Value[T] {
	return &Value[T]{value: initial}
}

// Load atomically loads and returns the value.
func (v *Value[T]) Load() T {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.value
}

// Store atomically stores the given value.
func (v *Value[T]) Store(val T) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.value = val
}

// Swap atomically stores the new value and returns the old value.
func (v *Value[T]) Swap(new T) T {
	v.mu.Lock()
	defer v.mu.Unlock()
	old := v.value
	v.value = new
	return old
}

// CompareAndSwap atomically compares the current value with old using the
// provided comparator, and if they are equal, sets the value to new.
// Returns true if the swap was performed.
func (v *Value[T]) CompareAndSwap(old, new T, equal func(a, b T) bool) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if equal(v.value, old) {
		v.value = new
		return true
	}
	return false
}

// Update atomically updates the value using the provided function.
// The function receives the current value and returns the new value.
func (v *Value[T]) Update(fn func(T) T) T {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.value = fn(v.value)
	return v.value
}

// UpdateAndGet atomically updates the value and returns the new value.
func (v *Value[T]) UpdateAndGet(fn func(T) T) T {
	return v.Update(fn)
}

// GetAndUpdate atomically updates the value and returns the old value.
func (v *Value[T]) GetAndUpdate(fn func(T) T) T {
	v.mu.Lock()
	defer v.mu.Unlock()
	old := v.value
	v.value = fn(v.value)
	return old
}

// Int64 is an atomic int64 value with additional numeric operations.
type Int64 struct {
	Value[int64]
}

// NewInt64 creates a new atomic int64 with the given initial value.
func NewInt64(initial int64) *Int64 {
	return &Int64{Value: Value[int64]{value: initial}}
}

// Add atomically adds delta to the value and returns the new value.
func (v *Int64) Add(delta int64) int64 {
	return v.Update(func(old int64) int64 { return old + delta })
}

// Sub atomically subtracts delta from the value and returns the new value.
func (v *Int64) Sub(delta int64) int64 {
	return v.Add(-delta)
}

// Inc atomically increments the value by 1 and returns the new value.
func (v *Int64) Inc() int64 {
	return v.Add(1)
}

// Dec atomically decrements the value by 1 and returns the new value.
func (v *Int64) Dec() int64 {
	return v.Add(-1)
}

// CompareAndSwapInt64 atomically compares and swaps int64 values.
func (v *Int64) CompareAndSwapInt64(old, new int64) bool {
	return v.CompareAndSwap(old, new, func(a, b int64) bool { return a == b })
}

// Uint64 is an atomic uint64 value with additional numeric operations.
type Uint64 struct {
	Value[uint64]
}

// NewUint64 creates a new atomic uint64 with the given initial value.
func NewUint64(initial uint64) *Uint64 {
	return &Uint64{Value: Value[uint64]{value: initial}}
}

// Add atomically adds delta to the value and returns the new value.
func (v *Uint64) Add(delta uint64) uint64 {
	return v.Update(func(old uint64) uint64 { return old + delta })
}

// Inc atomically increments the value by 1 and returns the new value.
func (v *Uint64) Inc() uint64 {
	return v.Add(1)
}

// Bool is an atomic boolean value.
type Bool struct {
	Value[bool]
}

// NewBool creates a new atomic bool with the given initial value.
func NewBool(initial bool) *Bool {
	return &Bool{Value: Value[bool]{value: initial}}
}

// Toggle atomically toggles the boolean value and returns the new value.
func (v *Bool) Toggle() bool {
	return v.Update(func(old bool) bool { return !old })
}

// CompareAndSwapBool atomically compares and swaps boolean values.
func (v *Bool) CompareAndSwapBool(old, new bool) bool {
	return v.CompareAndSwap(old, new, func(a, b bool) bool { return a == b })
}

// String is an atomic string value.
type String struct {
	Value[string]
}

// NewString creates a new atomic string with the given initial value.
func NewString(initial string) *String {
	return &String{Value: Value[string]{value: initial}}
}

// CompareAndSwapString atomically compares and swaps string values.
func (v *String) CompareAndSwapString(old, new string) bool {
	return v.CompareAndSwap(old, new, func(a, b string) bool { return a == b })
}
