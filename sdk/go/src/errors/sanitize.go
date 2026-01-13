package errors

import (
	"errors"
	"regexp"
	"strings"
)

// sensitivePatterns contains patterns that should not appear in error messages.
var sensitivePatterns = []string{
	"Bearer ",
	"DPoP ",
	"secret",
	"password",
	"credential",
	"eyJ", // JWT header prefix (base64)
}

// sensitiveRegexps contains compiled regex patterns for sensitive data.
var sensitiveRegexps = []*regexp.Regexp{
	regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9._-]+`),
	regexp.MustCompile(`(?i)dpop\s+[a-zA-Z0-9._-]+`),
	regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`), // JWT pattern
}

// ContainsSensitiveData checks if a string contains sensitive data patterns.
func ContainsSensitiveData(s string) bool {
	lower := strings.ToLower(s)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}
	for _, re := range sensitiveRegexps {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// SanitizeString removes sensitive data from a string.
func SanitizeString(s string) string {
	result := s
	for _, re := range sensitiveRegexps {
		result = re.ReplaceAllString(result, "[REDACTED]")
	}
	for _, pattern := range sensitivePatterns {
		result = strings.ReplaceAll(result, pattern, "[REDACTED]")
		result = strings.ReplaceAll(result, strings.ToLower(pattern), "[REDACTED]")
		result = strings.ReplaceAll(result, strings.ToUpper(pattern), "[REDACTED]")
	}
	return result
}

// SanitizeError removes sensitive data from error messages.
func SanitizeError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if ContainsSensitiveData(msg) {
		return errors.New("authplatform: error occurred (details redacted)")
	}
	return err
}

// SanitizeErrorPreserveCode sanitizes error but preserves SDKError code.
func SanitizeErrorPreserveCode(err error) error {
	if err == nil {
		return nil
	}
	var sdkErr *SDKError
	if errors.As(err, &sdkErr) {
		if ContainsSensitiveData(sdkErr.Message) {
			return NewError(sdkErr.Code, "details redacted for security")
		}
		return err
	}
	return SanitizeError(err)
}
