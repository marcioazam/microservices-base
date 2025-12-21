package bulkhead

import (
	"iter"
	"sync"

	"github.com/auth-platform/libs/go/resilience"
)

// Manager manages multiple bulkhead partitions.
type Manager struct {
	mu         sync.RWMutex
	partitions map[string]*Bulkhead
	config     resilience.BulkheadConfig
	emitter    resilience.EventEmitter
}

// NewManager creates a new bulkhead manager.
func NewManager(cfg resilience.BulkheadConfig, emitter resilience.EventEmitter) *Manager {
	return &Manager{
		partitions: make(map[string]*Bulkhead),
		config:     cfg,
		emitter:    emitter,
	}
}

// GetBulkhead returns the bulkhead for a partition.
func (m *Manager) GetBulkhead(partition string) resilience.Bulkhead {
	m.mu.RLock()
	if b, ok := m.partitions[partition]; ok {
		m.mu.RUnlock()
		return b
	}
	m.mu.RUnlock()

	// Create new bulkhead
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if b, ok := m.partitions[partition]; ok {
		return b
	}

	b := New(Config{
		Name:          partition,
		MaxConcurrent: m.config.MaxConcurrent,
		MaxQueue:      m.config.MaxQueue,
		QueueTimeout:  m.config.QueueTimeout,
		EventEmitter:  m.emitter,
	})

	m.partitions[partition] = b
	return b
}

// GetAllMetrics returns metrics for all partitions.
func (m *Manager) GetAllMetrics() map[string]resilience.BulkheadMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]resilience.BulkheadMetrics, len(m.partitions))
	for name, b := range m.partitions {
		result[name] = b.GetMetrics()
	}
	return result
}

// Partitions returns an iterator over all partitions with their metrics.
func (m *Manager) Partitions() iter.Seq2[string, resilience.BulkheadMetrics] {
	return func(yield func(string, resilience.BulkheadMetrics) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		for name, b := range m.partitions {
			if !yield(name, b.GetMetrics()) {
				return
			}
		}
	}
}
