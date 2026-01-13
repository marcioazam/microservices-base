package middleware

import (
	"context"

	"github.com/auth-platform/sdk-go/src/types"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	claimsContextKey contextKey = "auth_claims"
	tokenContextKey  contextKey = "auth_token"
)

// ContextWithClaims adds claims to the context.
func ContextWithClaims(ctx context.Context, claims *types.Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// GetClaimsFromContext retrieves claims from the context.
func GetClaimsFromContext(ctx context.Context) (*types.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*types.Claims)
	return claims, ok
}

// ContextWithToken adds the raw token to the context.
func ContextWithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenContextKey, token)
}

// GetTokenFromContext retrieves the raw token from the context.
func GetTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(tokenContextKey).(string)
	return token, ok
}

// MustGetClaims retrieves claims from context, panics if not present.
func MustGetClaims(ctx context.Context) *types.Claims {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		panic("claims not found in context")
	}
	return claims
}

// GetSubject retrieves the subject claim from context.
func GetSubject(ctx context.Context) (string, bool) {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.Subject, claims.Subject != ""
}

// GetClientID retrieves the client_id claim from context.
func GetClientID(ctx context.Context) (string, bool) {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.ClientID, claims.ClientID != ""
}

// GetScope retrieves the scope claim from context.
func GetScope(ctx context.Context) (string, bool) {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.Scope, claims.Scope != ""
}

// HasScope checks if the context contains a specific scope.
func HasScope(ctx context.Context, scope string) bool {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return false
	}
	// Simple space-separated scope check
	scopes := claims.Scope
	if scopes == "" {
		return false
	}
	// Check for exact match or space-bounded match
	if scopes == scope {
		return true
	}
	// Check within space-separated list
	for i := 0; i <= len(scopes)-len(scope); i++ {
		if i == 0 || scopes[i-1] == ' ' {
			if i+len(scope) == len(scopes) || scopes[i+len(scope)] == ' ' {
				if scopes[i:i+len(scope)] == scope {
					return true
				}
			}
		}
	}
	return false
}
