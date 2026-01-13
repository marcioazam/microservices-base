// Package token provides unit tests for JWKS cache.
package token

import (
	"testing"
	"time"

	"github.com/auth-platform/sdk-go/src/token"
	"github.com/auth-platform/sdk-go/src/types"
)

func TestJWKSMetrics_Initial(t *testing.T) {
	metrics := &token.JWKSMetrics{}

	if metrics.Hits != 0 {
		t.Errorf("initial Hits = %d, want 0", metrics.Hits)
	}
	if metrics.Misses != 0 {
		t.Errorf("initial Misses = %d, want 0", metrics.Misses)
	}
	if metrics.Refreshes != 0 {
		t.Errorf("initial Refreshes = %d, want 0", metrics.Refreshes)
	}
	if metrics.Errors != 0 {
		t.Errorf("initial Errors = %d, want 0", metrics.Errors)
	}
}

func TestDefaultJWKSCacheConfig(t *testing.T) {
	uri := "https://example.com/.well-known/jwks.json"
	config := token.DefaultJWKSCacheConfig(uri)

	if config.URI != uri {
		t.Errorf("URI = %s, want %s", config.URI, uri)
	}
	if config.TTL != time.Hour {
		t.Errorf("TTL = %v, want 1h", config.TTL)
	}
}

func TestValidationOptions_Defaults(t *testing.T) {
	opts := types.ValidationOptions{}

	if opts.Audience != "" {
		t.Error("default Audience should be empty")
	}
	if opts.Issuer != "" {
		t.Error("default Issuer should be empty")
	}
	if opts.SkipExpiry {
		t.Error("default SkipExpiry should be false")
	}
	if len(opts.RequiredClaims) != 0 {
		t.Error("default RequiredClaims should be empty")
	}
}

func TestValidationOptions_WithValues(t *testing.T) {
	opts := types.ValidationOptions{
		Audience:       "my-api",
		Issuer:         "https://issuer.example.com",
		RequiredClaims: []string{"sub", "scope"},
		SkipExpiry:     true,
	}

	if opts.Audience != "my-api" {
		t.Errorf("Audience = %s, want my-api", opts.Audience)
	}
	if opts.Issuer != "https://issuer.example.com" {
		t.Errorf("Issuer = %s, want https://issuer.example.com", opts.Issuer)
	}
	if len(opts.RequiredClaims) != 2 {
		t.Errorf("RequiredClaims length = %d, want 2", len(opts.RequiredClaims))
	}
	if !opts.SkipExpiry {
		t.Error("SkipExpiry should be true")
	}
}

// Note: Full integration tests for JWKSCache require a mock JWKS server
// These tests verify the structure and configuration aspects

func TestJWKSCacheConfig_CustomTTL(t *testing.T) {
	uri := "https://example.com/.well-known/jwks.json"
	config := token.JWKSCacheConfig{
		URI: uri,
		TTL: 30 * time.Minute,
	}

	if config.TTL != 30*time.Minute {
		t.Errorf("TTL = %v, want 30m", config.TTL)
	}
}

func TestTokenScheme_String(t *testing.T) {
	tests := []struct {
		scheme token.TokenScheme
		want   string
	}{
		{token.SchemeBearer, "Bearer"},
		{token.SchemeDPoP, "DPoP"},
		{token.SchemeUnknown, ""},
	}

	for _, tt := range tests {
		if got := tt.scheme.String(); got != tt.want {
			t.Errorf("TokenScheme(%v).String() = %s, want %s", tt.scheme, got, tt.want)
		}
	}
}

func TestTokenScheme_IsValid(t *testing.T) {
	tests := []struct {
		scheme token.TokenScheme
		valid  bool
	}{
		{token.SchemeBearer, true},
		{token.SchemeDPoP, true},
		{token.SchemeUnknown, false},
	}

	for _, tt := range tests {
		if got := tt.scheme.IsValid(); got != tt.valid {
			t.Errorf("TokenScheme(%v).IsValid() = %v, want %v", tt.scheme, got, tt.valid)
		}
	}
}

func TestAllSchemes(t *testing.T) {
	schemes := token.AllSchemes()

	if len(schemes) != 2 {
		t.Errorf("AllSchemes() length = %d, want 2", len(schemes))
	}

	hasBearer := false
	hasDPoP := false
	for _, s := range schemes {
		if s == token.SchemeBearer {
			hasBearer = true
		}
		if s == token.SchemeDPoP {
			hasDPoP = true
		}
	}

	if !hasBearer {
		t.Error("AllSchemes should include Bearer")
	}
	if !hasDPoP {
		t.Error("AllSchemes should include DPoP")
	}
}

func TestParseScheme(t *testing.T) {
	tests := []struct {
		input string
		want  token.TokenScheme
	}{
		{"Bearer", token.SchemeBearer},
		{"bearer", token.SchemeBearer},
		{"DPoP", token.SchemeDPoP},
		{"dpop", token.SchemeDPoP},
		{"unknown", token.SchemeUnknown},
		{"", token.SchemeUnknown},
		{"Basic", token.SchemeUnknown},
	}

	for _, tt := range tests {
		if got := token.ParseScheme(tt.input); got != tt.want {
			t.Errorf("ParseScheme(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
