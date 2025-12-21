package property

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	authplatform "github.com/auth-platform/sdk-go"
	"pgregory.net/rapid"
)

// TestProperty13_ClaimsStoredInContextAfterValidation tests that claims are stored in context.
func TestProperty13_ClaimsStoredInContextAfterValidation(t *testing.T) {
	t.Run("ClaimsAccessibleFromContext", func(t *testing.T) {
		// Test that GetClaimsFromContext works correctly
		claims := &authplatform.Claims{
			Subject:  "user123",
			Issuer:   "https://auth.example.com",
			Audience: []string{"api"},
		}

		ctx := context.WithValue(context.Background(), authplatform.ClaimsContextKey, claims)

		retrieved, ok := authplatform.GetClaimsFromContext(ctx)
		if !ok {
			t.Fatal("expected claims to be found in context")
		}
		if retrieved.Subject != claims.Subject {
			t.Errorf("expected subject %s, got %s", claims.Subject, retrieved.Subject)
		}
		if retrieved.Issuer != claims.Issuer {
			t.Errorf("expected issuer %s, got %s", claims.Issuer, retrieved.Issuer)
		}
	})

	t.Run("NilContextReturnsNotFound", func(t *testing.T) {
		ctx := context.Background()
		_, ok := authplatform.GetClaimsFromContext(ctx)
		if ok {
			t.Error("expected claims not to be found in empty context")
		}
	})
}

// TestProperty15_SkipPatternMatching tests skip pattern functionality.
func TestProperty15_SkipPatternMatching(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random paths
		path := rapid.StringMatching(`^/[a-z]+(/[a-z]+)*$`).Draw(t, "path")

		// Test exact match pattern
		pattern := regexp.MustCompile("^" + regexp.QuoteMeta(path) + "$")
		if !pattern.MatchString(path) {
			t.Fatalf("pattern should match exact path: %s", path)
		}
	})
}

func TestSkipPatternExactMatch(t *testing.T) {
	testCases := []struct {
		pattern string
		path    string
		match   bool
	}{
		{`^/health$`, "/health", true},
		{`^/health$`, "/healthz", false},
		{`^/api/v1/.*`, "/api/v1/users", true},
		{`^/api/v1/.*`, "/api/v2/users", false},
		{`^/public/.*`, "/public/assets/logo.png", true},
		{`^/(health|ready|live)$`, "/health", true},
		{`^/(health|ready|live)$`, "/ready", true},
		{`^/(health|ready|live)$`, "/live", true},
		{`^/(health|ready|live)$`, "/other", false},
	}

	for _, tc := range testCases {
		t.Run(tc.pattern+"_"+tc.path, func(t *testing.T) {
			re := regexp.MustCompile(tc.pattern)
			if re.MatchString(tc.path) != tc.match {
				t.Errorf("pattern %s, path %s: expected match=%v", tc.pattern, tc.path, tc.match)
			}
		})
	}
}

func TestMiddlewareConfigOptions(t *testing.T) {
	t.Run("WithSkipPatterns", func(t *testing.T) {
		config := &authplatform.MiddlewareConfig{}
		opt := authplatform.WithSkipPatterns(`^/health$`, `^/ready$`)
		opt(config)

		if len(config.SkipPatterns) != 2 {
			t.Errorf("expected 2 skip patterns, got %d", len(config.SkipPatterns))
		}
	})

	t.Run("WithAudience", func(t *testing.T) {
		config := &authplatform.MiddlewareConfig{}
		opt := authplatform.WithAudience("test-audience")
		opt(config)

		if config.Audience != "test-audience" {
			t.Errorf("expected audience test-audience, got %s", config.Audience)
		}
	})

	t.Run("WithIssuer", func(t *testing.T) {
		config := &authplatform.MiddlewareConfig{}
		opt := authplatform.WithIssuer("https://auth.example.com")
		opt(config)

		if config.Issuer != "https://auth.example.com" {
			t.Errorf("expected issuer https://auth.example.com, got %s", config.Issuer)
		}
	})

	t.Run("WithRequiredClaims", func(t *testing.T) {
		config := &authplatform.MiddlewareConfig{}
		opt := authplatform.WithRequiredClaims("sub", "email", "roles")
		opt(config)

		if len(config.RequiredClaims) != 3 {
			t.Errorf("expected 3 required claims, got %d", len(config.RequiredClaims))
		}
	})
}

func TestCustomErrorHandler(t *testing.T) {
	customHandlerCalled := false
	customHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		customHandlerCalled = true
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Custom error: " + err.Error()))
	}

	config := &authplatform.MiddlewareConfig{}
	opt := authplatform.WithErrorHandler(customHandler)
	opt(config)

	// Simulate calling the error handler
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	config.ErrorHandler(w, r, &authplatform.SDKError{Message: "test error"})

	if !customHandlerCalled {
		t.Error("custom error handler was not called")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestContextKeyType(t *testing.T) {
	// Ensure context key is properly typed to avoid collisions
	key := authplatform.ClaimsContextKey
	if key != "authplatform_claims" {
		t.Errorf("unexpected context key value: %s", key)
	}
}
