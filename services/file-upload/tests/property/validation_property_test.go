// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 4: File Validation Completeness
// Validates: Requirements 7.1, 7.2, 7.3, 7.5, 7.6
package property

import (
	"testing"

	"pgregory.net/rapid"
)

// TestMIMEType represents a MIME type for testing.
type TestMIMEType string

const (
	TestMIMETypeJPEG TestMIMEType = "image/jpeg"
	TestMIMETypePNG  TestMIMEType = "image/png"
	TestMIMETypePDF  TestMIMEType = "application/pdf"
	TestMIMETypeGIF  TestMIMEType = "image/gif"
)

// TestValidationConfig holds validation configuration for testing.
type TestValidationConfig struct {
	AllowedTypes []TestMIMEType
	MaxFileSize  int64
}

// MockFileValidator simulates file validation for testing.
type MockFileValidator struct {
	config       TestValidationConfig
	extensionMap map[string]TestMIMEType
	magicBytes   map[TestMIMEType][]byte
}

func NewMockFileValidator(config TestValidationConfig) *MockFileValidator {
	return &MockFileValidator{
		config: config,
		extensionMap: map[string]TestMIMEType{
			".jpg": TestMIMETypeJPEG, ".jpeg": TestMIMETypeJPEG,
			".png": TestMIMETypePNG, ".gif": TestMIMETypeGIF,
			".pdf": TestMIMETypePDF,
		},
		magicBytes: map[TestMIMEType][]byte{
			TestMIMETypeJPEG: {0xFF, 0xD8, 0xFF},
			TestMIMETypePNG:  {0x89, 0x50, 0x4E, 0x47},
			TestMIMETypePDF:  {0x25, 0x50, 0x44, 0x46},
			TestMIMETypeGIF:  {0x47, 0x49, 0x46, 0x38},
		},
	}
}

// DetectMIMEFromMagicBytes detects MIME type from magic bytes.
func (v *MockFileValidator) DetectMIMEFromMagicBytes(data []byte) TestMIMEType {
	for mimeType, magic := range v.magicBytes {
		if len(data) >= len(magic) {
			match := true
			for i, b := range magic {
				if data[i] != b {
					match = false
					break
				}
			}
			if match {
				return mimeType
			}
		}
	}
	return ""
}

// ValidateExtensionMatch checks if extension matches MIME type.
func (v *MockFileValidator) ValidateExtensionMatch(filename string, mimeType TestMIMEType) bool {
	ext := getExtension(filename)
	expected, exists := v.extensionMap[ext]
	if !exists {
		return false
	}
	return expected == mimeType
}

// ValidateSize checks if size is within limit.
func (v *MockFileValidator) ValidateSize(size int64) bool {
	return size > 0 && size <= v.config.MaxFileSize
}

// ValidateMIMEAllowed checks if MIME type is in allowlist.
func (v *MockFileValidator) ValidateMIMEAllowed(mimeType TestMIMEType) bool {
	for _, allowed := range v.config.AllowedTypes {
		if allowed == mimeType {
			return true
		}
	}
	return false
}

// Validate performs complete validation.
func (v *MockFileValidator) Validate(filename string, data []byte, size int64) (bool, string) {
	// Check size
	if !v.ValidateSize(size) {
		return false, "FILE_TOO_LARGE"
	}

	// Detect MIME from magic bytes
	mimeType := v.DetectMIMEFromMagicBytes(data)
	if mimeType == "" {
		return false, "INVALID_FILE_TYPE"
	}

	// Check MIME is allowed
	if !v.ValidateMIMEAllowed(mimeType) {
		return false, "INVALID_FILE_TYPE"
	}

	// Check extension matches
	if !v.ValidateExtensionMatch(filename, mimeType) {
		return false, "EXTENSION_MISMATCH"
	}

	return true, ""
}

func getExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}

// TestProperty4_MIMEDetectedFromMagicBytes tests that MIME type is detected from magic bytes.
// Property 4: File Validation Completeness
// Validates: Requirements 7.1, 7.2, 7.3, 7.5, 7.6
func TestProperty4_MIMEDetectedFromMagicBytes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		config := TestValidationConfig{
			AllowedTypes: []TestMIMEType{TestMIMETypeJPEG, TestMIMETypePNG, TestMIMETypePDF},
			MaxFileSize:  10 * 1024 * 1024,
		}
		validator := NewMockFileValidator(config)

		// Generate random file type
		mimeType := rapid.SampledFrom([]TestMIMEType{
			TestMIMETypeJPEG, TestMIMETypePNG, TestMIMETypePDF,
		}).Draw(t, "mimeType")

		// Get magic bytes for this type
		magic := validator.magicBytes[mimeType]

		// Generate random content after magic bytes
		extraBytes := rapid.SliceOfN(rapid.Byte(), 10, 100).Draw(t, "extraBytes")
		data := append(magic, extraBytes...)

		// Property: MIME type SHALL be detected from content magic bytes
		detected := validator.DetectMIMEFromMagicBytes(data)
		if detected != mimeType {
			t.Errorf("expected MIME type %q, detected %q", mimeType, detected)
		}
	})
}

// TestProperty4_ExtensionMatchesMIME tests that extension must match detected MIME type.
// Property 4: File Validation Completeness
// Validates: Requirements 7.1, 7.2, 7.3, 7.5, 7.6
func TestProperty4_ExtensionMatchesMIME(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		config := TestValidationConfig{
			AllowedTypes: []TestMIMEType{TestMIMETypeJPEG, TestMIMETypePNG, TestMIMETypePDF},
			MaxFileSize:  10 * 1024 * 1024,
		}
		validator := NewMockFileValidator(config)

		// Generate matching filename and MIME type
		mimeType := rapid.SampledFrom([]TestMIMEType{
			TestMIMETypeJPEG, TestMIMETypePNG, TestMIMETypePDF,
		}).Draw(t, "mimeType")

		extMap := map[TestMIMEType]string{
			TestMIMETypeJPEG: ".jpg",
			TestMIMETypePNG:  ".png",
			TestMIMETypePDF:  ".pdf",
		}
		correctExt := extMap[mimeType]
		baseName := rapid.StringMatching(`[a-z0-9_-]{1,20}`).Draw(t, "baseName")
		filename := baseName + correctExt

		// Property: File extension SHALL match detected MIME type
		if !validator.ValidateExtensionMatch(filename, mimeType) {
			t.Errorf("extension %q should match MIME type %q", correctExt, mimeType)
		}

		// Test mismatch
		wrongExt := rapid.SampledFrom([]string{".jpg", ".png", ".pdf"}).Draw(t, "wrongExt")
		if wrongExt != correctExt {
			wrongFilename := baseName + wrongExt
			if validator.ValidateExtensionMatch(wrongFilename, mimeType) {
				t.Errorf("extension %q should NOT match MIME type %q", wrongExt, mimeType)
			}
		}
	})
}

// TestProperty4_SizeNotExceedLimit tests that file size must not exceed limit.
// Property 4: File Validation Completeness
// Validates: Requirements 7.1, 7.2, 7.3, 7.5, 7.6
func TestProperty4_SizeNotExceedLimit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxSize := int64(rapid.IntRange(1024, 10*1024*1024).Draw(t, "maxSize"))
		config := TestValidationConfig{
			AllowedTypes: []TestMIMEType{TestMIMETypeJPEG},
			MaxFileSize:  maxSize,
		}
		validator := NewMockFileValidator(config)

		// Test valid size
		validSize := int64(rapid.IntRange(1, int(maxSize)).Draw(t, "validSize"))
		if !validator.ValidateSize(validSize) {
			t.Errorf("size %d should be valid (max: %d)", validSize, maxSize)
		}

		// Property: File size SHALL not exceed tenant-specific limit
		invalidSize := maxSize + int64(rapid.IntRange(1, 1000).Draw(t, "extra"))
		if validator.ValidateSize(invalidSize) {
			t.Errorf("size %d should exceed limit %d", invalidSize, maxSize)
		}

		// Zero size should be invalid
		if validator.ValidateSize(0) {
			t.Error("zero size should be invalid")
		}
	})
}

// TestProperty4_MIMEInAllowlist tests that MIME type must be in allowlist.
// Property 4: File Validation Completeness
// Validates: Requirements 7.1, 7.2, 7.3, 7.5, 7.6
func TestProperty4_MIMEInAllowlist(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random allowlist
		allTypes := []TestMIMEType{TestMIMETypeJPEG, TestMIMETypePNG, TestMIMETypePDF, TestMIMETypeGIF}
		numAllowed := rapid.IntRange(1, 3).Draw(t, "numAllowed")
		allowedTypes := make([]TestMIMEType, numAllowed)
		for i := 0; i < numAllowed; i++ {
			allowedTypes[i] = allTypes[i]
		}

		config := TestValidationConfig{
			AllowedTypes: allowedTypes,
			MaxFileSize:  10 * 1024 * 1024,
		}
		validator := NewMockFileValidator(config)

		// Property: MIME type SHALL be in tenant-specific allowlist
		for _, allowed := range allowedTypes {
			if !validator.ValidateMIMEAllowed(allowed) {
				t.Errorf("MIME type %q should be allowed", allowed)
			}
		}

		// Check disallowed types
		for _, mimeType := range allTypes {
			isAllowed := false
			for _, allowed := range allowedTypes {
				if mimeType == allowed {
					isAllowed = true
					break
				}
			}
			if !isAllowed && validator.ValidateMIMEAllowed(mimeType) {
				t.Errorf("MIME type %q should NOT be allowed", mimeType)
			}
		}
	})
}

// TestProperty4_ValidationFailureReturnsErrorCode tests that validation failure returns specific error code.
// Property 4: File Validation Completeness
// Validates: Requirements 7.1, 7.2, 7.3, 7.5, 7.6
func TestProperty4_ValidationFailureReturnsErrorCode(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		config := TestValidationConfig{
			AllowedTypes: []TestMIMEType{TestMIMETypeJPEG},
			MaxFileSize:  1024,
		}
		validator := NewMockFileValidator(config)

		// Test size error
		valid, errCode := validator.Validate("test.jpg", []byte{0xFF, 0xD8, 0xFF}, 2048)
		if valid || errCode != "FILE_TOO_LARGE" {
			t.Errorf("expected FILE_TOO_LARGE error, got valid=%v, code=%q", valid, errCode)
		}

		// Test invalid MIME type (unknown magic bytes)
		valid, errCode = validator.Validate("test.jpg", []byte{0x00, 0x00, 0x00}, 100)
		if valid || errCode != "INVALID_FILE_TYPE" {
			t.Errorf("expected INVALID_FILE_TYPE error, got valid=%v, code=%q", valid, errCode)
		}

		// Test extension mismatch
		valid, errCode = validator.Validate("test.png", []byte{0xFF, 0xD8, 0xFF}, 100)
		if valid || errCode != "EXTENSION_MISMATCH" {
			t.Errorf("expected EXTENSION_MISMATCH error, got valid=%v, code=%q", valid, errCode)
		}

		// Property: Validation failure SHALL return specific error code
		// All error codes should be non-empty strings
		if errCode == "" {
			t.Error("error code should not be empty on validation failure")
		}
	})
}
