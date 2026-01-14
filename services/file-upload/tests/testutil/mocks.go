// Package testutil provides test utilities for file-upload service.
package testutil

import (
	"context"
	"sync"
	"time"
)

// MockCacheClient implements a mock cache client for testing.
type MockCacheClient struct {
	data map[string][]byte
	ttls map[string]time.Time
	mu   sync.RWMutex
}

// NewMockCacheClient creates a new mock cache client.
func NewMockCacheClient() *MockCacheClient {
	return &MockCacheClient{
		data: make(map[string][]byte),
		ttls: make(map[string]time.Time),
	}
}

// Get retrieves a value from the cache.
func (c *MockCacheClient) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if expiry, ok := c.ttls[key]; ok && time.Now().After(expiry) {
		return nil, nil
	}

	data, ok := c.data[key]
	if !ok {
		return nil, nil
	}
	return data, nil
}

// Set stores a value in the cache.
func (c *MockCacheClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = value
	c.ttls[key] = time.Now().Add(ttl)
	return nil
}

// Delete removes a value from the cache.
func (c *MockCacheClient) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
	delete(c.ttls, key)
	return nil
}

// Clear clears all data from the cache.
func (c *MockCacheClient) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string][]byte)
	c.ttls = make(map[string]time.Time)
}

// MockStorageClient implements a mock storage client for testing.
type MockStorageClient struct {
	files map[string][]byte
	mu    sync.RWMutex
}

// NewMockStorageClient creates a new mock storage client.
func NewMockStorageClient() *MockStorageClient {
	return &MockStorageClient{
		files: make(map[string][]byte),
	}
}

// Upload stores a file.
func (s *MockStorageClient) Upload(ctx context.Context, path string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.files[path] = data
	return nil
}

// Download retrieves a file.
func (s *MockStorageClient) Download(ctx context.Context, path string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.files[path]
	if !ok {
		return nil, ErrNotFound
	}
	return data, nil
}

// Delete removes a file.
func (s *MockStorageClient) Delete(ctx context.Context, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.files, path)
	return nil
}

// Exists checks if a file exists.
func (s *MockStorageClient) Exists(ctx context.Context, path string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.files[path]
	return ok, nil
}

// MockDatabaseClient implements a mock database client for testing.
type MockDatabaseClient struct {
	records map[string]*TestFileMetadata
	mu      sync.RWMutex
}

// NewMockDatabaseClient creates a new mock database client.
func NewMockDatabaseClient() *MockDatabaseClient {
	return &MockDatabaseClient{
		records: make(map[string]*TestFileMetadata),
	}
}

// Create inserts a record.
func (d *MockDatabaseClient) Create(ctx context.Context, file *TestFileMetadata) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.records[file.ID] = file
	return nil
}

// GetByID retrieves a record by ID.
func (d *MockDatabaseClient) GetByID(ctx context.Context, id string) (*TestFileMetadata, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	file, ok := d.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return file, nil
}

// Update updates a record.
func (d *MockDatabaseClient) Update(ctx context.Context, file *TestFileMetadata) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, ok := d.records[file.ID]; !ok {
		return ErrNotFound
	}
	d.records[file.ID] = file
	return nil
}

// Delete removes a record.
func (d *MockDatabaseClient) Delete(ctx context.Context, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.records, id)
	return nil
}

// MockError represents a mock error.
type MockError struct {
	Code    string
	Message string
}

func (e *MockError) Error() string {
	return e.Code + ": " + e.Message
}

// Common errors
var (
	ErrNotFound = &MockError{Code: "NOT_FOUND", Message: "resource not found"}
)
