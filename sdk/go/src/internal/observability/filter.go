// Package observability provides tracing, logging, and sensitive data filtering.
package observability

import (
	"regexp"
	"strings"
)

// sensitivePatterns contains patterns that indicate sensitive data.
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._-]+`),
	regexp.MustCompile(`(?i)dpop\s+[A-Za-z0-9._-]+`),
	regexp.MustCompile(`(?i)authorization:\s*[^\s]+`),
	regexp.MustCompile(`eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*`), // JWT
	regexp.MustCompile(`(?i)password[=:]\s*[^\s&]+`),
	regexp.MustCompile(`(?i)secret[=:]\s*[^\s&]+`),
	regexp.MustCompile(`(?i)api[_-]?key[=:]\s*[^\s&]+`),
	regexp.MustCompile(`(?i)token[=:]\s*[^\s&]+`),
}

// sensitiveHeaders contains header names that should be redacted.
var sensitiveHeaders = map[string]bool{
	"authorization": true,
	"x-api-key":     true,
	"x-auth-token":  true,
	"cookie":        true,
	"set-cookie":    true,
	"dpop":          true,
}

// FilterSensitiveData redacts sensitive information from a string.
func FilterSensitiveData(input string) string {
	result := input
	for _, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllString(result, "[REDACTED]")
	}
	return result
}

// FilterSensitiveHeaders redacts sensitive headers from a map.
func FilterSensitiveHeaders(headers map[string][]string) map[string][]string {
	filtered := make(map[string][]string)
	for key, values := range headers {
		if sensitiveHeaders[strings.ToLower(key)] {
			filtered[key] = []string{"[REDACTED]"}
		} else {
			filtered[key] = values
		}
	}
	return filtered
}

// IsSensitiveHeader checks if a header name is sensitive.
func IsSensitiveHeader(name string) bool {
	return sensitiveHeaders[strings.ToLower(name)]
}

// ContainsSensitiveData checks if a string contains sensitive patterns.
func ContainsSensitiveData(input string) bool {
	for _, pattern := range sensitivePatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// RedactValue returns a redacted version of a value.
func RedactValue(value string) string {
	if len(value) <= 4 {
		return "[REDACTED]"
	}
	// Show first 2 and last 2 characters
	return value[:2] + "..." + value[len(value)-2:]
}

// SafeMap creates a map with sensitive values redacted.
func SafeMap(m map[string]string, sensitiveKeys ...string) map[string]string {
	sensitive := make(map[string]bool)
	for _, k := range sensitiveKeys {
		sensitive[strings.ToLower(k)] = true
	}

	result := make(map[string]string)
	for k, v := range m {
		if sensitive[strings.ToLower(k)] {
			result[k] = "[REDACTED]"
		} else {
			result[k] = v
		}
	}
	return result
}
