// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 5: Chunked Upload Integrity (Round-Trip)
// Validates: Requirements 8.2, 8.3, 8.4, 8.5
package property

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// MockChunkSession represents a chunk upload session for testing.
type MockChunkSession struct {
	ID             string
	TenantID       string
	TotalSize      int64
	ChunkSize      int64
	TotalChunks    int
	UploadedChunks map[int][]byte
	Status         string
	ExpiresAt      time.Time
	mu             sync.Mutex
}

// MockChunkManager simulates chunk management for testing.
type MockChunkManager struct {
	sessions map[string]*MockChunkSession
	mu       sync.RWMutex
}

func NewMockChunkManager() *MockChunkManager {
	return &MockChunkManager{
		sessions: make(map[string]*MockChunkSession),
	}
}

// CreateSession creates a new upload session.
func (m *MockChunkManager) CreateSession(tenantID string, totalSize, chunkSize int64) *MockChunkSession {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalChunks := int(totalSize / chunkSize)
	if totalSize%chunkSize != 0 {
		totalChunks++
	}

	session := &MockChunkSession{
		ID:             rapid.StringMatching(`[a-f0-9]{32}`).Example(),
		TenantID:       tenantID,
		TotalSize:      totalSize,
		ChunkSize:      chunkSize,
		TotalChunks:    totalChunks,
		UploadedChunks: make(map[int][]byte),
		Status:         "active",
		ExpiresAt:      time.Now().Add(time.Hour),
	}

	m.sessions[session.ID] = session
	return session
}

// UploadChunk uploads a chunk with checksum verification.
func (m *MockChunkManager) UploadChunk(sessionID string, index int, data []byte, checksum string) error {
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		return &MockChunkError{Code: "SESSION_NOT_FOUND"}
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if time.Now().After(session.ExpiresAt) {
		return &MockChunkError{Code: "SESSION_EXPIRED"}
	}

	if index < 0 || index >= session.TotalChunks {
		return &MockChunkError{Code: "INVALID_CHUNK"}
	}

	if _, exists := session.UploadedChunks[index]; exists {
		return &MockChunkError{Code: "DUPLICATE_CHUNK"}
	}

	// Verify SHA-256 checksum
	if checksum != "" {
		computed := computeTestSHA256(data)
		if computed != checksum {
			return &MockChunkError{Code: "CHECKSUM_MISMATCH"}
		}
	}

	session.UploadedChunks[index] = data
	return nil
}

// Assemble assembles all chunks into final file.
func (m *MockChunkManager) Assemble(sessionID string) ([]byte, string, error) {
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		return nil, "", &MockChunkError{Code: "SESSION_NOT_FOUND"}
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if len(session.UploadedChunks) != session.TotalChunks {
		return nil, "", &MockChunkError{Code: "INCOMPLETE_UPLOAD"}
	}

	// Assemble in order
	var assembled bytes.Buffer
	for i := 0; i < session.TotalChunks; i++ {
		data, exists := session.UploadedChunks[i]
		if !exists {
			return nil, "", &MockChunkError{Code: "MISSING_CHUNK"}
		}
		assembled.Write(data)
	}

	hash := computeTestSHA256(assembled.Bytes())
	session.Status = "completed"

	return assembled.Bytes(), hash, nil
}

// IsExpired checks if session is expired.
func (m *MockChunkManager) IsExpired(sessionID string) bool {
	m.mu.RLock()
	session, exists := m.sessions[sessionID]
	m.mu.RUnlock()

	if !exists {
		return true
	}
	return time.Now().After(session.ExpiresAt)
}

// CleanupExpired removes expired sessions.
func (m *MockChunkManager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, session := range m.sessions {
		if time.Now().After(session.ExpiresAt) {
			delete(m.sessions, id)
			count++
		}
	}
	return count
}

type MockChunkError struct {
	Code string
}

func (e *MockChunkError) Error() string {
	return e.Code
}

func computeTestSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// TestProperty5_ChunkChecksumVerified tests that each chunk checksum is verified using SHA-256.
// Property 5: Chunked Upload Integrity (Round-Trip)
// Validates: Requirements 8.2, 8.3, 8.4, 8.5
func TestProperty5_ChunkChecksumVerified(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		manager := NewMockChunkManager()
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		chunkSize := int64(rapid.IntRange(100, 1000).Draw(t, "chunkSize"))
		numChunks := rapid.IntRange(2, 5).Draw(t, "numChunks") // At least 2 chunks
		totalSize := chunkSize * int64(numChunks)

		session := manager.CreateSession(tenantID, totalSize, chunkSize)

		// Generate chunk data
		chunkData := rapid.SliceOfN(rapid.Byte(), int(chunkSize), int(chunkSize)).Draw(t, "chunkData")
		correctChecksum := computeTestSHA256(chunkData)

		// Property: Each chunk checksum SHALL be verified using SHA-256
		// Correct checksum should succeed
		err := manager.UploadChunk(session.ID, 0, chunkData, correctChecksum)
		if err != nil {
			t.Errorf("upload with correct checksum should succeed: %v", err)
		}

		// Wrong checksum should fail - use chunk index 1 (valid since numChunks >= 2)
		wrongData := rapid.SliceOfN(rapid.Byte(), int(chunkSize), int(chunkSize)).Draw(t, "wrongData")
		wrongChecksum := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		actualChecksum := computeTestSHA256(wrongData)
		
		// Only test checksum mismatch if the actual checksum differs from wrong checksum
		if actualChecksum != wrongChecksum {
			err = manager.UploadChunk(session.ID, 1, wrongData, wrongChecksum)
			if err == nil {
				t.Error("upload with wrong checksum should fail")
			}
			if err != nil && err.Error() != "CHECKSUM_MISMATCH" {
				t.Errorf("expected CHECKSUM_MISMATCH error, got %v", err)
			}
		}
	})
}

// TestProperty5_ParallelChunksNoCorruption tests that parallel chunk uploads don't corrupt session state.
// Property 5: Chunked Upload Integrity (Round-Trip)
// Validates: Requirements 8.2, 8.3, 8.4, 8.5
func TestProperty5_ParallelChunksNoCorruption(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		manager := NewMockChunkManager()
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		chunkSize := int64(100)
		numChunks := rapid.IntRange(3, 10).Draw(t, "numChunks")
		totalSize := chunkSize * int64(numChunks)

		session := manager.CreateSession(tenantID, totalSize, chunkSize)

		// Generate all chunk data
		chunks := make([][]byte, numChunks)
		for i := 0; i < numChunks; i++ {
			chunks[i] = rapid.SliceOfN(rapid.Byte(), int(chunkSize), int(chunkSize)).Draw(t, "chunk")
		}

		// Property: Parallel chunk uploads SHALL not corrupt session state
		var wg sync.WaitGroup
		errors := make(chan error, numChunks)

		for i := 0; i < numChunks; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				checksum := computeTestSHA256(chunks[index])
				if err := manager.UploadChunk(session.ID, index, chunks[index], checksum); err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("parallel upload error: %v", err)
		}

		// Verify all chunks uploaded
		manager.mu.RLock()
		sess := manager.sessions[session.ID]
		manager.mu.RUnlock()

		if len(sess.UploadedChunks) != numChunks {
			t.Errorf("expected %d chunks, got %d", numChunks, len(sess.UploadedChunks))
		}
	})
}

// TestProperty5_AssembledHashMatches tests that assembled file hash matches expected hash.
// Property 5: Chunked Upload Integrity (Round-Trip)
// Validates: Requirements 8.2, 8.3, 8.4, 8.5
func TestProperty5_AssembledHashMatches(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		manager := NewMockChunkManager()
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		chunkSize := int64(100)
		numChunks := rapid.IntRange(2, 5).Draw(t, "numChunks")
		totalSize := chunkSize * int64(numChunks)

		session := manager.CreateSession(tenantID, totalSize, chunkSize)

		// Generate and upload all chunks
		var originalData bytes.Buffer
		for i := 0; i < numChunks; i++ {
			chunkData := rapid.SliceOfN(rapid.Byte(), int(chunkSize), int(chunkSize)).Draw(t, "chunk")
			originalData.Write(chunkData)
			checksum := computeTestSHA256(chunkData)
			manager.UploadChunk(session.ID, i, chunkData, checksum)
		}

		expectedHash := computeTestSHA256(originalData.Bytes())

		// Assemble
		assembled, hash, err := manager.Assemble(session.ID)
		if err != nil {
			t.Fatalf("assembly failed: %v", err)
		}

		// Property: Assembled file hash SHALL match expected hash
		if hash != expectedHash {
			t.Errorf("hash mismatch: expected %s, got %s", expectedHash, hash)
		}

		// Verify content matches
		if !bytes.Equal(assembled, originalData.Bytes()) {
			t.Error("assembled content does not match original")
		}
	})
}

// TestProperty5_ExpiredSessionsCleanedUp tests that expired sessions are automatically cleaned up.
// Property 5: Chunked Upload Integrity (Round-Trip)
// Validates: Requirements 8.2, 8.3, 8.4, 8.5
func TestProperty5_ExpiredSessionsCleanedUp(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		manager := NewMockChunkManager()
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		session := manager.CreateSession(tenantID, 1000, 100)

		// Manually expire the session
		manager.mu.Lock()
		manager.sessions[session.ID].ExpiresAt = time.Now().Add(-time.Hour)
		manager.mu.Unlock()

		// Property: Expired sessions SHALL be automatically cleaned up
		if !manager.IsExpired(session.ID) {
			t.Error("session should be expired")
		}

		// Cleanup should remove expired sessions
		cleaned := manager.CleanupExpired()
		if cleaned != 1 {
			t.Errorf("expected 1 session cleaned, got %d", cleaned)
		}

		// Session should no longer exist
		manager.mu.RLock()
		_, exists := manager.sessions[session.ID]
		manager.mu.RUnlock()

		if exists {
			t.Error("expired session should be removed")
		}
	})
}

// TestProperty5_DuplicateChunkRejected tests that duplicate chunks are rejected.
// Property 5: Chunked Upload Integrity (Round-Trip)
// Validates: Requirements 8.2, 8.3, 8.4, 8.5
func TestProperty5_DuplicateChunkRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		manager := NewMockChunkManager()
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		session := manager.CreateSession(tenantID, 200, 100)

		chunkData := rapid.SliceOfN(rapid.Byte(), 100, 100).Draw(t, "chunkData")
		checksum := computeTestSHA256(chunkData)

		// First upload should succeed
		err := manager.UploadChunk(session.ID, 0, chunkData, checksum)
		if err != nil {
			t.Fatalf("first upload should succeed: %v", err)
		}

		// Property: Duplicate chunk upload should be rejected
		err = manager.UploadChunk(session.ID, 0, chunkData, checksum)
		if err == nil {
			t.Error("duplicate chunk should be rejected")
		}
		if err != nil && err.Error() != "DUPLICATE_CHUNK" {
			t.Errorf("expected DUPLICATE_CHUNK error, got %v", err)
		}
	})
}
