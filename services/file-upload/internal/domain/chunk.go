package domain

import (
	"io"
	"time"
)

// SessionStatus represents the status of a chunk upload session
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusExpired   SessionStatus = "expired"
	SessionStatusAborted   SessionStatus = "aborted"
)

// ChunkSession represents an upload session for chunked uploads
type ChunkSession struct {
	ID             string        `json:"id" db:"id"`
	TenantID       string        `json:"tenant_id" db:"tenant_id"`
	UserID         string        `json:"user_id" db:"user_id"`
	Filename       string        `json:"filename" db:"filename"`
	TotalSize      int64         `json:"total_size" db:"total_size"`
	ChunkSize      int64         `json:"chunk_size" db:"chunk_size"`
	TotalChunks    int           `json:"total_chunks" db:"total_chunks"`
	UploadedChunks []int         `json:"uploaded_chunks" db:"uploaded_chunks"`
	Status         SessionStatus `json:"status" db:"status"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
	ExpiresAt      time.Time     `json:"expires_at" db:"expires_at"`
	CompletedAt    *time.Time    `json:"completed_at,omitempty" db:"completed_at"`
}

// IsExpired returns true if the session has expired
func (s *ChunkSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsComplete returns true if all chunks have been uploaded
func (s *ChunkSession) IsComplete() bool {
	return len(s.UploadedChunks) == s.TotalChunks
}

// MissingChunks returns the indices of chunks that haven't been uploaded
func (s *ChunkSession) MissingChunks() []int {
	uploaded := make(map[int]bool)
	for _, idx := range s.UploadedChunks {
		uploaded[idx] = true
	}

	var missing []int
	for i := 0; i < s.TotalChunks; i++ {
		if !uploaded[i] {
			missing = append(missing, i)
		}
	}
	return missing
}

// Progress returns the upload progress as a percentage
func (s *ChunkSession) Progress() float64 {
	if s.TotalChunks == 0 {
		return 0
	}
	return float64(len(s.UploadedChunks)) / float64(s.TotalChunks) * 100
}

// ChunkData represents a single chunk of data
type ChunkData struct {
	Index    int
	Content  io.Reader
	Size     int64
	Checksum string
}

// CreateSessionRequest represents a request to create a chunk session
type CreateSessionRequest struct {
	TenantID    string
	UserID      string
	Filename    string
	TotalSize   int64
	ChunkSize   int64
	ContentType string
}

// CreateSessionResponse represents the response after creating a session
type CreateSessionResponse struct {
	SessionID   string    `json:"session_id"`
	ChunkSize   int64     `json:"chunk_size"`
	TotalChunks int       `json:"total_chunks"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// AssembledFile represents a file assembled from chunks
type AssembledFile struct {
	Content  io.Reader
	Size     int64
	Hash     string
	Filename string
}

// ChunkUploadRequest represents a request to upload a chunk
type ChunkUploadRequest struct {
	SessionID string
	Index     int
	Content   io.Reader
	Size      int64
	Checksum  string
}

// ChunkUploadResponse represents the response after uploading a chunk
type ChunkUploadResponse struct {
	Index          int     `json:"index"`
	UploadedChunks int     `json:"uploaded_chunks"`
	TotalChunks    int     `json:"total_chunks"`
	Progress       float64 `json:"progress"`
}
