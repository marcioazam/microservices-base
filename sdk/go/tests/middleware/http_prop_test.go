// Package middleware provides property-based tests for HTTP middleware.
package middleware

import (
	"context"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/auth-platform/sdk-go/src/middleware"
	"github.com/auth-platform/sdk-go/src/types"
	"pgregory.net/rapid"
)

// Property 21: Middleware Skip Patterns
func TestProperty_MiddlewareSkipPatterns(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		pattern := rapid.SampledFrom([]string{"/health", "/metrics", "/ready"}).Draw(t, "pattern")
		suffix := rapid.StringMatching(`[a-z]{0,10}`).Draw(t, "suffix")
		path := pattern + suffix

		config := middleware.NewConfig(
			middleware.WithSkipPatterns("^" + regexp.QuoteMeta(pattern)),
		)

		req := httptest.NewRequest("GET", path, nil)
		skipped := config.ShouldSkip(req)

		if !skipped {
			t.Fatalf("path %s should be skipped with pattern %s", path, pattern)
		}
	})
}

// Property 22: Middleware Claims Context
func TestProperty_MiddlewareClaimsContext(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		subject := rapid.StringMatching(`[a-z0-9]{5,20}`).Draw(t, "subject")
		scope := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "scope")

		claims := &types.Claims{
			Subject: subject,
			Scope:   scope,
		}

		ctx := middleware.ContextWithClaims(context.Background(), claims)
		retrieved, ok := middleware.GetClaimsFromContext(ctx)

		if !ok {
			t.Fatal("claims should be retrievable from context")
		}
		if retrieved.Subject != subject {
			t.Fatalf("subject = %s, want %s", retrieved.Subject, subject)
		}
		if retrieved.Scope != scope {
			t.Fatalf("scope = %s, want %s", retrieved.Scope, scope)
		}
	})
}

// Property 24: Middleware Audience/Issuer Validation
func TestProperty_MiddlewareAudienceIssuerConfig(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		audience := rapid.StringMatching(`[a-z0-9-]{5,20}`).Draw(t, "audience")
		issuer := "https://" + rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "issuer") + ".com"

		config := middleware.NewConfig(
			middleware.WithAudience(audience),
			middleware.WithIssuer(issuer),
		)

		if config.Audience != audience {
			t.Fatalf("audience = %s, want %s", config.Audience, audience)
		}
		if config.Issuer != issuer {
			t.Fatalf("issuer = %s, want %s", config.Issuer, issuer)
		}
	})
}

// Property: Token is preserved in context
func TestProperty_TokenPreservedInContext(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		token := rapid.StringMatching(`[A-Za-z0-9._-]{20,100}`).Draw(t, "token")

		ctx := middleware.ContextWithToken(context.Background(), token)
		retrieved, ok := middleware.GetTokenFromContext(ctx)

		if !ok {
			t.Fatal("token should be retrievable from context")
		}
		if retrieved != token {
			t.Fatalf("token = %s, want %s", retrieved, token)
		}
	})
}

// Property: HasScope correctly identifies scopes
func TestProperty_HasScopeCorrectlyIdentifies(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numScopes := rapid.IntRange(1, 5).Draw(t, "numScopes")
		scopes := make([]string, numScopes)
		for i := 0; i < numScopes; i++ {
			scopes[i] = rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "scope")
		}

		// Build space-separated scope string
		scopeStr := ""
		for i, s := range scopes {
			if i > 0 {
				scopeStr += " "
			}
			scopeStr += s
		}

		claims := &types.Claims{Scope: scopeStr}
		ctx := middleware.ContextWithClaims(context.Background(), claims)

		// Each scope should be found
		for _, s := range scopes {
			if !middleware.HasScope(ctx, s) {
				t.Fatalf("scope %s should be found in %s", s, scopeStr)
			}
		}
	})
}

// Property: Non-existent scope returns false
func TestProperty_NonExistentScopeReturnsFalse(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		existingScope := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "existing")
		nonExistentScope := rapid.StringMatching(`[A-Z]{3,10}`).Draw(t, "nonExistent")

		claims := &types.Claims{Scope: existingScope}
		ctx := middleware.ContextWithClaims(context.Background(), claims)

		if middleware.HasScope(ctx, nonExistentScope) {
			t.Fatalf("scope %s should not be found", nonExistentScope)
		}
	})
}

// Property: Skip patterns are compiled correctly
func TestProperty_SkipPatternsCompiled(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Use simple patterns that are valid regex
		patterns := []string{
			"^/health$",
			"^/metrics",
			"^/api/v[0-9]+",
		}
		pattern := rapid.SampledFrom(patterns).Draw(t, "pattern")

		config := middleware.NewConfig(
			middleware.WithSkipPatterns(pattern),
		)

		if len(config.SkipPatterns) != 1 {
			t.Fatalf("expected 1 skip pattern, got %d", len(config.SkipPatterns))
		}
	})
}

// Property: Required claims are stored correctly
func TestProperty_RequiredClaimsStored(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numClaims := rapid.IntRange(1, 5).Draw(t, "numClaims")
		claims := make([]string, numClaims)
		for i := 0; i < numClaims; i++ {
			claims[i] = rapid.StringMatching(`[a-z_]{3,10}`).Draw(t, "claim")
		}

		config := middleware.NewConfig(
			middleware.WithRequiredClaims(claims...),
		)

		if len(config.RequiredClaims) != numClaims {
			t.Fatalf("expected %d required claims, got %d", numClaims, len(config.RequiredClaims))
		}
	})
}

// Property: GetSubject returns empty for missing claims
func TestProperty_GetSubjectEmptyForMissing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		subject, ok := middleware.GetSubject(ctx)

		if ok {
			t.Fatal("GetSubject should return false for missing claims")
		}
		if subject != "" {
			t.Fatal("GetSubject should return empty string for missing claims")
		}
	})
}

// Property: GetClientID returns correct value
func TestProperty_GetClientIDReturnsCorrect(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		clientID := rapid.StringMatching(`[a-z0-9-]{10,30}`).Draw(t, "clientID")

		claims := &types.Claims{ClientID: clientID}
		ctx := middleware.ContextWithClaims(context.Background(), claims)

		retrieved, ok := middleware.GetClientID(ctx)
		if !ok {
			t.Fatal("GetClientID should return true")
		}
		if retrieved != clientID {
			t.Fatalf("clientID = %s, want %s", retrieved, clientID)
		}
	})
}
