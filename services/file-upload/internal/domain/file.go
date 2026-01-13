package domain

import (
	"time"
)

// FileStatus represents the lifecycle status of a file
type FileStatus string

const (
	FileStatusPending    FileStatus = "pending"
	FileStatusUploaded   FileStatus = "uploaded"
	FileStatusProcessing FileStatus = "processing"
	FileStatusReady      FileStatus = "ready"
	FileStatusFailed     FileStatus = "failed"
	FileStatusDeleted    FileStatus = "deleted"
)

// ScanStatus represents the malware scan status
type ScanStatus string

const (
	ScanStatusPending  ScanStatus = "pending"
	ScanStatusScanning ScanStatus = "scanning"
	ScanStatusClean    ScanStatus = "clean"
	ScanStatusInfected ScanStatus = "infected"
	ScanStatusFailed   ScanStatus = "failed"
)

// FileMetadata represents stored file information
type FileMetadata struct {
	ID           string            `json:"id" db:"id"`
	TenantID     string            `json:"tenant_id" db:"tenant_id"`
	UserID       string            `json:"user_id" db:"user_id"`
	Filename     string            `json:"filename" db:"filename"`
	OriginalName string            `json:"original_name" db:"original_name"`
	MIMEType     string            `json:"mime_type" db:"mime_type"`
	Size         int64             `json:"size" db:"size"`
	Hash         string            `json:"hash" db:"hash"`
	StoragePath  string            `json:"storage_path" db:"storage_path"`
	StorageURL   string            `json:"storage_url,omitempty" db:"storage_url"`
	Status       FileStatus        `json:"status" db:"status"`
	ScanStatus   ScanStatus        `json:"scan_status" db:"scan_status"`
	Metadata     map[string]string `json:"metadata,omitempty" db:"metadata"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time        `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsDeleted returns true if the file is soft-deleted
func (f *FileMetadata) IsDeleted() bool {
	return f.DeletedAt != nil
}

// IsReady returns true if the file is ready for access
func (f *FileMetadata) IsReady() bool {
	return f.Status == FileStatusReady && f.ScanStatus == ScanStatusClean
}

// UploadRequest represents an upload API request
type UploadRequest struct {
	Filename    string
	ContentType string
	Size        int64
	Metadata    map[string]string
}

// UploadResponse represents an upload API response
type UploadResponse struct {
	ID          string            `json:"id"`
	Filename    string            `json:"filename"`
	Size        int64             `json:"size"`
	Hash        string            `json:"hash"`
	MIMEType    string            `json:"mime_type"`
	StoragePath string            `json:"storage_path"`
	URL         string            `json:"url,omitempty"`
	Status      FileStatus        `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewUploadResponse creates an UploadResponse from FileMetadata
func NewUploadResponse(f *FileMetadata) *UploadResponse {
	return &UploadResponse{
		ID:          f.ID,
		Filename:    f.Filename,
		Size:        f.Size,
		Hash:        f.Hash,
		MIMEType:    f.MIMEType,
		StoragePath: f.StoragePath,
		URL:         f.StorageURL,
		Status:      f.Status,
		CreatedAt:   f.CreatedAt,
		Metadata:    f.Metadata,
	}
}

// ListRequest represents a list API request
type ListRequest struct {
	TenantID  string
	PageSize  int
	PageToken string
	StartDate *time.Time
	EndDate   *time.Time
	SortBy    string
	SortOrder string
	Status    *FileStatus
}

// ListResponse represents a list API response
type ListResponse struct {
	Files         []*FileMetadata `json:"files"`
	NextPageToken string          `json:"next_page_token,omitempty"`
	TotalCount    int64           `json:"total_count"`
}

// DownloadURL represents a signed download URL response
type DownloadURL struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}
