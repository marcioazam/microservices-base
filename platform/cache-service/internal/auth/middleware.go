package auth

import (
	"net/http"
)

// Middleware provides HTTP middleware for JWT authentication.
type Middleware struct {
	validator *JWTValidator
}

// NewMiddleware creates a new authentication middleware.
func NewMiddleware(validator *JWTValidator) *Middleware {
	return &Middleware{validator: validator}
}

// Authenticate is an HTTP middleware that validates JWT tokens.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := m.validator.Validate(authHeader)
		if err != nil {
			switch err {
			case ErrMissingToken:
				http.Error(w, "missing token", http.StatusUnauthorized)
			case ErrExpiredToken:
				http.Error(w, "token expired", http.StatusUnauthorized)
			case ErrInvalidToken, ErrInvalidClaims:
				http.Error(w, "invalid token", http.StatusUnauthorized)
			default:
				http.Error(w, "authentication failed", http.StatusUnauthorized)
			}
			return
		}

		// Add claims to context
		ctx := SetClaimsInContext(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireScope is an HTTP middleware that requires a specific scope.
func (m *Middleware) RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetClaimsFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if !claims.HasScope(scope) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireNamespace is an HTTP middleware that requires a specific namespace.
func (m *Middleware) RequireNamespace(namespace string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetClaimsFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if claims.Namespace != namespace {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Optional is an HTTP middleware that optionally validates JWT tokens.
// If a token is present, it validates it and adds claims to context.
// If no token is present, it continues without authentication.
func (m *Middleware) Optional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		claims, err := m.validator.Validate(authHeader)
		if err != nil {
			// Invalid token, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		ctx := SetClaimsInContext(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
