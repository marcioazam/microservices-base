package circuitbreaker

import (
	"sync"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
)

// MockEventEmitter is a test implementation of EventEmitter.
type MockEventEmitter struct {
	mu     sync.Mutex
	events []domain.ResilienceEvent
	audits []domain.AuditEvent
}

// NewMockEventEmitter creates a new mock event emitter.
func NewMockEventEmitter() *MockEventEmitter {
	return &MockEventEmitter{
		events: make([]domain.ResilienceEvent, 0),
		audits: make([]domain.AuditEvent, 0),
	}
}

// Emit records a resilience event.
func (m *MockEventEmitter) Emit(event domain.ResilienceEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

// EmitAudit records an audit event.
func (m *MockEventEmitter) EmitAudit(event domain.AuditEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audits = append(m.audits, event)
}

// GetEvents returns all recorded events.
func (m *MockEventEmitter) GetEvents() []domain.ResilienceEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]domain.ResilienceEvent, len(m.events))
	copy(result, m.events)
	return result
}

// GetAuditEvents returns all recorded audit events.
func (m *MockEventEmitter) GetAuditEvents() []domain.AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]domain.AuditEvent, len(m.audits))
	copy(result, m.audits)
	return result
}

// Clear removes all recorded events.
func (m *MockEventEmitter) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = make([]domain.ResilienceEvent, 0)
	m.audits = make([]domain.AuditEvent, 0)
}

// GetStateChangeEvents returns only circuit state change events.
func (m *MockEventEmitter) GetStateChangeEvents() []domain.ResilienceEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []domain.ResilienceEvent
	for _, e := range m.events {
		if e.Type == domain.EventCircuitStateChange {
			result = append(result, e)
		}
	}
	return result
}
