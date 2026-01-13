package token

import "github.com/auth-platform/sdk-go/src/types"

// ValidationResult holds the result of token validation.
type ValidationResult struct {
	Claims *types.Claims
	Token  string
	Scheme TokenScheme
}

// NewValidationResult creates a new validation result.
func NewValidationResult(claims *types.Claims, token string, scheme TokenScheme) *ValidationResult {
	return &ValidationResult{
		Claims: claims,
		Token:  token,
		Scheme: scheme,
	}
}
