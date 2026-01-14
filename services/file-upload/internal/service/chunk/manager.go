// Package chunk provides chunked upload management using Cache Service.
package chunk

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CacheClient defines the interface for cache operations.
type CacheClient interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// Session represents a chunked upload session.
type Session struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	UserID         string    `json:"user_id"`
	Filename       string    `json:"filename"`
	TotalSize      int64     `json:"total_size"`
	ChunkSize      int64     `json:"chunk_size"`
	TotalChunks    int       `json:"total_chunks"`
	UploadedChunks []int     `json:"uploaded_chunks"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// ChunkData represents a single chunk.
type ChunkData struct {
	Index    int
	Content  io.Reader
	Size     int64
	Checksum string
}

// AssembledFile represents the final assembled file.
type AssembledFile struct {
	Content  io.Reader
	Size     int64
	Hash     string
	Filename string
}

// Config holds manager configuration.
type Config struct {
	ChunkSize     int64
	SessionExpiry time.Duration
	KeyPrefix     string
}

// Manager handles chunked uploads.
type Manager struct {
	cache      CacheClient
	chunkSize  int64
	sessionExp time.Duration
	keyPrefix  string
	mu         sync.RWMutex
}

// NewManager creates a new chunk manager.
func NewManager(cache CacheClient, cfg Config) *Manager {
	keyPrefix := cfg.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = "file-upload:chunk"
	}
	sessionExp := cfg.SessionExpiry
	if sessionExp == 0 {
		sessionExp = 24 * time.Hour
	}

	return &Manager{
		cache:      cache,
		chunkSize:  cfg.ChunkSize,
		sessionExp: sessionExp,
		keyPrefix:  keyPrefix,
	}
}

// CreateSession creates a new upload session.
func (m *Manager) CreateSession(ctx context.Context, tenantID, userID, filename string, totalSize int64) (*Session, error) {
	sessionID := uuid.New().String()

	totalChunks := int(totalSize / m.chunkSize)
	if totalSize%m.chunkSize != 0 {
		totalChunks++
	}

	session := &Session{
		ID:             sessionID,
		TenantID:       tenantID,
		UserID:         userID,
		Filename:       filename,
		TotalSize:      totalSize,
		ChunkSize:      m.chunkSize,
		TotalChunks:    totalChunks,
		UploadedChunks: []int{},
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      time.Now().UTC().Add(m.sessionExp),
	}

	if err := m.saveSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession retrieves session state.
func (m *Manager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	key := m.sessionKey(sessionID)
	data, err := m.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if data == nil {
		return nil, ErrSessionNotFound
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	return &session, nil
}

// UploadChunk stores a chunk with SHA-256 verification.
func (m *Manager) UploadChunk(ctx context.Context, sessionID string, chunk *ChunkData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Validate chunk index
	if chunk.Index < 0 || chunk.Index >= session.TotalChunks {
		return ErrInvalidChunk
	}

	// Check for duplicate
	for _, idx := range session.UploadedChunks {
		if idx == chunk.Index {
			return ErrDuplicateChunk
		}
	}

	// Read chunk content
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, chunk.Content); err != nil {
		return err
	}

	// Verify SHA-256 checksum
	if chunk.Checksum != "" {
		computed := computeSHA256(buf.Bytes())
		if computed != chunk.Checksum {
			return ErrChecksumMismatch
		}
	}

	// Store chunk data
	chunkKey := m.chunkKey(sessionID, chunk.Index)
	if err := m.cache.Set(ctx, chunkKey, buf.Bytes(), m.sessionExp); err != nil {
		return fmt.Errorf("failed to store chunk: %w", err)
	}

	// Update session
	session.UploadedChunks = append(session.UploadedChunks, chunk.Index)
	if err := m.saveSession(ctx, session); err != nil {
		return err
	}

	return nil
}

// CompleteUpload assembles chunks and returns the final file.
func (m *Manager) CompleteUpload(ctx context.Context, sessionID string) (*AssembledFile, error) {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if len(session.UploadedChunks) != session.TotalChunks {
		return nil, ErrIncompleteUpload
	}

	// Assemble chunks in order
	var assembled bytes.Buffer
	for i := 0; i < session.TotalChunks; i++ {
		chunkKey := m.chunkKey(sessionID, i)
		data, err := m.cache.Get(ctx, chunkKey)
		if err != nil || data == nil {
			return nil, fmt.Errorf("failed to read chunk %d: %w", i, err)
		}
		assembled.Write(data)
	}

	// Compute hash of assembled file
	fileHash := computeSHA256(assembled.Bytes())

	// Mark session as completed
	session.Status = "completed"
	m.saveSession(ctx, session)

	return &AssembledFile{
		Content:  bytes.NewReader(assembled.Bytes()),
		Size:     int64(assembled.Len()),
		Hash:     fileHash,
		Filename: session.Filename,
	}, nil
}

// AbortUpload cancels session and cleans up.
func (m *Manager) AbortUpload(ctx context.Context, sessionID string) error {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil // Already cleaned up
	}

	// Delete all chunks
	for i := 0; i < session.TotalChunks; i++ {
		chunkKey := m.chunkKey(sessionID, i)
		m.cache.Delete(ctx, chunkKey)
	}

	// Delete session
	sessionKey := m.sessionKey(sessionID)
	return m.cache.Delete(ctx, sessionKey)
}

func (m *Manager) saveSession(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	key := m.sessionKey(session.ID)
	return m.cache.Set(ctx, key, data, m.sessionExp)
}

func (m *Manager) sessionKey(sessionID string) string {
	return fmt.Sprintf("%s:session:%s", m.keyPrefix, sessionID)
}

func (m *Manager) chunkKey(sessionID string, index int) string {
	return fmt.Sprintf("%s:data:%s:%d", m.keyPrefix, sessionID, index)
}

func computeSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Errors
var (
	ErrSessionNotFound  = &ChunkError{Code: "SESSION_NOT_FOUND", Message: "session not found"}
	ErrSessionExpired   = &ChunkError{Code: "SESSION_EXPIRED", Message: "session has expired"}
	ErrInvalidChunk     = &ChunkError{Code: "INVALID_CHUNK", Message: "invalid chunk index"}
	ErrDuplicateChunk   = &ChunkError{Code: "DUPLICATE_CHUNK", Message: "chunk already uploaded"}
	ErrChecksumMismatch = &ChunkError{Code: "CHECKSUM_MISMATCH", Message: "chunk checksum mismatch"}
	ErrIncompleteUpload = &ChunkError{Code: "INCOMPLETE_UPLOAD", Message: "not all chunks uploaded"}
)

// ChunkError represents a chunk operation error.
type ChunkError struct {
	Code    string
	Message string
}

func (e *ChunkError) Error() string {
	return e.Code + ": " + e.Message
}
