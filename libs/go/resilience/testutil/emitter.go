package testutil

import (
	"sync"

	"github.com/auth-platform/libs/go/resilience"
)

// MockEventEmitter is a test implementation of EventEmitter.
type MockEventEmitter struct {
	mu     sync.Mutex
	events []resilience.Event
	audits []resilience.AuditEvent
}

// NewMockEventEmitter creates a new mock event emitter.
func NewMockEventEmitter() *MockEventEmitter {
	return &MockEventEmitter{
		events: make([]resilience.Event, 0),
		audits: make([]resilience.AuditEvent, 0),
	}
}

// Emit records a resilience event.
func (m *MockEventEmitter) Emit(event resilience.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

// EmitAudit records an audit event.
func (m *MockEventEmitter) EmitAudit(event resilience.AuditEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audits = append(m.audits, event)
}

// GetEvents returns all recorded events.
func (m *MockEventEmitter) GetEvents() []resilience.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]resilience.Event, len(m.events))
	copy(result, m.events)
	return result
}

// GetAuditEvents returns all recorded audit events.
func (m *MockEventEmitter) GetAuditEvents() []resilience.AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]resilience.AuditEvent, len(m.audits))
	copy(result, m.audits)
	return result
}

// Clear removes all recorded events.
func (m *MockEventEmitter) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = make([]resilience.Event, 0)
	m.audits = make([]resilience.AuditEvent, 0)
}

// GetStateChangeEvents returns only circuit state change events.
func (m *MockEventEmitter) GetStateChangeEvents() []resilience.Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []resilience.Event
	for _, e := range m.events {
		if e.Type == resilience.EventCircuitStateChange {
			result = append(result, e)
		}
	}
	return result
}

// EventCount returns the number of recorded events.
func (m *MockEventEmitter) EventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

// AuditEventCount returns the number of recorded audit events.
func (m *MockEventEmitter) AuditEventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.audits)
}
