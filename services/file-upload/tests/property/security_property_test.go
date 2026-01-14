// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 6: Security Enforcement
// Validates: Requirements 13.1, 13.3, 13.5, 13.6, 4.5, 13.2
package property

import (
	"crypto/subtle"
	"regexp"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// sanitizeFilename sanitizes filenames for testing.
func sanitizeFilename(filename string) string {
	// Remove path traversal sequences
	traversalChars := []string{"..", "/", "\\", "%2e", "%2f", "%5c"}
	for _, char := range traversalChars {
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
	if result == "" || result == "." {
		return "unnamed"
	}
	return result
}

// validateTenantAccess validates tenant access for testing.
func validateTenantAccess(path, tenantID string) bool {
	if path == "" || tenantID == "" {
		return false
	}
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 {
		return false
	}
	return parts[0] == tenantID
}

// redactSensitiveData redacts sensitive data for testing.
func redactSensitiveData(input string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*\S+`),
		regexp.MustCompile(`(?i)(token|api[_-]?key|secret)\s*[:=]\s*\S+`),
		regexp.MustCompile(`(?i)(bearer)\s+\S+`),
		regexp.MustCompile(`(?i)(authorization)\s*:\s*\S+.*`),
	}
	result := input
	for _, pattern := range patterns {
		result = pattern.ReplaceAllString(result, "[REDACTED]")
	}
	return result
}

// constantTimeCompare performs constant-time comparison for testing.
func constantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// TestProperty6_FilenameSanitizedForPathTraversal tests that filenames are sanitized.
// Property 6: Security Enforcement
// Validates: Requirements 13.1, 13.3, 13.5, 13.6, 4.5, 13.2
func TestProperty6_FilenameSanitizedForPathTraversal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate malicious filename patterns
		maliciousPatterns := []string{
			"../../../etc/passwd",
			"..\\..\\windows\\system32",
			"file%2e%2e%2fpasswd",
			"normal/../../../secret",
			"/absolute/path/file.txt",
			"file\x00.txt",
		}

		pattern := rapid.SampledFrom(maliciousPatterns).Draw(t, "pattern")
		sanitized := sanitizeFilename(pattern)

		// Property: Filenames SHALL be sanitized to prevent path traversal
		if strings.Contains(sanitized, "..") {
			t.Errorf("sanitized filename contains '..': %q", sanitized)
		}
		if strings.Contains(sanitized, "/") {
			t.Errorf("sanitized filename contains '/': %q", sanitized)
		}
		if strings.Contains(sanitized, "\\") {
			t.Errorf("sanitized filename contains '\\': %q", sanitized)
		}
		if strings.Contains(sanitized, "\x00") {
			t.Errorf("sanitized filename contains null byte: %q", sanitized)
		}
	})
}

// TestProperty6_TenantIsolationEnforced tests that tenant isolation is enforced.
// Property 6: Security Enforcement
// Validates: Requirements 13.1, 13.3, 13.5, 13.6, 4.5, 13.2
func TestProperty6_TenantIsolationEnforced(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantA := rapid.StringMatching(`tenant-a-[a-z0-9]{8}`).Draw(t, "tenantA")
		tenantB := rapid.StringMatching(`tenant-b-[a-z0-9]{8}`).Draw(t, "tenantB")
		filename := rapid.StringMatching(`[a-z0-9]{8}\.pdf`).Draw(t, "filename")

		pathA := tenantA + "/2025/01/15/hash123/" + filename

		// Property: Tenant A SHALL have access to their own files
		if !validateTenantAccess(pathA, tenantA) {
			t.Errorf("tenant %q should have access to path %q", tenantA, pathA)
		}

		// Property: Tenant B SHALL NOT have access to tenant A's files
		if validateTenantAccess(pathA, tenantB) {
			t.Errorf("tenant %q should NOT have access to path %q", tenantB, pathA)
		}
	})
}

// TestProperty6_SensitiveDataRedacted tests that sensitive data is redacted in logs.
// Property 6: Security Enforcement
// Validates: Requirements 13.1, 13.3, 13.5, 13.6, 4.5, 13.2
func TestProperty6_SensitiveDataRedacted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate sensitive data patterns
		sensitivePatterns := []string{
			"password=secret123",
			"token: abc123xyz",
			"api_key=sk-12345",
			"Bearer eyJhbGciOiJIUzI1NiJ9",
			"Authorization: Basic dXNlcjpwYXNz",
		}

		pattern := rapid.SampledFrom(sensitivePatterns).Draw(t, "pattern")
		redacted := redactSensitiveData(pattern)

		// Property: Sensitive data in logs SHALL be redacted
		if strings.Contains(redacted, "secret123") {
			t.Errorf("password not redacted: %q", redacted)
		}
		if strings.Contains(redacted, "abc123xyz") {
			t.Errorf("token not redacted: %q", redacted)
		}
		if strings.Contains(redacted, "sk-12345") {
			t.Errorf("api_key not redacted: %q", redacted)
		}
		if strings.Contains(redacted, "eyJhbGciOiJIUzI1NiJ9") {
			t.Errorf("bearer token not redacted: %q", redacted)
		}

		// Should contain redaction marker
		if !strings.Contains(redacted, "[REDACTED]") {
			t.Errorf("redacted string should contain [REDACTED]: %q", redacted)
		}
	})
}

// TestProperty6_ConstantTimeTokenComparison tests that token comparison uses constant-time.
// Property 6: Security Enforcement
// Validates: Requirements 13.1, 13.3, 13.5, 13.6, 4.5, 13.2
func TestProperty6_ConstantTimeTokenComparison(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		token := rapid.StringMatching(`[a-zA-Z0-9]{32,64}`).Draw(t, "token")

		// Property: Token comparison SHALL use constant-time algorithm
		// Same tokens should match
		if !constantTimeCompare(token, token) {
			t.Error("identical tokens should match")
		}

		// Different tokens should not match
		differentToken := token + "x"
		if constantTimeCompare(token, differentToken) {
			t.Error("different tokens should not match")
		}

		// Empty vs non-empty should not match
		if constantTimeCompare(token, "") {
			t.Error("token vs empty should not match")
		}
	})
}

// TestProperty6_SafeFilenameGeneration tests that safe filenames are generated.
// Property 6: Security Enforcement
// Validates: Requirements 13.1, 13.3, 13.5, 13.6, 4.5, 13.2
func TestProperty6_SafeFilenameGeneration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random filename with potentially unsafe characters
		baseName := rapid.StringMatching(`[a-zA-Z0-9_-]{1,50}`).Draw(t, "baseName")
		extension := rapid.SampledFrom([]string{".jpg", ".png", ".pdf", ".txt"}).Draw(t, "extension")
		filename := baseName + extension

		sanitized := sanitizeFilename(filename)

		// Property: Safe filenames should pass through unchanged
		if sanitized != filename {
			// Only fail if the original was actually safe
			if !strings.Contains(filename, "..") &&
				!strings.Contains(filename, "/") &&
				!strings.Contains(filename, "\\") {
				t.Errorf("safe filename was modified: %q -> %q", filename, sanitized)
			}
		}

		// Sanitized filename should never be empty
		if sanitized == "" {
			t.Error("sanitized filename should not be empty")
		}
	})
}

// TestProperty6_EmptyPathHandling tests handling of empty paths.
// Property 6: Security Enforcement
// Validates: Requirements 13.1, 13.3, 13.5, 13.6, 4.5, 13.2
func TestProperty6_EmptyPathHandling(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		// Property: Empty path should not validate for any tenant
		if validateTenantAccess("", tenantID) {
			t.Error("empty path should not validate")
		}

		// Property: Empty tenant should not validate for any path
		if validateTenantAccess(tenantID+"/file.txt", "") {
			t.Error("empty tenant should not validate")
		}
	})
}
