// Package property contains property-based tests for the cache service.
package property

import (
	"context"
	"sync"
	"time"
)

const (
	PropertyTestIterations = 100
	PropertyTestSeed       = 12345
)

// MockRedisClient is a mock implementation for testing without real Redis.
type MockRedisClient struct {
	mu      sync.RWMutex
	data    map[string]mockEntry
	healthy bool
}

type mockEntry struct {
	value     []byte
	expiresAt time.Time
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data:    make(map[string]mockEntry),
		healthy: true,
	}
}

func (m *MockRedisClient) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.healthy {
		return nil, &mockError{code: "unavailable"}
	}

	entry, ok := m.data[key]
	if !ok {
		return nil, &mockError{code: "not_found"}
	}

	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return nil, &mockError{code: "not_found"}
	}

	return entry.value, nil
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.healthy {
		return &mockError{code: "unavailable"}
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	m.data[key] = mockEntry{value: value, expiresAt: expiresAt}
	return nil
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.healthy {
		return 0, &mockError{code: "unavailable"}
	}

	var count int64
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			count++
		}
	}
	return count, nil
}

func (m *MockRedisClient) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.healthy {
		return nil, &mockError{code: "unavailable"}
	}

	results := make([]interface{}, len(keys))
	for i, key := range keys {
		entry, ok := m.data[key]
		if !ok || (!entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt)) {
			results[i] = nil
			continue
		}
		results[i] = string(entry.value)
	}
	return results, nil
}

func (m *MockRedisClient) SetWithExpire(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.healthy {
		return &mockError{code: "unavailable"}
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	for key, value := range entries {
		m.data[key] = mockEntry{value: value, expiresAt: expiresAt}
	}
	return nil
}

func (m *MockRedisClient) Ping(ctx context.Context) error {
	if !m.healthy {
		return &mockError{code: "unavailable"}
	}
	return nil
}

func (m *MockRedisClient) Close() error { return nil }

func (m *MockRedisClient) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

func (m *MockRedisClient) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]mockEntry)
}

type mockError struct{ code string }

func (e *mockError) Error() string { return e.code }

func isNotFound(err error) bool {
	if me, ok := err.(*mockError); ok {
		return me.code == "not_found"
	}
	return false
}
