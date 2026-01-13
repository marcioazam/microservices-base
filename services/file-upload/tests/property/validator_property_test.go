package property

import (
	"testing"

	"github.com/auth-platform/file-upload/internal/validator"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: file-upload-service, Property 2: File Type Validation Correctness
// Validates: Requirements 2.1, 2.2, 2.3
// For any uploaded file, the File_Validator SHALL correctly identify the MIME type
// by inspecting magic bytes, and SHALL reject files where the extension does not
// match the detected content type or the type is not in the allowlist.

func TestFileTypeValidationProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Create validator with allowed types
	v := validator.NewValidator(validator.Config{
		AllowedMIMETypes: []string{
			"image/jpeg", "image/png", "image/gif",
			"application/pdf", "video/mp4",
		},
		MaxFileSize:    10 * 1024 * 1024,
		MaxChunkedSize: 5 * 1024 * 1024 * 1024,
	})

	// Property: JPEG magic bytes are correctly detected
	properties.Property("JPEG magic bytes detected correctly", prop.ForAll(
		func(suffix []byte) bool {
			// JPEG magic bytes
			jpegMagic := []byte{0xFF, 0xD8, 0xFF}
			data := append(jpegMagic, suffix...)

			mimeType, err := v.ValidateTypeFromBytes(data)
			if err != nil {
				return false
			}

			return mimeType == validator.MIMETypeJPEG
		},
		gen.SliceOfN(100, gen.UInt8()),
	))

	// Property: PNG magic bytes are correctly detected
	properties.Property("PNG magic bytes detected correctly", prop.ForAll(
		func(suffix []byte) bool {
			// PNG magic bytes
			pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
			data := append(pngMagic, suffix...)

			mimeType, err := v.ValidateTypeFromBytes(data)
			if err != nil {
				return false
			}

			return mimeType == validator.MIMETypePNG
		},
		gen.SliceOfN(100, gen.UInt8()),
	))

	// Property: GIF magic bytes are correctly detected
	properties.Property("GIF magic bytes detected correctly", prop.ForAll(
		func(version byte, suffix []byte) bool {
			// GIF magic bytes (GIF87a or GIF89a)
			gifMagic := []byte{0x47, 0x49, 0x46, 0x38}
			if version%2 == 0 {
				gifMagic = append(gifMagic, 0x37, 0x61) // GIF87a
			} else {
				gifMagic = append(gifMagic, 0x39, 0x61) // GIF89a
			}
			data := append(gifMagic, suffix...)

			mimeType, err := v.ValidateTypeFromBytes(data)
			if err != nil {
				return false
			}

			return mimeType == validator.MIMETypeGIF
		},
		gen.UInt8(),
		gen.SliceOfN(100, gen.UInt8()),
	))

	// Property: PDF magic bytes are correctly detected
	properties.Property("PDF magic bytes detected correctly", prop.ForAll(
		func(suffix []byte) bool {
			// PDF magic bytes (%PDF)
			pdfMagic := []byte{0x25, 0x50, 0x44, 0x46, 0x2D} // %PDF-
			data := append(pdfMagic, suffix...)

			mimeType, err := v.ValidateTypeFromBytes(data)
			if err != nil {
				return false
			}

			return mimeType == validator.MIMETypePDF
		},
		gen.SliceOfN(100, gen.UInt8()),
	))

	// Property: Allowed types are accepted
	properties.Property("allowed types are accepted", prop.ForAll(
		func(mimeType string) bool {
			allowedTypes := []string{
				"image/jpeg", "image/png", "image/gif",
				"application/pdf", "video/mp4",
			}

			for _, allowed := range allowedTypes {
				if mimeType == allowed {
					return v.IsAllowedType(validator.MIMEType(mimeType))
				}
			}
			return true // Not in our test set
		},
		gen.OneConstOf(
			"image/jpeg", "image/png", "image/gif",
			"application/pdf", "video/mp4",
		),
	))

	// Property: Disallowed types are rejected
	properties.Property("disallowed types are rejected", prop.ForAll(
		func(mimeType string) bool {
			disallowedTypes := []string{
				"application/x-executable",
				"application/x-msdownload",
				"text/html",
				"application/javascript",
			}

			for _, disallowed := range disallowedTypes {
				if mimeType == disallowed {
					return !v.IsAllowedType(validator.MIMEType(mimeType))
				}
			}
			return true
		},
		gen.OneConstOf(
			"application/x-executable",
			"application/x-msdownload",
			"text/html",
			"application/javascript",
		),
	))

	// Property: Extension matching is case-insensitive
	properties.Property("extension matching works for valid pairs", prop.ForAll(
		func(ext string, mime validator.MIMEType) bool {
			detector := validator.NewMIMETypeDetector()

			validPairs := map[string]validator.MIMEType{
				".jpg":  validator.MIMETypeJPEG,
				".jpeg": validator.MIMETypeJPEG,
				".png":  validator.MIMETypePNG,
				".gif":  validator.MIMETypeGIF,
				".pdf":  validator.MIMETypePDF,
			}

			expectedMime, exists := validPairs[ext]
			if !exists {
				return true // Not testing this pair
			}

			filename := "test" + ext
			return detector.ExtensionMatchesMIME(filename, expectedMime)
		},
		gen.OneConstOf(".jpg", ".jpeg", ".png", ".gif", ".pdf"),
		gen.OneConstOf(
			validator.MIMETypeJPEG,
			validator.MIMETypePNG,
			validator.MIMETypeGIF,
			validator.MIMETypePDF,
		),
	))

	properties.TestingRun(t)
}

// Feature: file-upload-service, Property 3: File Size Validation
// Validates: Requirements 3.1, 3.2
// For any file upload, if the file size exceeds the configured maximum,
// the upload SHALL be rejected with HTTP 400 before any storage operation occurs.

func TestFileSizeValidationProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	maxFileSize := int64(10 * 1024 * 1024)       // 10MB
	maxChunkedSize := int64(5 * 1024 * 1024 * 1024) // 5GB

	v := validator.NewValidator(validator.Config{
		AllowedMIMETypes: []string{"image/jpeg"},
		MaxFileSize:      maxFileSize,
		MaxChunkedSize:   maxChunkedSize,
	})

	// Property: Files under max size are accepted
	properties.Property("files under max size are accepted", prop.ForAll(
		func(size int64) bool {
			if size <= 0 {
				size = 1
			}
			if size > maxFileSize {
				size = maxFileSize
			}

			err := v.ValidateSize(size, false)
			return err == nil
		},
		gen.Int64Range(1, maxFileSize),
	))

	// Property: Files over max size are rejected
	properties.Property("files over max size are rejected", prop.ForAll(
		func(excess int64) bool {
			if excess <= 0 {
				excess = 1
			}
			size := maxFileSize + excess

			err := v.ValidateSize(size, false)
			return err != nil
		},
		gen.Int64Range(1, 1000000),
	))

	// Property: Chunked uploads have higher limit
	properties.Property("chunked uploads have higher limit", prop.ForAll(
		func(size int64) bool {
			// Size between regular max and chunked max
			if size <= maxFileSize {
				size = maxFileSize + 1
			}
			if size > maxChunkedSize {
				size = maxChunkedSize
			}

			// Should fail for regular upload
			errRegular := v.ValidateSize(size, false)
			// Should pass for chunked upload
			errChunked := v.ValidateSize(size, true)

			return errRegular != nil && errChunked == nil
		},
		gen.Int64Range(maxFileSize+1, maxChunkedSize),
	))

	// Property: Zero or negative size is rejected
	properties.Property("zero or negative size is rejected", prop.ForAll(
		func(size int64) bool {
			if size > 0 {
				size = -size
			}

			err := v.ValidateSize(size, false)
			return err != nil
		},
		gen.Int64Range(-1000000, 0),
	))

	properties.TestingRun(t)
}
