// Package middleware provides unit tests for HTTP middleware.
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/auth-platform/sdk-go/src/middleware"
	"github.com/auth-platform/sdk-go/src/token"
	"github.com/auth-platform/sdk-go/src/types"
)

// mockValidator implements TokenValidator for testing.
type mockValidator struct {
	claims *types.Claims
	err    error
}

func (m *mockValidator) ValidateToken(tokenStr string, audience string) (*token.ValidationResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return token.NewValidationResult(m.claims, tokenStr, token.SchemeBearer), nil
}

func TestConfig_ShouldSkip(t *testing.T) {
	config := middleware.NewConfig(
		middleware.WithSkipPatterns("^/health$", "^/metrics"),
	)

	tests := []struct {
		path string
		skip bool
	}{
		{"/health", true},
		{"/metrics", true},
		{"/metrics/cpu", true},
		{"/api/users", false},
		{"/healthcheck", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		if got := config.ShouldSkip(req); got != tt.skip {
			t.Errorf("ShouldSkip(%s) = %v, want %v", tt.path, got, tt.skip)
		}
	}
}

func TestHTTPMiddleware_SkipPattern(t *testing.T) {
	validator := &mockValidator{claims: &types.Claims{Subject: "user1"}}
	mw := middleware.NewHTTPMiddleware(validator,
		middleware.WithSkipPatterns("^/health$"),
	)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request to skipped path should succeed without token
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHTTPMiddleware_MissingToken(t *testing.T) {
	validator := &mockValidator{claims: &types.Claims{Subject: "user1"}}
	mw := middleware.NewHTTPMiddleware(validator)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/users", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestContextWithClaims(t *testing.T) {
	claims := &types.Claims{Subject: "user123", Scope: "read write"}
	ctx := middleware.ContextWithClaims(context.Background(), claims)

	retrieved, ok := middleware.GetClaimsFromContext(ctx)
	if !ok {
		t.Fatal("claims not found in context")
	}
	if retrieved.Subject != "user123" {
		t.Errorf("Subject = %s, want user123", retrieved.Subject)
	}
}

func TestGetClaimsFromContext_NotPresent(t *testing.T) {
	ctx := context.Background()
	_, ok := middleware.GetClaimsFromContext(ctx)
	if ok {
		t.Error("expected claims not found")
	}
}

func TestContextWithToken(t *testing.T) {
	ctx := middleware.ContextWithToken(context.Background(), "mytoken")

	retrieved, ok := middleware.GetTokenFromContext(ctx)
	if !ok {
		t.Fatal("token not found in context")
	}
	if retrieved != "mytoken" {
		t.Errorf("token = %s, want mytoken", retrieved)
	}
}

func TestGetSubject(t *testing.T) {
	claims := &types.Claims{Subject: "user123"}
	ctx := middleware.ContextWithClaims(context.Background(), claims)

	subject, ok := middleware.GetSubject(ctx)
	if !ok {
		t.Fatal("subject not found")
	}
	if subject != "user123" {
		t.Errorf("subject = %s, want user123", subject)
	}
}

func TestHasScope(t *testing.T) {
	claims := &types.Claims{Scope: "read write admin"}
	ctx := middleware.ContextWithClaims(context.Background(), claims)

	tests := []struct {
		scope string
		has   bool
	}{
		{"read", true},
		{"write", true},
		{"admin", true},
		{"delete", false},
		{"rea", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := middleware.HasScope(ctx, tt.scope); got != tt.has {
			t.Errorf("HasScope(%s) = %v, want %v", tt.scope, got, tt.has)
		}
	}
}

func TestRequireScope(t *testing.T) {
	claims := &types.Claims{Scope: "read write"}
	ctx := middleware.ContextWithClaims(context.Background(), claims)

	handler := middleware.RequireScope("read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireScope_Forbidden(t *testing.T) {
	claims := &types.Claims{Scope: "read"}
	ctx := middleware.ContextWithClaims(context.Background(), claims)

	handler := middleware.RequireScope("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCustomErrorHandler(t *testing.T) {
	validator := &mockValidator{claims: &types.Claims{}}
	customCalled := false

	mw := middleware.NewHTTPMiddleware(validator,
		middleware.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
			customCalled = true
			w.WriteHeader(http.StatusTeapot)
		}),
	)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api", nil) // No token
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !customCalled {
		t.Error("custom error handler not called")
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
}
