// Package security provides security utilities.
package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"html"
	"regexp"
	"strings"
)

// ConstantTimeCompare compares two strings in constant time.
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ConstantTimeCompareBytes compares two byte slices in constant time.
func ConstantTimeCompareBytes(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// GenerateRandomBytes generates cryptographically secure random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

// GenerateRandomHex generates a random hex string.
func GenerateRandomHex(n int) (string, error) {
	b, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateRandomBase64 generates a random base64 string.
func GenerateRandomBase64(n int) (string, error) {
	b, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GenerateToken generates a secure random token.
func GenerateToken(length int) (string, error) {
	return GenerateRandomBase64(length)
}

// SanitizeHTML escapes HTML special characters.
func SanitizeHTML(s string) string {
	return html.EscapeString(s)
}

// sqlInjectionPatterns for SQL injection detection.
var sqlInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(\b(SELECT|INSERT|UPDATE|DELETE|DROP|UNION|ALTER|CREATE|TRUNCATE)\b)`),
	regexp.MustCompile(`(?i)(--|#|/\*|\*/)`),
	regexp.MustCompile(`(?i)(\bOR\b\s+\d+\s*=\s*\d+)`),
	regexp.MustCompile(`(?i)(\bAND\b\s+\d+\s*=\s*\d+)`),
	regexp.MustCompile(`['";]`),
}

// SanitizeSQL removes potentially dangerous SQL characters.
func SanitizeSQL(s string) string {
	result := s
	// Remove SQL comments
	result = regexp.MustCompile(`--.*$`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(result, "")
	// Escape single quotes
	result = strings.ReplaceAll(result, "'", "''")
	return result
}

// DetectSQLInjection checks if string contains SQL injection patterns.
func DetectSQLInjection(s string) bool {
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
}

// shellDangerousChars for shell injection prevention.
var shellDangerousChars = []string{
	";", "&", "|", "`", "$", "(", ")", "{", "}", "[", "]",
	"<", ">", "!", "\\", "\n", "\r", "'", "\"",
}

// SanitizeShell removes shell metacharacters.
func SanitizeShell(s string) string {
	result := s
	for _, char := range shellDangerousChars {
		result = strings.ReplaceAll(result, char, "")
	}
	return result
}

// EscapeShell escapes shell metacharacters.
func EscapeShell(s string) string {
	result := s
	for _, char := range shellDangerousChars {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}

// ValidateInput checks if input is safe (no injection patterns).
func ValidateInput(s string) bool {
	// Check for null bytes
	if strings.Contains(s, "\x00") {
		return false
	}
	// Check for SQL injection
	if DetectSQLInjection(s) {
		return false
	}
	return true
}

// MaskSensitive masks sensitive data for logging.
func MaskSensitive(s string, visibleChars int) string {
	if len(s) <= visibleChars*2 {
		return strings.Repeat("*", len(s))
	}
	return s[:visibleChars] + strings.Repeat("*", len(s)-visibleChars*2) + s[len(s)-visibleChars:]
}

// HashEquals compares two hashes in constant time.
func HashEquals(hash1, hash2 []byte) bool {
	return ConstantTimeCompareBytes(hash1, hash2)
}
