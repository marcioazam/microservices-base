// Package validation provides file validation service.
package validation

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
)

// MIMEType represents a validated MIME type.
type MIMEType string

// Supported MIME types.
const (
	MIMETypeJPEG MIMEType = "image/jpeg"
	MIMETypePNG  MIMEType = "image/png"
	MIMETypeGIF  MIMEType = "image/gif"
	MIMETypePDF  MIMEType = "application/pdf"
	MIMETypeMP4  MIMEType = "video/mp4"
	MIMETypeMOV  MIMEType = "video/quicktime"
	MIMETypeDOCX MIMEType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	MIMETypeXLSX MIMEType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
)

// ValidationRequest contains validation parameters.
type ValidationRequest struct {
	Filename  string
	Content   io.Reader
	Size      int64
	TenantID  string
	IsChunked bool
}

// ValidationResult contains validation result.
type ValidationResult struct {
	Valid       bool
	MIMEType    MIMEType
	DetectedExt string
	Error       *ValidationError
}

// ValidationError represents a validation error.
type ValidationError struct {
	Code    string
	Message string
	Field   string
}

func (e *ValidationError) Error() string {
	return e.Code + ": " + e.Message
}

// TenantConfig holds tenant-specific validation configuration.
type TenantConfig struct {
	AllowedMIMETypes []MIMEType
	MaxFileSize      int64
	MaxChunkedSize   int64
}

// Service provides file validation functionality.
type Service struct {
	defaultConfig TenantConfig
	tenantConfigs map[string]TenantConfig
	extensionMap  map[string]MIMEType
}

// Config holds service configuration.
type Config struct {
	DefaultAllowedTypes []string
	DefaultMaxFileSize  int64
	DefaultMaxChunked   int64
}

// NewService creates a new validation service.
func NewService(cfg Config) *Service {
	allowedTypes := make([]MIMEType, len(cfg.DefaultAllowedTypes))
	for i, t := range cfg.DefaultAllowedTypes {
		allowedTypes[i] = MIMEType(t)
	}

	return &Service{
		defaultConfig: TenantConfig{
			AllowedMIMETypes: allowedTypes,
			MaxFileSize:      cfg.DefaultMaxFileSize,
			MaxChunkedSize:   cfg.DefaultMaxChunked,
		},
		tenantConfigs: make(map[string]TenantConfig),
		extensionMap: map[string]MIMEType{
			".jpg": MIMETypeJPEG, ".jpeg": MIMETypeJPEG,
			".png": MIMETypePNG, ".gif": MIMETypeGIF,
			".pdf": MIMETypePDF, ".mp4": MIMETypeMP4,
			".mov": MIMETypeMOV, ".docx": MIMETypeDOCX,
			".xlsx": MIMETypeXLSX,
		},
	}
}

// SetTenantConfig sets tenant-specific configuration.
func (s *Service) SetTenantConfig(tenantID string, cfg TenantConfig) {
	s.tenantConfigs[tenantID] = cfg
}

// getConfig returns configuration for tenant.
func (s *Service) getConfig(tenantID string) TenantConfig {
	if cfg, ok := s.tenantConfigs[tenantID]; ok {
		return cfg
	}
	return s.defaultConfig
}

// Validate performs complete file validation.
func (s *Service) Validate(ctx context.Context, req *ValidationRequest) (*ValidationResult, error) {
	cfg := s.getConfig(req.TenantID)

	// Validate size
	if err := s.validateSize(req.Size, req.IsChunked, cfg); err != nil {
		return &ValidationResult{Valid: false, Error: err}, nil
	}

	// Read content for MIME detection
	buf := make([]byte, 512)
	n, err := req.Content.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	buf = buf[:n]

	// Detect MIME type from magic bytes
	mimeType, err := s.detectMIMEType(buf)
	if err != nil {
		return nil, err
	}

	// Validate MIME type is allowed
	if err := s.validateMIMEType(mimeType, cfg); err != nil {
		return &ValidationResult{Valid: false, MIMEType: mimeType, Error: err}, nil
	}

	// Validate extension matches content
	if err := s.validateExtension(req.Filename, mimeType); err != nil {
		return &ValidationResult{Valid: false, MIMEType: mimeType, Error: err}, nil
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	return &ValidationResult{Valid: true, MIMEType: mimeType, DetectedExt: ext}, nil
}

// ValidateChunk validates a single chunk.
func (s *Service) ValidateChunk(ctx context.Context, chunk *ChunkData) error {
	if chunk.Size <= 0 {
		return &ValidationError{Code: "INVALID_CHUNK", Message: "chunk size must be positive"}
	}
	if chunk.Index < 0 {
		return &ValidationError{Code: "INVALID_CHUNK", Message: "chunk index must be non-negative"}
	}
	if chunk.Checksum == "" {
		return &ValidationError{Code: "INVALID_CHUNK", Message: "chunk checksum is required"}
	}
	return nil
}

// ValidateWithBuffer validates and returns buffered content.
func (s *Service) ValidateWithBuffer(ctx context.Context, req *ValidationRequest) (*ValidationResult, io.Reader, error) {
	cfg := s.getConfig(req.TenantID)

	// Validate size
	if err := s.validateSize(req.Size, req.IsChunked, cfg); err != nil {
		return &ValidationResult{Valid: false, Error: err}, nil, nil
	}

	// Read all content
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, req.Content); err != nil {
		return nil, nil, err
	}
	data := buf.Bytes()

	// Detect MIME type
	mimeType, err := s.detectMIMEType(data)
	if err != nil {
		return nil, nil, err
	}

	// Validate MIME type
	if err := s.validateMIMEType(mimeType, cfg); err != nil {
		return &ValidationResult{Valid: false, MIMEType: mimeType, Error: err}, nil, nil
	}

	// Validate extension
	if err := s.validateExtension(req.Filename, mimeType); err != nil {
		return &ValidationResult{Valid: false, MIMEType: mimeType, Error: err}, nil, nil
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	return &ValidationResult{Valid: true, MIMEType: mimeType, DetectedExt: ext}, bytes.NewReader(data), nil
}

func (s *Service) validateSize(size int64, isChunked bool, cfg TenantConfig) *ValidationError {
	maxSize := cfg.MaxFileSize
	if isChunked {
		maxSize = cfg.MaxChunkedSize
	}

	if size <= 0 {
		return &ValidationError{Code: "INVALID_SIZE", Message: "file size must be positive", Field: "size"}
	}
	if size > maxSize {
		return &ValidationError{Code: "FILE_TOO_LARGE", Message: "file exceeds maximum size", Field: "size"}
	}
	return nil
}

func (s *Service) detectMIMEType(data []byte) (MIMEType, error) {
	kind, err := filetype.Match(data)
	if err != nil {
		return "", err
	}
	if kind == filetype.Unknown {
		return "", nil
	}
	return MIMEType(kind.MIME.Value), nil
}

func (s *Service) validateMIMEType(mimeType MIMEType, cfg TenantConfig) *ValidationError {
	if mimeType == "" {
		return &ValidationError{Code: "INVALID_FILE_TYPE", Message: "unable to detect file type", Field: "content"}
	}

	for _, allowed := range cfg.AllowedMIMETypes {
		if allowed == mimeType {
			return nil
		}
	}
	return &ValidationError{Code: "INVALID_FILE_TYPE", Message: "file type not allowed", Field: "content"}
}

func (s *Service) validateExtension(filename string, mimeType MIMEType) *ValidationError {
	ext := strings.ToLower(filepath.Ext(filename))
	expected, exists := s.extensionMap[ext]
	if !exists {
		return &ValidationError{Code: "EXTENSION_MISMATCH", Message: "unknown file extension", Field: "filename"}
	}
	if expected != mimeType {
		return &ValidationError{Code: "EXTENSION_MISMATCH", Message: "extension does not match content", Field: "filename"}
	}
	return nil
}

// ChunkData represents chunk validation data.
type ChunkData struct {
	Index    int
	Size     int64
	Checksum string
}
