package logging

import (
	"regexp"
	"strings"
)

// sensitiveKeyPatterns for field name redaction.
var sensitiveKeyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|passwd|pwd)`),
	regexp.MustCompile(`(?i)(token|api[_-]?key|secret|credential)`),
	regexp.MustCompile(`(?i)(ssn|social[_-]?security)`),
	regexp.MustCompile(`(?i)(credit[_-]?card|card[_-]?number)`),
	regexp.MustCompile(`(?i)(private[_-]?key|secret[_-]?key)`),
}

// piiPatterns for value redaction.
var piiPatterns = []*regexp.Regexp{
	// Email
	regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
	// Phone (various formats)
	regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
	// SSN
	regexp.MustCompile(`\b\d{3}[-]?\d{2}[-]?\d{4}\b`),
	// Credit card (basic)
	regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`),
}

const redactedValue = "[REDACTED]"
const piiValue = "[PII]"

// RedactSensitive redacts sensitive field values.
func RedactSensitive(key string, value any) any {
	lowerKey := strings.ToLower(key)
	for _, pattern := range sensitiveKeyPatterns {
		if pattern.MatchString(lowerKey) {
			return redactedValue
		}
	}
	if str, ok := value.(string); ok {
		return redactPII(str)
	}
	return value
}

// redactPII redacts PII patterns from a string.
func redactPII(s string) string {
	result := s
	for _, pattern := range piiPatterns {
		result = pattern.ReplaceAllString(result, piiValue)
	}
	return result
}

// ContainsPII checks if a string contains PII patterns.
func ContainsPII(s string) bool {
	for _, pattern := range piiPatterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
}

// redactFields applies redaction to all fields.
func redactFields(fields map[string]any) map[string]any {
	if fields == nil {
		return nil
	}
	result := make(map[string]any, len(fields))
	for k, v := range fields {
		result[k] = RedactSensitive(k, v)
	}
	return result
}
