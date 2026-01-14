// Package security provides security utilities for file upload service.
package security

import (
	"crypto/subtle"
	"path/filepath"
	"regexp"
	"strings"
)

// PathTraversalChars contains characters that could be used for path traversal.
var PathTraversalChars = []string{"..", "/", "\\", "%2e", "%2f", "%5c"}

// SensitivePatterns contains patterns for sensitive data redaction.
var SensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*\S+`),
	regexp.MustCompile(`(?i)(token|api[_-]?key|secret)\s*[:=]\s*\S+`),
	regexp.MustCompile(`(?i)(bearer)\s+\S+`),
	regexp.MustCompile(`(?i)(authorization)\s*[:=]\s*\S+`),
}

// SanitizeFilename sanitizes a filename to prevent path traversal attacks.
func SanitizeFilename(filename string) string {
	// Remove path separators
	filename = filepath.Base(filename)

	// Remove path traversal sequences
	for _, char := range PathTraversalChars {
		filename = strings.ReplaceAll(filename, char, "")
	}

	// Remove null bytes
	filename = strings.ReplaceAll(filename, "\x00", "")

	// Remove control characters
	var sanitized strings.Builder
	for _, r := range filename {
		if r >= 32 && r != 127 {
			sanitized.WriteRune(r)
		}
	}

	result := sanitized.String()

	// Ensure filename is not empty
	if result == "" || result == "." {
		return "unnamed"
	}

	return result
}

// ValidateFilename checks if a filename is safe.
func ValidateFilename(filename string) bool {
	if filename == "" {
		return false
	}

	// Check for path traversal
	for _, char := range PathTraversalChars {
		if strings.Contains(filename, char) {
			return false
		}
	}

	// Check for null bytes
	if strings.Contains(filename, "\x00") {
		return false
	}

	// Check for control characters
	for _, r := range filename {
		if r < 32 || r == 127 {
			return false
		}
	}

	return true
}

// ConstantTimeCompare performs constant-time comparison of two strings.
// This prevents timing attacks when comparing secrets.
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// RedactSensitiveData redacts sensitive data from a string.
func RedactSensitiveData(input string) string {
	result := input
	for _, pattern := range SensitivePatterns {
		result = pattern.ReplaceAllString(result, "[REDACTED]")
	}
	return result
}

// RedactMap redacts sensitive values from a map.
func RedactMap(data map[string]string) map[string]string {
	sensitiveKeys := []string{
		"password", "passwd", "pwd", "secret",
		"token", "api_key", "apikey", "api-key",
		"authorization", "auth", "bearer",
		"credential", "credentials",
	}

	result := make(map[string]string, len(data))
	for k, v := range data {
		keyLower := strings.ToLower(k)
		isSensitive := false
		for _, sensitive := range sensitiveKeys {
			if strings.Contains(keyLower, sensitive) {
				isSensitive = true
				break
			}
		}
		if isSensitive {
			result[k] = "[REDACTED]"
		} else {
			result[k] = v
		}
	}
	return result
}

// ValidateTenantAccess validates that a path belongs to the specified tenant.
func ValidateTenantAccess(path, tenantID string) bool {
	if path == "" || tenantID == "" {
		return false
	}

	// Extract tenant from path (first segment)
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 {
		return false
	}

	return parts[0] == tenantID
}

// SanitizePath sanitizes a storage path.
func SanitizePath(path string) string {
	// Remove leading/trailing slashes
	path = strings.Trim(path, "/")

	// Remove double slashes
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	// Remove path traversal
	parts := strings.Split(path, "/")
	var sanitized []string
	for _, part := range parts {
		if part != ".." && part != "." && part != "" {
			sanitized = append(sanitized, part)
		}
	}

	return strings.Join(sanitized, "/")
}
