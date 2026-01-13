package validator

import (
	"bytes"
	"io"

	"github.com/auth-platform/file-upload/internal/domain"
)

// Validator validates uploaded files
type Validator struct {
	detector        *MIMETypeDetector
	allowedTypes    map[MIMEType]bool
	maxFileSize     int64
	maxChunkedSize  int64
}

// Config holds validator configuration
type Config struct {
	AllowedMIMETypes []string
	MaxFileSize      int64
	MaxChunkedSize   int64
}

// NewValidator creates a new file validator
func NewValidator(cfg Config) *Validator {
	allowedTypes := make(map[MIMEType]bool)
	for _, t := range cfg.AllowedMIMETypes {
		allowedTypes[MIMEType(t)] = true
	}

	return &Validator{
		detector:       NewMIMETypeDetector(),
		allowedTypes:   allowedTypes,
		maxFileSize:    cfg.MaxFileSize,
		maxChunkedSize: cfg.MaxChunkedSize,
	}
}

// ValidationResult contains the result of file validation
type ValidationResult struct {
	Valid    bool
	MIMEType MIMEType
	Error    error
}

// ValidateFile performs complete file validation
func (v *Validator) ValidateFile(filename string, content io.Reader, size int64, isChunked bool) (*ValidationResult, error) {
	// Validate size first (before reading content)
	if err := v.ValidateSize(size, isChunked); err != nil {
		return &ValidationResult{Valid: false, Error: err}, nil
	}

	// Read content for MIME type detection
	// We need to buffer the first bytes for detection
	buf := make([]byte, 512)
	n, err := content.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	buf = buf[:n]

	// Detect MIME type from content
	mimeType, err := v.detector.DetectFromBytes(buf)
	if err != nil {
		return nil, err
	}

	// Validate MIME type is allowed
	if err := v.validateMIMEType(mimeType); err != nil {
		return &ValidationResult{Valid: false, MIMEType: mimeType, Error: err}, nil
	}

	// Validate extension matches content
	if err := v.ValidateExtension(filename, mimeType); err != nil {
		return &ValidationResult{Valid: false, MIMEType: mimeType, Error: err}, nil
	}

	return &ValidationResult{Valid: true, MIMEType: mimeType}, nil
}

// ValidateType checks file type using magic bytes
func (v *Validator) ValidateType(content io.Reader) (MIMEType, error) {
	return v.detector.DetectFromContent(content)
}

// ValidateTypeFromBytes validates type from byte slice
func (v *Validator) ValidateTypeFromBytes(data []byte) (MIMEType, error) {
	return v.detector.DetectFromBytes(data)
}

// ValidateSize checks file size against limits
func (v *Validator) ValidateSize(size int64, isChunked bool) error {
	maxSize := v.maxFileSize
	if isChunked {
		maxSize = v.maxChunkedSize
	}

	if size > maxSize {
		return domain.NewDomainError(
			domain.ErrCodeFileTooLarge,
			"file size exceeds maximum allowed",
			nil,
		)
	}

	if size <= 0 {
		return domain.NewDomainError(
			domain.ErrCodeFileTooLarge,
			"file size must be greater than zero",
			nil,
		)
	}

	return nil
}

// ValidateExtension checks extension matches content type
func (v *Validator) ValidateExtension(filename string, mimeType MIMEType) error {
	if mimeType == "" {
		return domain.NewDomainError(
			domain.ErrCodeInvalidFileType,
			"unable to detect file type",
			nil,
		)
	}

	if !v.detector.ExtensionMatchesMIME(filename, mimeType) {
		return domain.NewDomainError(
			domain.ErrCodeExtensionMismatch,
			"file extension does not match content type",
			nil,
		)
	}

	return nil
}

// IsAllowedType checks if MIME type is in allowlist
func (v *Validator) IsAllowedType(mimeType MIMEType) bool {
	return v.allowedTypes[mimeType]
}

// validateMIMEType validates that the MIME type is allowed
func (v *Validator) validateMIMEType(mimeType MIMEType) error {
	if mimeType == "" {
		return domain.NewDomainError(
			domain.ErrCodeInvalidFileType,
			"unable to detect file type",
			nil,
		)
	}

	if !v.IsAllowedType(mimeType) {
		return domain.NewDomainError(
			domain.ErrCodeInvalidFileType,
			"file type is not allowed",
			nil,
		)
	}

	return nil
}

// GetAllowedTypes returns the list of allowed MIME types
func (v *Validator) GetAllowedTypes() []MIMEType {
	types := make([]MIMEType, 0, len(v.allowedTypes))
	for t := range v.allowedTypes {
		types = append(types, t)
	}
	return types
}

// ValidateWithBuffer validates file and returns buffered content for reuse
func (v *Validator) ValidateWithBuffer(filename string, content io.Reader, size int64, isChunked bool) (*ValidationResult, io.Reader, error) {
	// Validate size first
	if err := v.ValidateSize(size, isChunked); err != nil {
		return &ValidationResult{Valid: false, Error: err}, nil, nil
	}

	// Read all content into buffer
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, content); err != nil {
		return nil, nil, err
	}

	data := buf.Bytes()

	// Detect MIME type
	mimeType, err := v.detector.DetectFromBytes(data)
	if err != nil {
		return nil, nil, err
	}

	// Validate MIME type
	if err := v.validateMIMEType(mimeType); err != nil {
		return &ValidationResult{Valid: false, MIMEType: mimeType, Error: err}, nil, nil
	}

	// Validate extension
	if err := v.ValidateExtension(filename, mimeType); err != nil {
		return &ValidationResult{Valid: false, MIMEType: mimeType, Error: err}, nil, nil
	}

	// Return buffered reader for reuse
	return &ValidationResult{Valid: true, MIMEType: mimeType}, bytes.NewReader(data), nil
}
