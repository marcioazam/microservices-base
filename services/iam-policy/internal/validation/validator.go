// Package validation provides input validation for IAM Policy Service.
package validation

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/auth-platform/iam-policy-service/internal/errors"
)

var (
	// Patterns for validation
	uuidPattern     = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
	alphanumPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	actionPattern   = regexp.MustCompile(`^[a-z][a-z0-9_:]*$`)
)

// AuthorizationRequestValidator validates authorization requests.
type AuthorizationRequestValidator struct {
	maxSubjectIDLen   int
	maxResourceIDLen  int
	maxResourceTypeLen int
	maxActionLen      int
}

// NewAuthorizationRequestValidator creates a new validator.
func NewAuthorizationRequestValidator() *AuthorizationRequestValidator {
	return &AuthorizationRequestValidator{
		maxSubjectIDLen:   256,
		maxResourceIDLen:  256,
		maxResourceTypeLen: 64,
		maxActionLen:      64,
	}
}

// ValidateSubjectID validates a subject ID.
func (v *AuthorizationRequestValidator) ValidateSubjectID(id string) error {
	if id == "" {
		return errors.InvalidInput("subject_id is required")
	}
	if len(id) > v.maxSubjectIDLen {
		return errors.InvalidInput("subject_id exceeds maximum length")
	}
	if !isSafeString(id) {
		return errors.InvalidInput("subject_id contains invalid characters")
	}
	return nil
}

// ValidateResourceID validates a resource ID.
func (v *AuthorizationRequestValidator) ValidateResourceID(id string) error {
	if len(id) > v.maxResourceIDLen {
		return errors.InvalidInput("resource_id exceeds maximum length")
	}
	if id != "" && !isSafeString(id) {
		return errors.InvalidInput("resource_id contains invalid characters")
	}
	return nil
}

// ValidateResourceType validates a resource type.
func (v *AuthorizationRequestValidator) ValidateResourceType(resourceType string) error {
	if resourceType == "" {
		return errors.InvalidInput("resource_type is required")
	}
	if len(resourceType) > v.maxResourceTypeLen {
		return errors.InvalidInput("resource_type exceeds maximum length")
	}
	if !alphanumPattern.MatchString(resourceType) {
		return errors.InvalidInput("resource_type contains invalid characters")
	}
	return nil
}

// ValidateAction validates an action.
func (v *AuthorizationRequestValidator) ValidateAction(action string) error {
	if action == "" {
		return errors.InvalidInput("action is required")
	}
	if len(action) > v.maxActionLen {
		return errors.InvalidInput("action exceeds maximum length")
	}
	if !actionPattern.MatchString(action) {
		return errors.InvalidInput("action contains invalid characters")
	}
	return nil
}

// ValidateAuthorizationRequest validates a complete authorization request.
func (v *AuthorizationRequestValidator) ValidateAuthorizationRequest(subjectID, resourceType, resourceID, action string) error {
	if err := v.ValidateSubjectID(subjectID); err != nil {
		return err
	}
	if err := v.ValidateResourceType(resourceType); err != nil {
		return err
	}
	if err := v.ValidateResourceID(resourceID); err != nil {
		return err
	}
	if err := v.ValidateAction(action); err != nil {
		return err
	}
	return nil
}

// isSafeString checks if a string is safe (no control characters or injection attempts).
func isSafeString(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	// Check for common injection patterns
	lower := strings.ToLower(s)
	dangerousPatterns := []string{"<script", "javascript:", "data:", "vbscript:"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			return false
		}
	}
	return true
}

// SanitizeForLog sanitizes a string for safe logging.
func SanitizeForLog(s string) string {
	if len(s) > 256 {
		s = s[:256] + "..."
	}
	var result strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			result.WriteRune('?')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ValidateUUID validates a UUID string.
func ValidateUUID(s string) bool {
	return uuidPattern.MatchString(s)
}

// ValidateAlphanumeric validates an alphanumeric string.
func ValidateAlphanumeric(s string) bool {
	return alphanumPattern.MatchString(s)
}
