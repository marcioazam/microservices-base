// Package testutil provides test utilities for file-upload service.
package testutil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateID generates a random ID.
func GenerateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateTenantID generates a random tenant ID.
func GenerateTenantID() string {
	return "tenant-" + GenerateID()[:8]
}

// GenerateUserID generates a random user ID.
func GenerateUserID() string {
	return "user-" + GenerateID()[:8]
}

// GenerateCorrelationID generates a random correlation ID.
func GenerateCorrelationID() string {
	return GenerateID() + GenerateID()
}

// GenerateHash generates a SHA-256 hash of data.
func GenerateHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// GenerateRandomBytes generates random bytes of specified size.
func GenerateRandomBytes(size int) []byte {
	b := make([]byte, size)
	rand.Read(b)
	return b
}

// GenerateFilename generates a random filename with extension.
func GenerateFilename(ext string) string {
	return fmt.Sprintf("file-%s%s", GenerateID()[:8], ext)
}

// GenerateStoragePath generates a storage path.
func GenerateStoragePath(tenantID, hash, filename string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s",
		tenantID,
		now.Format("2006"),
		now.Format("01"),
		now.Format("02"),
		hash,
		filename,
	)
}

// TestFileMetadata represents file metadata for testing.
type TestFileMetadata struct {
	ID           string
	TenantID     string
	UserID       string
	Filename     string
	OriginalName string
	MIMEType     string
	Size         int64
	Hash         string
	StoragePath  string
	Status       string
	ScanStatus   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// GenerateTestFileMetadata generates test file metadata.
func GenerateTestFileMetadata() *TestFileMetadata {
	tenantID := GenerateTenantID()
	filename := GenerateFilename(".pdf")
	data := GenerateRandomBytes(1024)
	hash := GenerateHash(data)

	return &TestFileMetadata{
		ID:           GenerateID(),
		TenantID:     tenantID,
		UserID:       GenerateUserID(),
		Filename:     filename,
		OriginalName: filename,
		MIMEType:     "application/pdf",
		Size:         int64(len(data)),
		Hash:         hash,
		StoragePath:  GenerateStoragePath(tenantID, hash, filename),
		Status:       "ready",
		ScanStatus:   "clean",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
}

// TestChunkSession represents a chunk session for testing.
type TestChunkSession struct {
	ID             string
	TenantID       string
	UserID         string
	Filename       string
	TotalSize      int64
	ChunkSize      int64
	TotalChunks    int
	UploadedChunks []int
	Status         string
	CreatedAt      time.Time
	ExpiresAt      time.Time
}

// GenerateTestChunkSession generates a test chunk session.
func GenerateTestChunkSession(totalSize, chunkSize int64) *TestChunkSession {
	totalChunks := int(totalSize / chunkSize)
	if totalSize%chunkSize != 0 {
		totalChunks++
	}

	return &TestChunkSession{
		ID:             GenerateID(),
		TenantID:       GenerateTenantID(),
		UserID:         GenerateUserID(),
		Filename:       GenerateFilename(".pdf"),
		TotalSize:      totalSize,
		ChunkSize:      chunkSize,
		TotalChunks:    totalChunks,
		UploadedChunks: []int{},
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
	}
}
