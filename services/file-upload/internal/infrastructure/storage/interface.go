// Package storage provides file storage abstraction.
package storage

import (
	"context"
	"io"
	"time"
)

// Storage defines the contract for file storage providers.
type Storage interface {
	// Upload uploads a file to storage.
	Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error)

	// Download downloads a file from storage.
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete deletes a file from storage.
	Delete(ctx context.Context, path string) error

	// GeneratePresignedUploadURL generates a presigned URL for direct upload.
	GeneratePresignedUploadURL(ctx context.Context, path string, expiry time.Duration) (string, error)

	// GeneratePresignedDownloadURL generates a presigned URL for download.
	GeneratePresignedDownloadURL(ctx context.Context, path string, expiry time.Duration) (string, error)

	// Exists checks if a file exists.
	Exists(ctx context.Context, path string) (bool, error)

	// GetMetadata retrieves file metadata from storage.
	GetMetadata(ctx context.Context, path string) (*ObjectMetadata, error)
}

// UploadRequest contains upload parameters.
type UploadRequest struct {
	TenantID    string
	FileHash    string
	Filename    string
	Content     io.Reader
	ContentType string
	Size        int64
	Metadata    map[string]string
}

// UploadResult contains upload result.
type UploadResult struct {
	Path      string
	URL       string
	ETag      string
	VersionID string
}

// ObjectMetadata contains storage object metadata.
type ObjectMetadata struct {
	Path         string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
	Metadata     map[string]string
}

// PathBuilder builds tenant-isolated storage paths.
type PathBuilder struct{}

// NewPathBuilder creates a new path builder.
func NewPathBuilder() *PathBuilder {
	return &PathBuilder{}
}

// BuildPath creates a tenant-isolated hierarchical path.
// Format: {tenant_id}/{year}/{month}/{day}/{hash}/{filename}
func (b *PathBuilder) BuildPath(tenantID, hash, filename string) string {
	now := time.Now().UTC()
	return tenantID + "/" +
		now.Format("2006") + "/" +
		now.Format("01") + "/" +
		now.Format("02") + "/" +
		hash + "/" +
		filename
}

// BuildChunkPath creates a path for chunk storage.
// Format: {tenant_id}/chunks/{session_id}/{chunk_index}
func (b *PathBuilder) BuildChunkPath(tenantID, sessionID string, chunkIndex int) string {
	return tenantID + "/chunks/" + sessionID + "/" + string(rune('0'+chunkIndex))
}

// ExtractTenantID extracts tenant ID from a storage path.
func (b *PathBuilder) ExtractTenantID(path string) string {
	if len(path) == 0 {
		return ""
	}
	for i, c := range path {
		if c == '/' {
			return path[:i]
		}
	}
	return path
}

// ValidateTenantAccess validates that the path belongs to the tenant.
func (b *PathBuilder) ValidateTenantAccess(path, tenantID string) bool {
	extractedTenant := b.ExtractTenantID(path)
	return extractedTenant == tenantID
}
