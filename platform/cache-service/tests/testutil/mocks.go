package testutil

import (
	"context"
	"sync"
	"time"

	"github.com/auth-platform/cache-service/internal/cache"
)

// MockCacheService is a mock implementation of cache.Service.
type MockCacheService struct {
	mu      sync.RWMutex
	data    map[string]map[string][]byte
	healthy bool
}

// NewMockCacheService creates a new mock cache service.
func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		data:    make(map[string]map[string][]byte),
		healthy: true,
	}
}

// Get retrieves a value from the mock cache.
func (m *MockCacheService) Get(ctx context.Context, namespace, key string) (*cache.Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if ns, ok := m.data[namespace]; ok {
		if val, ok := ns[key]; ok {
			return &cache.Entry{
				Value:  val,
				Source: cache.SourceLocal,
			}, nil
		}
	}
	return nil, cache.ErrNotFound
}

// Set stores a value in the mock cache.
func (m *MockCacheService) Set(ctx context.Context, namespace, key string, value []byte, ttl time.Duration, opts ...cache.SetOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.data[namespace]; !ok {
		m.data[namespace] = make(map[string][]byte)
	}
	m.data[namespace][key] = value
	return nil
}

// Delete removes a value from the mock cache.
func (m *MockCacheService) Delete(ctx context.Context, namespace, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ns, ok := m.data[namespace]; ok {
		if _, ok := ns[key]; ok {
			delete(ns, key)
			return true, nil
		}
	}
	return false, nil
}

// BatchGet retrieves multiple values from the mock cache.
func (m *MockCacheService) BatchGet(ctx context.Context, namespace string, keys []string) (map[string][]byte, []string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	found := make(map[string][]byte)
	var missing []string

	ns, ok := m.data[namespace]
	if !ok {
		return found, keys, nil
	}

	for _, key := range keys {
		if val, ok := ns[key]; ok {
			found[key] = val
		} else {
			missing = append(missing, key)
		}
	}

	return found, missing, nil
}

// BatchSet stores multiple values in the mock cache.
func (m *MockCacheService) BatchSet(ctx context.Context, namespace string, entries map[string][]byte, ttl time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.data[namespace]; !ok {
		m.data[namespace] = make(map[string][]byte)
	}

	for key, value := range entries {
		m.data[namespace][key] = value
	}

	return len(entries), nil
}

// Health returns the health status of the mock cache.
func (m *MockCacheService) Health(ctx context.Context) (*cache.HealthStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &cache.HealthStatus{
		Healthy:     m.healthy,
		RedisStatus: "ok",
		LocalCache:  true,
	}, nil
}

// SetHealthy sets the health status of the mock cache.
func (m *MockCacheService) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

// MockLoggingService is a mock implementation of the logging service.
type MockLoggingService struct {
	mu      sync.Mutex
	entries []MockLogEntry
}

// MockLogEntry represents a captured log entry.
type MockLogEntry struct {
	Level   string
	Message string
	Fields  map[string]string
}

// NewMockLoggingService creates a new mock logging service.
func NewMockLoggingService() *MockLoggingService {
	return &MockLoggingService{
		entries: make([]MockLogEntry, 0),
	}
}

// Log captures a log entry.
func (m *MockLoggingService) Log(level, message string, fields map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, MockLogEntry{
		Level:   level,
		Message: message,
		Fields:  fields,
	})
}

// Entries returns all captured log entries.
func (m *MockLoggingService) Entries() []MockLogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]MockLogEntry{}, m.entries...)
}

// Clear clears all captured log entries.
func (m *MockLoggingService) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = m.entries[:0]
}
