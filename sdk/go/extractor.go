package authplatform

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

// TokenScheme represents the authentication scheme.
type TokenScheme string

const (
	// TokenSchemeBearer represents Bearer token authentication.
	TokenSchemeBearer TokenScheme = "Bearer"
	// TokenSchemeDPoP represents DPoP token authentication.
	TokenSchemeDPoP TokenScheme = "DPoP"
	// TokenSchemeUnknown represents an unknown token scheme.
	TokenSchemeUnknown TokenScheme = ""
)

// TokenExtractor extracts authentication tokens from requests.
type TokenExtractor interface {
	// Extract extracts the token and scheme from the context or request.
	Extract(ctx context.Context) (token string, scheme TokenScheme, err error)
}

// HTTPTokenExtractor extracts tokens from HTTP Authorization headers.
type HTTPTokenExtractor struct {
	request *http.Request
}

// NewHTTPTokenExtractor creates a new HTTP token extractor.
func NewHTTPTokenExtractor(r *http.Request) *HTTPTokenExtractor {
	return &HTTPTokenExtractor{request: r}
}

// Extract extracts the token from the HTTP Authorization header.
func (e *HTTPTokenExtractor) Extract(ctx context.Context) (string, TokenScheme, error) {
	if e.request == nil {
		return "", TokenSchemeUnknown, NewError(ErrCodeUnauthorized, "no request provided")
	}

	auth := e.request.Header.Get("Authorization")
	if auth == "" {
		return "", TokenSchemeUnknown, NewError(ErrCodeUnauthorized, "missing authorization header")
	}

	return parseAuthorizationHeader(auth)
}

// GRPCTokenExtractor extracts tokens from gRPC metadata.
type GRPCTokenExtractor struct{}

// NewGRPCTokenExtractor creates a new gRPC token extractor.
func NewGRPCTokenExtractor() *GRPCTokenExtractor {
	return &GRPCTokenExtractor{}
}

// Extract extracts the token from gRPC metadata.
func (e *GRPCTokenExtractor) Extract(ctx context.Context) (string, TokenScheme, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", TokenSchemeUnknown, NewError(ErrCodeUnauthorized, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", TokenSchemeUnknown, NewError(ErrCodeUnauthorized, "missing authorization header")
	}

	return parseAuthorizationHeader(values[0])
}

// CookieTokenExtractor extracts tokens from HTTP cookies.
type CookieTokenExtractor struct {
	request    *http.Request
	cookieName string
	scheme     TokenScheme
}

// NewCookieTokenExtractor creates a new cookie token extractor.
func NewCookieTokenExtractor(r *http.Request, cookieName string, scheme TokenScheme) *CookieTokenExtractor {
	return &CookieTokenExtractor{
		request:    r,
		cookieName: cookieName,
		scheme:     scheme,
	}
}

// Extract extracts the token from the specified cookie.
func (e *CookieTokenExtractor) Extract(ctx context.Context) (string, TokenScheme, error) {
	if e.request == nil {
		return "", TokenSchemeUnknown, NewError(ErrCodeUnauthorized, "no request provided")
	}

	cookie, err := e.request.Cookie(e.cookieName)
	if err != nil {
		return "", TokenSchemeUnknown, NewError(ErrCodeUnauthorized, "missing token cookie")
	}

	if cookie.Value == "" {
		return "", TokenSchemeUnknown, NewError(ErrCodeUnauthorized, "empty token cookie")
	}

	scheme := e.scheme
	if scheme == TokenSchemeUnknown {
		scheme = TokenSchemeBearer
	}

	return cookie.Value, scheme, nil
}

// ChainedTokenExtractor tries multiple extractors in order.
type ChainedTokenExtractor struct {
	extractors []TokenExtractor
}

// NewChainedTokenExtractor creates a new chained token extractor.
func NewChainedTokenExtractor(extractors ...TokenExtractor) *ChainedTokenExtractor {
	return &ChainedTokenExtractor{extractors: extractors}
}

// Extract tries each extractor in order until one succeeds.
func (e *ChainedTokenExtractor) Extract(ctx context.Context) (string, TokenScheme, error) {
	var lastErr error
	for _, extractor := range e.extractors {
		token, scheme, err := extractor.Extract(ctx)
		if err == nil {
			return token, scheme, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", TokenSchemeUnknown, lastErr
	}
	return "", TokenSchemeUnknown, NewError(ErrCodeUnauthorized, "no token found")
}

// parseAuthorizationHeader parses an Authorization header value.
func parseAuthorizationHeader(auth string) (string, TokenScheme, error) {
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		return "", TokenSchemeUnknown, NewError(ErrCodeUnauthorized, "invalid authorization format")
	}

	schemeStr := strings.ToLower(parts[0])
	token := parts[1]

	if token == "" {
		return "", TokenSchemeUnknown, NewError(ErrCodeUnauthorized, "empty token")
	}

	var scheme TokenScheme
	switch schemeStr {
	case "bearer":
		scheme = TokenSchemeBearer
	case "dpop":
		scheme = TokenSchemeDPoP
	default:
		return "", TokenSchemeUnknown, NewError(ErrCodeUnauthorized, "unsupported authorization scheme")
	}

	return token, scheme, nil
}

// FormatAuthorizationHeader formats a token with its scheme for use in headers.
func FormatAuthorizationHeader(token string, scheme TokenScheme) string {
	return string(scheme) + " " + token
}

// ExtractBearerToken is a convenience function to extract a Bearer token from an HTTP request.
func ExtractBearerToken(r *http.Request) (string, error) {
	extractor := NewHTTPTokenExtractor(r)
	token, scheme, err := extractor.Extract(r.Context())
	if err != nil {
		return "", err
	}
	if scheme != TokenSchemeBearer {
		return "", NewError(ErrCodeUnauthorized, "expected Bearer token")
	}
	return token, nil
}

// ExtractDPoPToken is a convenience function to extract a DPoP token from an HTTP request.
func ExtractDPoPToken(r *http.Request) (string, error) {
	extractor := NewHTTPTokenExtractor(r)
	token, scheme, err := extractor.Extract(r.Context())
	if err != nil {
		return "", err
	}
	if scheme != TokenSchemeDPoP {
		return "", NewError(ErrCodeUnauthorized, "expected DPoP token")
	}
	return token, nil
}

// GetDPoPProof extracts the DPoP proof from the DPoP header.
func GetDPoPProof(r *http.Request) (string, error) {
	proof := r.Header.Get("DPoP")
	if proof == "" {
		return "", NewError(ErrCodeDPoPRequired, "missing DPoP header")
	}
	return proof, nil
}
