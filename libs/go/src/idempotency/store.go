// Package idempotency provides idempotency key handling.
package idempotency

import (
	"context"
	"sync"
	"time"
)

// Entry represents a cached idempotency response.
type Entry struct {
	Key        string
	Response   []byte
	StatusCode int
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

// Store is the interface for idempotency storage.
type Store interface {
	Get(ctx context.Context, key string) (*Entry, error)
	Set(ctx context.Context, entry *Entry) error
	Delete(ctx context.Context, key string) error
	Lock(ctx context.Context, key string) (bool, error)
	Unlock(ctx context.Context, key string) error
}

// MemoryStore is an in-memory idempotency store.
type MemoryStore struct {
	mu      sync.RWMutex
	entries map[string]*Entry
	locks   map[string]bool
	ttl     time.Duration
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore(ttl time.Duration) *MemoryStore {
	return &MemoryStore{
		entries: make(map[string]*Entry),
		locks:   make(map[string]bool),
		ttl:     ttl,
	}
}

// Get retrieves an entry by key.
func (s *MemoryStore) Get(ctx context.Context, key string) (*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[key]
	if !ok {
		return nil, nil
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil, nil
	}

	return entry, nil
}

// Set stores an entry.
func (s *MemoryStore) Set(ctx context.Context, entry *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry.ExpiresAt.IsZero() {
		entry.ExpiresAt = time.Now().Add(s.ttl)
	}
	entry.CreatedAt = time.Now()
	s.entries[entry.Key] = entry
	return nil
}

// Delete removes an entry.
func (s *MemoryStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
	return nil
}

// Lock acquires a lock for a key.
func (s *MemoryStore) Lock(ctx context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.locks[key] {
		return false, nil
	}
	s.locks[key] = true
	return true, nil
}

// Unlock releases a lock for a key.
func (s *MemoryStore) Unlock(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.locks, key)
	return nil
}

// Cleanup removes expired entries.
func (s *MemoryStore) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	removed := 0
	for key, entry := range s.entries {
		if now.After(entry.ExpiresAt) {
			delete(s.entries, key)
			removed++
		}
	}
	return removed
}
