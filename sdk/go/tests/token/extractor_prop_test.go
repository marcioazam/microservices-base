// Package token provides property-based tests for token extraction.
package token

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/auth-platform/sdk-go/src/token"
	"pgregory.net/rapid"
)

// Property 7: Token Extraction Scheme Correctness
func TestProperty_TokenExtractionScheme(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		scheme := rapid.SampledFrom([]string{"Bearer", "DPoP"}).Draw(t, "scheme")
		// Generate a valid token-like string
		tokenVal := rapid.StringMatching(`[A-Za-z0-9._-]{10,50}`).Draw(t, "token")
		header := scheme + " " + tokenVal

		extracted, extractedScheme, err := token.ParseAuthorizationHeader(header)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if extracted != tokenVal {
			t.Fatalf("extracted token = %v, want %v", extracted, tokenVal)
		}
		if extractedScheme != token.ParseScheme(scheme) {
			t.Fatalf("extracted scheme = %v, want %v", extractedScheme, scheme)
		}
	})
}

// Property 8: Chained Extractor Fallback
func TestProperty_ChainedExtractorFallback(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tokenVal := rapid.StringMatching(`[A-Za-z0-9._-]{10,50}`).Draw(t, "token")
		cookieName := rapid.StringMatching(`[a-z_]{3,10}`).Draw(t, "cookieName")

		// Create request with only cookie (no auth header)
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: cookieName, Value: tokenVal})

		chain := token.NewChainedExtractor(
			token.NewHTTPExtractor(req), // Will fail
			token.NewCookieExtractor(req, cookieName, token.SchemeBearer),
		)

		extracted, scheme, err := chain.Extract(context.Background())
		if err != nil {
			t.Fatalf("chained extractor should fallback, got error: %v", err)
		}
		if extracted != tokenVal {
			t.Fatalf("extracted token = %v, want %v", extracted, tokenVal)
		}
		if scheme != token.SchemeBearer {
			t.Fatalf("scheme = %v, want Bearer", scheme)
		}
	})
}

// Property 9: Token Format Validation
func TestProperty_TokenFormatValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate various malformed headers
		malformedType := rapid.IntRange(0, 3).Draw(t, "malformedType")

		var header string
		switch malformedType {
		case 0:
			// No space separator
			header = "Bearer" + rapid.StringMatching(`[A-Za-z0-9]{5,10}`).Draw(t, "token")
		case 1:
			// Empty token
			header = "Bearer "
		case 2:
			// Unknown scheme
			header = "Unknown " + rapid.StringMatching(`[A-Za-z0-9]{5,10}`).Draw(t, "token")
		case 3:
			// Empty header
			header = ""
		}

		_, _, err := token.ParseAuthorizationHeader(header)
		if err == nil {
			t.Fatalf("expected error for malformed header: %q", header)
		}
	})
}

// Property: HTTP Extractor preserves token exactly
func TestProperty_HTTPExtractorPreservesToken(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		scheme := rapid.SampledFrom([]string{"Bearer", "DPoP"}).Draw(t, "scheme")
		tokenVal := rapid.StringMatching(`[A-Za-z0-9._-]{10,100}`).Draw(t, "token")

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", scheme+" "+tokenVal)

		extractor := token.NewHTTPExtractor(req)
		extracted, _, err := extractor.Extract(context.Background())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if extracted != tokenVal {
			t.Fatalf("token not preserved: got %q, want %q", extracted, tokenVal)
		}
	})
}

// Property: Cookie Extractor preserves token exactly
func TestProperty_CookieExtractorPreservesToken(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tokenVal := rapid.StringMatching(`[A-Za-z0-9._-]{10,100}`).Draw(t, "token")
		cookieName := rapid.StringMatching(`[a-z_]{3,15}`).Draw(t, "cookieName")

		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: cookieName, Value: tokenVal})

		extractor := token.NewCookieExtractor(req, cookieName, token.SchemeBearer)
		extracted, _, err := extractor.Extract(context.Background())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if extracted != tokenVal {
			t.Fatalf("token not preserved: got %q, want %q", extracted, tokenVal)
		}
	})
}

// Property: FormatAuthorizationHeader is inverse of ParseAuthorizationHeader
func TestProperty_FormatParseRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		scheme := rapid.SampledFrom([]token.TokenScheme{token.SchemeBearer, token.SchemeDPoP}).Draw(t, "scheme")
		tokenVal := rapid.StringMatching(`[A-Za-z0-9._-]{10,100}`).Draw(t, "token")

		formatted := token.FormatAuthorizationHeader(tokenVal, scheme)
		parsed, parsedScheme, err := token.ParseAuthorizationHeader(formatted)

		if err != nil {
			t.Fatalf("round-trip failed: %v", err)
		}
		if parsed != tokenVal {
			t.Fatalf("token not preserved in round-trip: got %q, want %q", parsed, tokenVal)
		}
		if parsedScheme != scheme {
			t.Fatalf("scheme not preserved in round-trip: got %v, want %v", parsedScheme, scheme)
		}
	})
}

// Property: Scheme parsing is case-insensitive
func TestProperty_SchemeCaseInsensitive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseScheme := rapid.SampledFrom([]string{"bearer", "dpop"}).Draw(t, "baseScheme")
		tokenVal := rapid.StringMatching(`[A-Za-z0-9._-]{10,50}`).Draw(t, "token")

		// Test various case combinations
		variations := []string{
			baseScheme,
			capitalizeFirst(baseScheme),
			allCaps(baseScheme),
		}

		var expectedScheme token.TokenScheme
		if baseScheme == "bearer" {
			expectedScheme = token.SchemeBearer
		} else {
			expectedScheme = token.SchemeDPoP
		}

		for _, variant := range variations {
			header := variant + " " + tokenVal
			_, scheme, err := token.ParseAuthorizationHeader(header)
			if err != nil {
				t.Fatalf("case variant %q should be valid: %v", variant, err)
			}
			if scheme != expectedScheme {
				t.Fatalf("scheme for %q = %v, want %v", variant, scheme, expectedScheme)
			}
		}
	})
}

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func allCaps(s string) string {
	result := make([]byte, len(s))
	for i, c := range []byte(s) {
		if c >= 'a' && c <= 'z' {
			result[i] = c - 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// Property: Valid JWT-like tokens are extracted correctly
func TestProperty_JWTLikeTokenExtraction(t *testing.T) {
	jwtPattern := regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)

	rapid.Check(t, func(t *rapid.T) {
		// Generate JWT-like token (header.payload.signature)
		header := rapid.StringMatching(`[A-Za-z0-9_-]{10,30}`).Draw(t, "header")
		payload := rapid.StringMatching(`[A-Za-z0-9_-]{10,50}`).Draw(t, "payload")
		signature := rapid.StringMatching(`[A-Za-z0-9_-]{20,50}`).Draw(t, "signature")
		jwtToken := header + "." + payload + "." + signature

		if !jwtPattern.MatchString(jwtToken) {
			t.Skip("generated token doesn't match JWT pattern")
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)

		extractor := token.NewHTTPExtractor(req)
		extracted, scheme, err := extractor.Extract(context.Background())

		if err != nil {
			t.Fatalf("JWT-like token extraction failed: %v", err)
		}
		if extracted != jwtToken {
			t.Fatalf("JWT token not preserved: got %q, want %q", extracted, jwtToken)
		}
		if scheme != token.SchemeBearer {
			t.Fatalf("scheme = %v, want Bearer", scheme)
		}
	})
}
