// Package testutil provides a generic mock emitter for testing.
package testutil

import (
	"sync"
)

// MockEmitter is a generic mock event emitter for testing.
type MockEmitter[T any] struct {
	mu     sync.RWMutex
	events []T
}

// NewMockEmitter creates a new mock emitter.
func NewMockEmitter[T any]() *MockEmitter[T] {
	return &MockEmitter[T]{
		events: make([]T, 0),
	}
}

// Emit records an event.
func (m *MockEmitter[T]) Emit(event T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

// Events returns all recorded events.
func (m *MockEmitter[T]) Events() []T {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]T, len(m.events))
	copy(result, m.events)
	return result
}

// Filter returns events matching the predicate.
func (m *MockEmitter[T]) Filter(predicate func(T) bool) []T {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []T
	for _, e := range m.events {
		if predicate(e) {
			result = append(result, e)
		}
	}
	return result
}

// Clear removes all recorded events.
func (m *MockEmitter[T]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = make([]T, 0)
}

// Len returns the number of recorded events.
func (m *MockEmitter[T]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.events)
}

// First returns the first event or zero value if empty.
func (m *MockEmitter[T]) First() (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.events) == 0 {
		var zero T
		return zero, false
	}
	return m.events[0], true
}

// Last returns the last event or zero value if empty.
func (m *MockEmitter[T]) Last() (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.events) == 0 {
		var zero T
		return zero, false
	}
	return m.events[len(m.events)-1], true
}

// At returns the event at the given index.
func (m *MockEmitter[T]) At(index int) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if index < 0 || index >= len(m.events) {
		var zero T
		return zero, false
	}
	return m.events[index], true
}

// Contains checks if any event matches the predicate.
func (m *MockEmitter[T]) Contains(predicate func(T) bool) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.events {
		if predicate(e) {
			return true
		}
	}
	return false
}

// Count returns the number of events matching the predicate.
func (m *MockEmitter[T]) Count(predicate func(T) bool) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, e := range m.events {
		if predicate(e) {
			count++
		}
	}
	return count
}

// ForEach iterates over all events.
func (m *MockEmitter[T]) ForEach(fn func(T)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.events {
		fn(e)
	}
}

// IsEmpty returns true if no events have been recorded.
func (m *MockEmitter[T]) IsEmpty() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.events) == 0
}

// WaitForEvents waits until at least n events are recorded.
// Returns immediately if already have enough events.
// Note: This is a simple polling implementation for testing.
func (m *MockEmitter[T]) WaitForEvents(n int, check func() bool) bool {
	for i := 0; i < 100; i++ {
		if m.Len() >= n {
			return true
		}
		if check != nil && !check() {
			return false
		}
	}
	return m.Len() >= n
}
