package authplatform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

// PKCEGenerator generates PKCE code verifiers and challenges.
type PKCEGenerator interface {
	// GenerateVerifier generates a cryptographically secure code verifier.
	GenerateVerifier() (string, error)
	// ComputeChallenge computes the S256 code challenge from a verifier.
	ComputeChallenge(verifier string) string
}

// DefaultPKCEGenerator is the default implementation of PKCEGenerator.
type DefaultPKCEGenerator struct {
	// VerifierLength is the length of generated verifiers (43-128).
	VerifierLength int
}

// NewPKCEGenerator creates a new PKCE generator with default settings.
func NewPKCEGenerator() *DefaultPKCEGenerator {
	return &DefaultPKCEGenerator{
		VerifierLength: 64, // Default to 64 characters
	}
}

// unreservedChars contains the unreserved characters for PKCE verifiers.
// Per RFC 7636: [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"
const unreservedChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"

// GenerateVerifier generates a cryptographically secure code verifier.
// The verifier is between 43 and 128 characters from the unreserved set.
func (g *DefaultPKCEGenerator) GenerateVerifier() (string, error) {
	length := g.VerifierLength
	if length < 43 {
		length = 43
	}
	if length > 128 {
		length = 128
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", &SDKError{
			Code:    ErrCodePKCEInvalid,
			Message: "failed to generate random bytes",
			Cause:   err,
		}
	}

	verifier := make([]byte, length)
	for i := 0; i < length; i++ {
		verifier[i] = unreservedChars[int(bytes[i])%len(unreservedChars)]
	}

	return string(verifier), nil
}

// ComputeChallenge computes the S256 code challenge from a verifier.
// The challenge is the base64url-encoded SHA-256 hash of the verifier.
func (g *DefaultPKCEGenerator) ComputeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// PKCEPair holds a code verifier and its corresponding challenge.
type PKCEPair struct {
	Verifier  string
	Challenge string
	Method    string // Always "S256"
}

// GeneratePKCE generates a new PKCE verifier/challenge pair.
func GeneratePKCE() (*PKCEPair, error) {
	gen := NewPKCEGenerator()
	verifier, err := gen.GenerateVerifier()
	if err != nil {
		return nil, err
	}
	return &PKCEPair{
		Verifier:  verifier,
		Challenge: gen.ComputeChallenge(verifier),
		Method:    "S256",
	}, nil
}

// VerifyPKCE verifies that a code verifier matches a code challenge.
func VerifyPKCE(verifier, challenge string) bool {
	gen := NewPKCEGenerator()
	computed := gen.ComputeChallenge(verifier)
	return computed == challenge
}

// AuthorizationRequest represents an OAuth 2.0 authorization request.
type AuthorizationRequest struct {
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	ResponseType        string
	Nonce               string
	AdditionalParams    map[string]string
}

// BuildAuthorizationURL builds the authorization URL for the OAuth flow.
func (c *Client) BuildAuthorizationURL(req AuthorizationRequest) (string, error) {
	if req.ClientID == "" {
		req.ClientID = c.config.ClientID
	}
	if req.ResponseType == "" {
		req.ResponseType = "code"
	}

	u, err := url.Parse(c.config.BaseURL + "/oauth/authorize")
	if err != nil {
		return "", &SDKError{
			Code:    ErrCodeInvalidConfig,
			Message: "invalid base URL",
			Cause:   err,
		}
	}

	q := u.Query()
	q.Set("client_id", req.ClientID)
	q.Set("response_type", req.ResponseType)

	if req.RedirectURI != "" {
		q.Set("redirect_uri", req.RedirectURI)
	}
	if req.Scope != "" {
		q.Set("scope", req.Scope)
	}
	if req.State != "" {
		q.Set("state", req.State)
	}
	if req.CodeChallenge != "" {
		q.Set("code_challenge", req.CodeChallenge)
		q.Set("code_challenge_method", req.CodeChallengeMethod)
		if req.CodeChallengeMethod == "" {
			q.Set("code_challenge_method", "S256")
		}
	}
	if req.Nonce != "" {
		q.Set("nonce", req.Nonce)
	}
	for k, v := range req.AdditionalParams {
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}


// TokenExchangeRequest represents an OAuth 2.0 token exchange request.
type TokenExchangeRequest struct {
	Code         string
	RedirectURI  string
	CodeVerifier string
	ClientID     string
	ClientSecret string
}

// ExchangeCode exchanges an authorization code for tokens.
func (c *Client) ExchangeCode(ctx context.Context, req TokenExchangeRequest) (*TokenResponse, error) {
	if req.Code == "" {
		return nil, &SDKError{
			Code:    ErrCodeValidation,
			Message: "authorization code is required",
		}
	}

	data := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {req.Code},
	}

	if req.ClientID != "" {
		data.Set("client_id", req.ClientID)
	} else {
		data.Set("client_id", c.config.ClientID)
	}

	if req.RedirectURI != "" {
		data.Set("redirect_uri", req.RedirectURI)
	}

	if req.CodeVerifier != "" {
		data.Set("code_verifier", req.CodeVerifier)
	}

	if req.ClientSecret != "" {
		data.Set("client_secret", req.ClientSecret)
	} else if c.config.ClientSecret != "" {
		data.Set("client_secret", c.config.ClientSecret)
	}

	return c.tokenRequest(ctx, data)
}

// RefreshTokenRequest represents a token refresh request.
type RefreshTokenRequest struct {
	RefreshToken string
	Scope        string
}

// RefreshToken refreshes an access token using a refresh token.
func (c *Client) RefreshToken(ctx context.Context, req RefreshTokenRequest) (*TokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, &SDKError{
			Code:    ErrCodeValidation,
			Message: "refresh token is required",
		}
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {req.RefreshToken},
		"client_id":     {c.config.ClientID},
	}

	if req.Scope != "" {
		data.Set("scope", req.Scope)
	}

	if c.config.ClientSecret != "" {
		data.Set("client_secret", c.config.ClientSecret)
	}

	return c.tokenRequest(ctx, data)
}

func (c *Client) tokenRequest(ctx context.Context, data url.Values) (*TokenResponse, error) {
	req, err := newRequestWithContext(
		ctx,
		"POST",
		c.config.BaseURL+"/oauth/token",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, &SDKError{
			Code:    ErrCodeNetwork,
			Message: "failed to create request",
			Cause:   err,
		}
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := decodeJSON(resp.Body, &tokenResp); err != nil {
		return nil, &SDKError{
			Code:    ErrCodeNetwork,
			Message: "failed to decode token response",
			Cause:   err,
		}
	}

	return &tokenResp, nil
}

// GenerateState generates a cryptographically secure state parameter.
func GenerateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", &SDKError{
			Code:    ErrCodeNetwork,
			Message: "failed to generate state",
			Cause:   err,
		}
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// GenerateNonce generates a cryptographically secure nonce.
func GenerateNonce() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", &SDKError{
			Code:    ErrCodeNetwork,
			Message: "failed to generate nonce",
			Cause:   err,
		}
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// IsUnreservedChar checks if a character is in the PKCE unreserved set.
func IsUnreservedChar(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '.' || c == '_' || c == '~'
}

// ValidateVerifier validates that a code verifier meets PKCE requirements.
func ValidateVerifier(verifier string) error {
	if len(verifier) < 43 {
		return &SDKError{
			Code:    ErrCodePKCEInvalid,
			Message: fmt.Sprintf("verifier too short: %d < 43", len(verifier)),
		}
	}
	if len(verifier) > 128 {
		return &SDKError{
			Code:    ErrCodePKCEInvalid,
			Message: fmt.Sprintf("verifier too long: %d > 128", len(verifier)),
		}
	}
	for i, c := range verifier {
		if !IsUnreservedChar(c) {
			return &SDKError{
				Code:    ErrCodePKCEInvalid,
				Message: fmt.Sprintf("invalid character at position %d: %c", i, c),
			}
		}
	}
	return nil
}
