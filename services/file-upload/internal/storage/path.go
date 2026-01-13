package storage

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// PathGenerator generates storage paths for files
type PathGenerator struct{}

// NewPathGenerator creates a new path generator
func NewPathGenerator() *PathGenerator {
	return &PathGenerator{}
}

// GeneratePath generates a storage path in the format:
// /{tenant_id}/{year}/{month}/{day}/{file_hash}/{filename}
func (g *PathGenerator) GeneratePath(tenantID, fileHash, filename string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("%s/%d/%02d/%02d/%s/%s",
		sanitizePath(tenantID),
		now.Year(),
		now.Month(),
		now.Day(),
		sanitizePath(fileHash),
		sanitizeFilename(filename),
	)
}

// GeneratePathWithTime generates a storage path with a specific timestamp
func (g *PathGenerator) GeneratePathWithTime(tenantID, fileHash, filename string, t time.Time) string {
	return fmt.Sprintf("%s/%d/%02d/%02d/%s/%s",
		sanitizePath(tenantID),
		t.Year(),
		t.Month(),
		t.Day(),
		sanitizePath(fileHash),
		sanitizeFilename(filename),
	)
}

// ParsePath extracts components from a storage path
func (g *PathGenerator) ParsePath(path string) (*PathComponents, error) {
	parts := strings.Split(path, "/")
	if len(parts) < 6 {
		return nil, fmt.Errorf("invalid path format: %s", path)
	}

	return &PathComponents{
		TenantID: parts[0],
		Year:     parts[1],
		Month:    parts[2],
		Day:      parts[3],
		FileHash: parts[4],
		Filename: parts[5],
	}, nil
}

// PathComponents represents the components of a storage path
type PathComponents struct {
	TenantID string
	Year     string
	Month    string
	Day      string
	FileHash string
	Filename string
}

// sanitizePath removes potentially dangerous characters from path components
func sanitizePath(s string) string {
	// Remove path separators and other dangerous characters
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "..", "_")
	s = strings.TrimSpace(s)
	return s
}

// sanitizeFilename cleans up a filename for safe storage
func sanitizeFilename(filename string) string {
	// Get just the filename without any path
	filename = filepath.Base(filename)
	
	// Remove potentially dangerous characters
	filename = strings.ReplaceAll(filename, "..", "_")
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	
	// Limit length
	if len(filename) > 255 {
		ext := filepath.Ext(filename)
		name := filename[:255-len(ext)]
		filename = name + ext
	}
	
	return filename
}

// ValidatePath checks if a path follows the expected format
func ValidatePath(path string) bool {
	parts := strings.Split(path, "/")
	if len(parts) < 6 {
		return false
	}

	// Check year format (4 digits)
	if len(parts[1]) != 4 {
		return false
	}

	// Check month format (2 digits, 01-12)
	if len(parts[2]) != 2 {
		return false
	}

	// Check day format (2 digits, 01-31)
	if len(parts[3]) != 2 {
		return false
	}

	return true
}

// GenerateChunkPath generates a path for storing chunks
func GenerateChunkPath(sessionID string, chunkIndex int) string {
	return fmt.Sprintf("chunks/%s/%d", sessionID, chunkIndex)
}

// GenerateTempPath generates a temporary path for processing
func GenerateTempPath(tenantID, fileID string) string {
	return fmt.Sprintf("temp/%s/%s", tenantID, fileID)
}
