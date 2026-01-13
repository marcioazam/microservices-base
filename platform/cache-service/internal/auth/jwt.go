// Package auth provides authentication and authorization functionality.
package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrMissingToken indicates no token was provided.
	ErrMissingToken = errors.New("missing authorization token")
	// ErrInvalidToken indicates the token is invalid.
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken indicates the token has expired.
	ErrExpiredToken = errors.New("token has expired")
	// ErrInvalidClaims indicates the token claims are invalid.
	ErrInvalidClaims = errors.New("invalid token claims")
)

// Claims represents the JWT claims.
type Claims struct {
	jwt.RegisteredClaims
	Namespace string   `json:"namespace,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
}

// JWTValidator validates JWT tokens.
type JWTValidator struct {
	secret []byte
	issuer string
}

// NewJWTValidator creates a new JWT validator.
func NewJWTValidator(secret, issuer string) *JWTValidator {
	return &JWTValidator{
		secret: []byte(secret),
		issuer: issuer,
	}
}

// Validate validates a JWT token and returns the claims.
func (v *JWTValidator) Validate(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrMissingToken
	}

	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	tokenString = strings.TrimSpace(tokenString)

	if tokenString == "" {
		return nil, ErrMissingToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return v.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	// Validate issuer if configured
	if v.issuer != "" && claims.Issuer != v.issuer {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// GenerateToken generates a new JWT token (for testing purposes).
func (v *JWTValidator) GenerateToken(namespace string, scopes []string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    v.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
		Namespace: namespace,
		Scopes:    scopes,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(v.secret)
}

// Context keys for authentication.
type contextKey string

const (
	// ClaimsContextKey is the context key for JWT claims.
	ClaimsContextKey contextKey = "jwt_claims"
)

// GetClaimsFromContext retrieves JWT claims from context.
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	return claims, ok
}

// SetClaimsInContext sets JWT claims in context.
func SetClaimsInContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ClaimsContextKey, claims)
}

// GetNamespaceFromContext retrieves the namespace from context.
func GetNamespaceFromContext(ctx context.Context) (string, bool) {
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.Namespace, claims.Namespace != ""
}

// HasScope checks if the claims have a specific scope.
func (c *Claims) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}
