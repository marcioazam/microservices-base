package authplatform

import (
	"context"
	"net/http"
	"regexp"
	"strings"
)

// ContextKey is the type for context keys.
type ContextKey string

const (
	// ClaimsContextKey is the context key for JWT claims.
	ClaimsContextKey ContextKey = "authplatform_claims"
)

// MiddlewareConfig holds configuration for HTTP middleware.
type MiddlewareConfig struct {
	SkipPatterns   []*regexp.Regexp
	ErrorHandler   ErrorHandler
	TokenExtractor TokenExtractor
	Audience       string
	Issuer         string
	RequiredClaims []string
	CookieName     string
	UseCookie      bool
}

// ErrorHandler handles authentication errors in middleware.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// MiddlewareOption configures the middleware.
type MiddlewareOption func(*MiddlewareConfig)

// WithSkipPatterns sets URL patterns to skip authentication.
func WithSkipPatterns(patterns ...string) MiddlewareOption {
	return func(c *MiddlewareConfig) {
		for _, p := range patterns {
			if re, err := regexp.Compile(p); err == nil {
				c.SkipPatterns = append(c.SkipPatterns, re)
			}
		}
	}
}

// WithErrorHandler sets a custom error handler.
func WithErrorHandler(handler ErrorHandler) MiddlewareOption {
	return func(c *MiddlewareConfig) {
		c.ErrorHandler = handler
	}
}

// WithTokenExtractor sets a custom token extractor.
func WithTokenExtractor(extractor TokenExtractor) MiddlewareOption {
	return func(c *MiddlewareConfig) {
		c.TokenExtractor = extractor
	}
}

// WithCookieExtraction enables token extraction from cookies.
func WithCookieExtraction(cookieName string) MiddlewareOption {
	return func(c *MiddlewareConfig) {
		c.CookieName = cookieName
		c.UseCookie = true
	}
}

// WithAudience sets the expected audience for token validation.
func WithAudience(audience string) MiddlewareOption {
	return func(c *MiddlewareConfig) {
		c.Audience = audience
	}
}

// WithIssuer sets the expected issuer for token validation.
func WithIssuer(issuer string) MiddlewareOption {
	return func(c *MiddlewareConfig) {
		c.Issuer = issuer
	}
}

// WithRequiredClaims sets claims that must be present in the token.
func WithRequiredClaims(claims ...string) MiddlewareOption {
	return func(c *MiddlewareConfig) {
		c.RequiredClaims = claims
	}
}


// defaultErrorHandler is the default error handler for middleware.
func defaultErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
}

// Middleware returns HTTP middleware for token validation with options.
func (c *Client) Middleware(opts ...MiddlewareOption) func(http.Handler) http.Handler {
	config := &MiddlewareConfig{
		ErrorHandler: defaultErrorHandler,
	}

	for _, opt := range opts {
		opt(config)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check skip patterns
			if shouldSkip(r.URL.Path, config.SkipPatterns) {
				next.ServeHTTP(w, r)
				return
			}

			// Create appropriate extractor based on config
			var extractor TokenExtractor
			if config.UseCookie && config.CookieName != "" {
				extractor = NewCookieTokenExtractor(r, config.CookieName, TokenSchemeBearer)
			} else if config.TokenExtractor != nil {
				extractor = config.TokenExtractor
			} else {
				extractor = NewHTTPTokenExtractor(r)
			}

			// Extract token
			token, scheme, err := extractor.Extract(r.Context())
			if err != nil {
				config.ErrorHandler(w, r, &SDKError{
					Code:    ErrCodeUnauthorized,
					Message: "missing or invalid token",
					Cause:   err,
				})
				return
			}

			// Validate token
			claims, err := c.validateWithOptions(r.Context(), token, scheme, config)
			if err != nil {
				config.ErrorHandler(w, r, err)
				return
			}

			// Store claims in context
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (c *Client) validateWithOptions(ctx context.Context, token string, scheme TokenScheme, config *MiddlewareConfig) (*Claims, error) {
	// For DPoP tokens, additional validation would be needed
	if scheme == TokenSchemeDPoP {
		// DPoP validation requires the proof header - handled separately
		return nil, &SDKError{
			Code:    ErrCodeValidation,
			Message: "DPoP validation requires proof header",
		}
	}

	// Use JWKS cache if available, otherwise fall back to client validation
	if c.jwksCache != nil {
		opts := ValidationOptions{
			Audience:       config.Audience,
			Issuer:         config.Issuer,
			RequiredClaims: config.RequiredClaims,
		}
		return c.jwksCache.ValidateTokenWithOpts(ctx, token, opts)
	}

	return c.ValidateToken(ctx, token)
}

func shouldSkip(path string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

// MiddlewareFunc returns middleware as a function type (legacy support).
func (c *Client) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	middleware := c.Middleware()
	return func(w http.ResponseWriter, r *http.Request) {
		middleware(next).ServeHTTP(w, r)
	}
}

// GetClaimsFromContext extracts claims from request context.
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	return claims, ok
}

// extractBearerToken extracts bearer token from Authorization header (legacy).
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return parts[1]
}
