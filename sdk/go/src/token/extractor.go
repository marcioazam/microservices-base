package token

import (
	"context"
	"net/http"
	"strings"

	"github.com/auth-platform/sdk-go/src/errors"
	"google.golang.org/grpc/metadata"
)

// Extractor extracts authentication tokens from requests.
type Extractor interface {
	// Extract extracts the token and scheme from the context or request.
	Extract(ctx context.Context) (token string, scheme TokenScheme, err error)
}

// HTTPExtractor extracts tokens from HTTP Authorization headers.
type HTTPExtractor struct {
	request *http.Request
}

// NewHTTPExtractor creates a new HTTP token extractor.
func NewHTTPExtractor(r *http.Request) *HTTPExtractor {
	return &HTTPExtractor{request: r}
}

// Extract extracts the token from the HTTP Authorization header.
func (e *HTTPExtractor) Extract(ctx context.Context) (string, TokenScheme, error) {
	if e.request == nil {
		return "", SchemeUnknown, errors.NewError(errors.ErrCodeUnauthorized, "no request provided")
	}

	auth := e.request.Header.Get("Authorization")
	if auth == "" {
		return "", SchemeUnknown, errors.NewError(errors.ErrCodeTokenMissing, "missing authorization header")
	}

	return ParseAuthorizationHeader(auth)
}

// GRPCExtractor extracts tokens from gRPC metadata.
type GRPCExtractor struct{}

// NewGRPCExtractor creates a new gRPC token extractor.
func NewGRPCExtractor() *GRPCExtractor {
	return &GRPCExtractor{}
}

// Extract extracts the token from gRPC metadata.
func (e *GRPCExtractor) Extract(ctx context.Context) (string, TokenScheme, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", SchemeUnknown, errors.NewError(errors.ErrCodeUnauthorized, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", SchemeUnknown, errors.NewError(errors.ErrCodeTokenMissing, "missing authorization header")
	}

	return ParseAuthorizationHeader(values[0])
}

// CookieExtractor extracts tokens from HTTP cookies.
type CookieExtractor struct {
	request    *http.Request
	cookieName string
	scheme     TokenScheme
}

// NewCookieExtractor creates a new cookie token extractor.
func NewCookieExtractor(r *http.Request, cookieName string, scheme TokenScheme) *CookieExtractor {
	return &CookieExtractor{
		request:    r,
		cookieName: cookieName,
		scheme:     scheme,
	}
}

// Extract extracts the token from the specified cookie.
func (e *CookieExtractor) Extract(ctx context.Context) (string, TokenScheme, error) {
	if e.request == nil {
		return "", SchemeUnknown, errors.NewError(errors.ErrCodeUnauthorized, "no request provided")
	}

	cookie, err := e.request.Cookie(e.cookieName)
	if err != nil {
		return "", SchemeUnknown, errors.NewError(errors.ErrCodeTokenMissing, "missing token cookie")
	}

	if cookie.Value == "" {
		return "", SchemeUnknown, errors.NewError(errors.ErrCodeTokenMissing, "empty token cookie")
	}

	scheme := e.scheme
	if scheme == SchemeUnknown {
		scheme = SchemeBearer
	}

	return cookie.Value, scheme, nil
}

// ChainedExtractor tries multiple extractors in order.
type ChainedExtractor struct {
	extractors []Extractor
}

// NewChainedExtractor creates a new chained token extractor.
func NewChainedExtractor(extractors ...Extractor) *ChainedExtractor {
	return &ChainedExtractor{extractors: extractors}
}

// Extract tries each extractor in order until one succeeds.
func (e *ChainedExtractor) Extract(ctx context.Context) (string, TokenScheme, error) {
	var lastErr error
	for _, extractor := range e.extractors {
		token, scheme, err := extractor.Extract(ctx)
		if err == nil {
			return token, scheme, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", SchemeUnknown, lastErr
	}
	return "", SchemeUnknown, errors.NewError(errors.ErrCodeTokenMissing, "no token found")
}

// ParseAuthorizationHeader parses an Authorization header value.
func ParseAuthorizationHeader(auth string) (string, TokenScheme, error) {
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		return "", SchemeUnknown, errors.NewError(errors.ErrCodeTokenInvalid, "invalid authorization format")
	}

	schemeStr := strings.ToLower(parts[0])
	token := parts[1]

	if token == "" {
		return "", SchemeUnknown, errors.NewError(errors.ErrCodeTokenMissing, "empty token")
	}

	scheme := ParseScheme(schemeStr)
	if scheme == SchemeUnknown {
		return "", SchemeUnknown, errors.NewError(errors.ErrCodeTokenInvalid, "unsupported authorization scheme")
	}

	return token, scheme, nil
}

// FormatAuthorizationHeader formats a token with its scheme for use in headers.
func FormatAuthorizationHeader(token string, scheme TokenScheme) string {
	return string(scheme) + " " + token
}

// ExtractBearerToken is a convenience function to extract a Bearer token from an HTTP request.
func ExtractBearerToken(r *http.Request) (string, error) {
	extractor := NewHTTPExtractor(r)
	token, scheme, err := extractor.Extract(r.Context())
	if err != nil {
		return "", err
	}
	if scheme != SchemeBearer {
		return "", errors.NewError(errors.ErrCodeTokenInvalid, "expected Bearer token")
	}
	return token, nil
}

// ExtractDPoPToken is a convenience function to extract a DPoP token from an HTTP request.
func ExtractDPoPToken(r *http.Request) (string, error) {
	extractor := NewHTTPExtractor(r)
	token, scheme, err := extractor.Extract(r.Context())
	if err != nil {
		return "", err
	}
	if scheme != SchemeDPoP {
		return "", errors.NewError(errors.ErrCodeTokenInvalid, "expected DPoP token")
	}
	return token, nil
}

// GetDPoPProof extracts the DPoP proof from the DPoP header.
func GetDPoPProof(r *http.Request) (string, error) {
	proof := r.Header.Get("DPoP")
	if proof == "" {
		return "", errors.NewError(errors.ErrCodeDPoPRequired, "missing DPoP header")
	}
	return proof, nil
}
