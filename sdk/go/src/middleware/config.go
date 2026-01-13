// Package middleware provides HTTP and gRPC middleware for authentication.
package middleware

import (
	"net/http"
	"regexp"

	"github.com/auth-platform/sdk-go/src/token"
)

// ErrorHandler handles authentication errors.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// Config holds middleware configuration.
type Config struct {
	SkipPatterns   []*regexp.Regexp
	ErrorHandler   ErrorHandler
	TokenExtractor token.Extractor
	Audience       string
	Issuer         string
	RequiredClaims []string
}

// Option configures middleware.
type Option func(*Config)

// WithSkipPatterns sets URL patterns to skip authentication.
func WithSkipPatterns(patterns ...string) Option {
	return func(c *Config) {
		c.SkipPatterns = make([]*regexp.Regexp, 0, len(patterns))
		for _, p := range patterns {
			if re, err := regexp.Compile(p); err == nil {
				c.SkipPatterns = append(c.SkipPatterns, re)
			}
		}
	}
}

// WithErrorHandler sets a custom error handler.
func WithErrorHandler(handler ErrorHandler) Option {
	return func(c *Config) {
		c.ErrorHandler = handler
	}
}

// WithTokenExtractor sets a custom token extractor.
func WithTokenExtractor(extractor token.Extractor) Option {
	return func(c *Config) {
		c.TokenExtractor = extractor
	}
}

// WithAudience sets the expected audience for token validation.
func WithAudience(audience string) Option {
	return func(c *Config) {
		c.Audience = audience
	}
}

// WithIssuer sets the expected issuer for token validation.
func WithIssuer(issuer string) Option {
	return func(c *Config) {
		c.Issuer = issuer
	}
}

// WithRequiredClaims sets claims that must be present in the token.
func WithRequiredClaims(claims ...string) Option {
	return func(c *Config) {
		c.RequiredClaims = claims
	}
}

// DefaultConfig returns a default middleware configuration.
func DefaultConfig() *Config {
	return &Config{
		SkipPatterns: []*regexp.Regexp{},
		ErrorHandler: defaultErrorHandler,
	}
}

// NewConfig creates a new configuration with options.
func NewConfig(opts ...Option) *Config {
	c := DefaultConfig()
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// ShouldSkip checks if the request path matches any skip pattern.
func (c *Config) ShouldSkip(r *http.Request) bool {
	path := r.URL.Path
	for _, pattern := range c.SkipPatterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}
