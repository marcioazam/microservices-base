package validator

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
)

// MIMEType represents a validated MIME type
type MIMEType string

// Supported MIME types
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

// MIMETypeDetector detects MIME types from file content
type MIMETypeDetector struct {
	// extensionMap maps file extensions to expected MIME types
	extensionMap map[string]MIMEType
}

// NewMIMETypeDetector creates a new MIME type detector
func NewMIMETypeDetector() *MIMETypeDetector {
	return &MIMETypeDetector{
		extensionMap: map[string]MIMEType{
			".jpg":  MIMETypeJPEG,
			".jpeg": MIMETypeJPEG,
			".png":  MIMETypePNG,
			".gif":  MIMETypeGIF,
			".pdf":  MIMETypePDF,
			".mp4":  MIMETypeMP4,
			".mov":  MIMETypeMOV,
			".docx": MIMETypeDOCX,
			".xlsx": MIMETypeXLSX,
		},
	}
}

// DetectFromContent detects MIME type by reading file content (magic bytes)
func (d *MIMETypeDetector) DetectFromContent(content io.Reader) (MIMEType, error) {
	// Read first 262 bytes for magic number detection
	head := make([]byte, 262)
	n, err := content.Read(head)
	if err != nil && err != io.EOF {
		return "", err
	}
	head = head[:n]

	// Use filetype library for detection
	kind, err := filetype.Match(head)
	if err != nil {
		return "", err
	}

	if kind == filetype.Unknown {
		// Try to detect Office documents (DOCX, XLSX) which are ZIP-based
		if isZipBased(head) {
			// For ZIP-based files, we need additional detection
			// This is a simplified check - in production, you'd inspect the ZIP contents
			return "", nil // Unknown ZIP-based format
		}
		return "", nil // Unknown type
	}

	return MIMEType(kind.MIME.Value), nil
}

// DetectFromBytes detects MIME type from byte slice
func (d *MIMETypeDetector) DetectFromBytes(data []byte) (MIMEType, error) {
	kind, err := filetype.Match(data)
	if err != nil {
		return "", err
	}

	if kind == filetype.Unknown {
		return "", nil
	}

	return MIMEType(kind.MIME.Value), nil
}

// GetExpectedMIMEType returns the expected MIME type for a file extension
func (d *MIMETypeDetector) GetExpectedMIMEType(filename string) MIMEType {
	ext := strings.ToLower(filepath.Ext(filename))
	return d.extensionMap[ext]
}

// ExtensionMatchesMIME checks if the file extension matches the detected MIME type
func (d *MIMETypeDetector) ExtensionMatchesMIME(filename string, mimeType MIMEType) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	expectedMIME, exists := d.extensionMap[ext]
	if !exists {
		return false
	}
	return expectedMIME == mimeType
}

// GetExtensionsForMIME returns valid extensions for a MIME type
func (d *MIMETypeDetector) GetExtensionsForMIME(mimeType MIMEType) []string {
	var extensions []string
	for ext, mime := range d.extensionMap {
		if mime == mimeType {
			extensions = append(extensions, ext)
		}
	}
	return extensions
}

// isZipBased checks if the file starts with ZIP magic bytes
func isZipBased(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	// ZIP magic number: PK\x03\x04
	return data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04
}

// MagicBytes contains magic byte signatures for common file types
var MagicBytes = map[MIMEType][]byte{
	MIMETypeJPEG: {0xFF, 0xD8, 0xFF},
	MIMETypePNG:  {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
	MIMETypeGIF:  {0x47, 0x49, 0x46, 0x38}, // GIF8
	MIMETypePDF:  {0x25, 0x50, 0x44, 0x46}, // %PDF
}

// HasValidMagicBytes checks if data starts with valid magic bytes for the MIME type
func HasValidMagicBytes(data []byte, mimeType MIMEType) bool {
	magic, exists := MagicBytes[mimeType]
	if !exists {
		return true // No magic bytes defined, assume valid
	}

	if len(data) < len(magic) {
		return false
	}

	for i, b := range magic {
		if data[i] != b {
			return false
		}
	}
	return true
}
