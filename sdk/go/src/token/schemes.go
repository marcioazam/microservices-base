// Package token provides token extraction and validation utilities.
package token

import "strings"

// TokenScheme represents the authentication scheme.
type TokenScheme string

const (
	// SchemeBearer represents Bearer token authentication.
	SchemeBearer TokenScheme = "Bearer"
	// SchemeDPoP represents DPoP token authentication.
	SchemeDPoP TokenScheme = "DPoP"
	// SchemeUnknown represents an unknown token scheme.
	SchemeUnknown TokenScheme = ""
)

// String returns the string representation of the scheme.
func (s TokenScheme) String() string {
	return string(s)
}

// IsValid returns true if the scheme is a known valid scheme.
func (s TokenScheme) IsValid() bool {
	return s == SchemeBearer || s == SchemeDPoP
}

// AllSchemes returns all valid token schemes.
func AllSchemes() []TokenScheme {
	return []TokenScheme{SchemeBearer, SchemeDPoP}
}

// ParseScheme parses a string into a TokenScheme (case-insensitive).
func ParseScheme(s string) TokenScheme {
	lower := strings.ToLower(s)
	switch lower {
	case "bearer":
		return SchemeBearer
	case "dpop":
		return SchemeDPoP
	default:
		return SchemeUnknown
	}
}
