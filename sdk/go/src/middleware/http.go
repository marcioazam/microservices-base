package middleware

import (
	"net/http"

	"github.com/auth-platform/sdk-go/src/token"
)

// TokenValidator validates tokens and returns claims.
type TokenValidator interface {
	ValidateToken(tokenStr string, audience string) (*token.ValidationResult, error)
}

// HTTPMiddleware creates HTTP middleware for authentication.
type HTTPMiddleware struct {
	config    *Config
	validator TokenValidator
}

// NewHTTPMiddleware creates a new HTTP middleware.
func NewHTTPMiddleware(validator TokenValidator, opts ...Option) *HTTPMiddleware {
	config := NewConfig(opts...)
	return &HTTPMiddleware{
		config:    config,
		validator: validator,
	}
}

// Handler returns the middleware handler function.
func (m *HTTPMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check skip patterns
		if m.config.ShouldSkip(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token
		extractor := m.config.TokenExtractor
		if extractor == nil {
			extractor = token.NewHTTPExtractor(r)
		}

		tokenStr, _, err := extractor.Extract(r.Context())
		if err != nil {
			m.handleError(w, r, err)
			return
		}

		// Validate token
		result, err := m.validator.ValidateToken(tokenStr, m.config.Audience)
		if err != nil {
			m.handleError(w, r, err)
			return
		}

		// Add claims to context
		ctx := ContextWithClaims(r.Context(), result.Claims)
		ctx = ContextWithToken(ctx, tokenStr)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *HTTPMiddleware) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if m.config.ErrorHandler != nil {
		m.config.ErrorHandler(w, r, err)
	} else {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

// Middleware is a convenience function that returns the handler.
func (m *HTTPMiddleware) Middleware() func(http.Handler) http.Handler {
	return m.Handler
}

// RequireScope creates middleware that requires a specific scope.
func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !HasScope(r.Context(), scope) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyScope creates middleware that requires any of the specified scopes.
func RequireAnyScope(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, scope := range scopes {
				if HasScope(r.Context(), scope) {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
		})
	}
}

// RequireAllScopes creates middleware that requires all specified scopes.
func RequireAllScopes(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, scope := range scopes {
				if !HasScope(r.Context(), scope) {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
