// Package authplatform provides a Go SDK for the Auth Platform.
package authplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/auth-platform/sdk-go/internal/observability"
)

// ClientOption is a functional option for configuring the client.
type ClientOption func(*Client)

// WithTimeout sets the HTTP timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithJWKSCacheTTL sets the JWKS cache TTL.
func WithJWKSCacheTTL(d time.Duration) ClientOption {
	return func(c *Client) {
		c.jwksCache.ttl = d
	}
}

// WithRetryPolicy sets a custom retry policy.
func WithRetryPolicy(p *RetryPolicy) ClientOption {
	return func(c *Client) {
		c.retryPolicy = p
	}
}

// WithDPoPProver sets a DPoP prover for sender-constrained tokens.
func WithDPoPProver(prover DPoPProver) ClientOption {
	return func(c *Client) {
		c.dpopProver = prover
	}
}

// WithTracer sets a custom tracer.
func WithTracer(t *observability.Tracer) ClientOption {
	return func(c *Client) {
		c.tracer = t
	}
}

// WithLogger sets a custom logger.
func WithLogger(l observability.Logger) ClientOption {
	return func(c *Client) {
		c.logger = l
	}
}

// Client is the Auth Platform SDK client.
type Client struct {
	config      Config
	httpClient  *http.Client
	jwksCache   *JWKSCache
	tokens      *TokenData
	tokensMu    sync.RWMutex
	retryPolicy *RetryPolicy
	dpopProver  DPoPProver
	tracer      *observability.Tracer
	logger      observability.Logger
}

// New creates a new Auth Platform client.
func New(config Config, opts ...ClientOption) (*Client, error) {
	if config.BaseURL == "" {
		return nil, ErrInvalidConfig
	}
	if config.ClientID == "" {
		return nil, ErrInvalidConfig
	}

	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.JWKSCacheTTL == 0 {
		config.JWKSCacheTTL = time.Hour
	}

	c := &Client{
		config:      config,
		httpClient:  &http.Client{Timeout: config.Timeout},
		jwksCache:   NewJWKSCache(config.BaseURL+"/.well-known/jwks.json", config.JWKSCacheTTL),
		retryPolicy: DefaultRetryPolicy(),
		tracer:      observability.NewTracer(),
		logger:      observability.NewDefaultLogger(observability.LogLevelInfo),
	}

	// Initialize DPoP if enabled
	if config.DPoPEnabled {
		keyPair, err := GenerateES256KeyPair()
		if err != nil {
			return nil, err
		}
		c.dpopProver = NewDPoPProver(keyPair)
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// TokenData holds token information.
type TokenData struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	TokenType    string
	Scope        string
}

// TokenResponse represents an OAuth token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// Claims represents JWT claims.
type Claims struct {
	Subject   string   `json:"sub"`
	Issuer    string   `json:"iss"`
	Audience  []string `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	Scope     string   `json:"scope,omitempty"`
	ClientID  string   `json:"client_id,omitempty"`
}

// ClientCredentials obtains a token using client credentials flow.
func (c *Client) ClientCredentials(ctx context.Context) (*TokenResponse, error) {
	ctx, span := c.tracer.ClientCredentialsSpan(ctx)
	defer span.End()

	if c.config.ClientSecret == "" {
		err := fmt.Errorf("%w: client_secret required", ErrInvalidConfig)
		observability.RecordError(span, err)
		return nil, err
	}

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.config.ClientID},
		"client_secret": {c.config.ClientSecret},
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.config.BaseURL+"/oauth/token",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		observability.RecordError(span, err)
		return nil, fmt.Errorf("%w: %v", ErrNetwork, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add DPoP proof if enabled
	if c.dpopProver != nil {
		proof, err := c.dpopProver.GenerateProof(ctx, http.MethodPost, c.config.BaseURL+"/oauth/token", "")
		if err != nil {
			observability.RecordError(span, err)
			return nil, err
		}
		req.Header.Set("DPoP", proof)
	}

	resp, err := c.doWithRetryPolicy(ctx, req)
	if err != nil {
		observability.RecordError(span, err)
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		observability.RecordError(span, err)
		return nil, fmt.Errorf("%w: %v", ErrNetwork, err)
	}

	// Store tokens
	c.tokensMu.Lock()
	c.tokens = &TokenData{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
	}
	c.tokensMu.Unlock()

	observability.SetSuccess(span)
	c.logger.Info(ctx, "client credentials flow completed")
	return &tokenResp, nil
}

// ValidateToken validates a JWT and returns claims.
func (c *Client) ValidateToken(ctx context.Context, token string) (*Claims, error) {
	ctx, span := c.tracer.TokenValidationSpan(ctx)
	defer span.End()

	claims, err := c.jwksCache.ValidateToken(ctx, token, c.config.ClientID)
	if err != nil {
		observability.RecordError(span, err)
		observability.LogTokenValidation(c.logger, ctx, false, err)
		return nil, err
	}

	observability.SetSuccess(span)
	observability.LogTokenValidation(c.logger, ctx, true, nil)
	return claims, nil
}

// GetAccessToken returns a valid access token, refreshing if necessary.
func (c *Client) GetAccessToken(ctx context.Context) (string, error) {
	c.tokensMu.RLock()
	tokens := c.tokens
	c.tokensMu.RUnlock()

	if tokens == nil {
		return "", ErrTokenExpired
	}

	// Check if token needs refresh (with 1 minute buffer)
	if time.Now().Add(time.Minute).After(tokens.ExpiresAt) {
		if tokens.RefreshToken != "" {
			if err := c.refreshTokens(ctx); err != nil {
				return "", err
			}
			c.tokensMu.RLock()
			tokens = c.tokens
			c.tokensMu.RUnlock()
		} else {
			return "", ErrTokenExpired
		}
	}

	return tokens.AccessToken, nil
}

func (c *Client) refreshTokens(ctx context.Context) error {
	ctx, span := c.tracer.TokenRefreshSpan(ctx)
	defer span.End()

	c.tokensMu.RLock()
	refreshToken := c.tokens.RefreshToken
	c.tokensMu.RUnlock()

	if refreshToken == "" {
		err := ErrTokenRefresh
		observability.RecordError(span, err)
		return err
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {c.config.ClientID},
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.config.BaseURL+"/oauth/token",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		observability.RecordError(span, err)
		return fmt.Errorf("%w: %v", ErrNetwork, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doWithRetryPolicy(ctx, req)
	if err != nil {
		c.tokensMu.Lock()
		c.tokens = nil
		c.tokensMu.Unlock()
		observability.RecordError(span, err)
		observability.LogTokenRefresh(c.logger, ctx, false, err)
		return fmt.Errorf("%w: %v", ErrTokenRefresh, err)
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		observability.RecordError(span, err)
		return fmt.Errorf("%w: %v", ErrNetwork, err)
	}

	c.tokensMu.Lock()
	c.tokens = &TokenData{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
	}
	c.tokensMu.Unlock()

	observability.SetSuccess(span)
	observability.LogTokenRefresh(c.logger, ctx, true, nil)
	return nil
}

// doWithRetryPolicy executes HTTP request with retry policy.
func (c *Client) doWithRetryPolicy(ctx context.Context, req *http.Request) (*http.Response, error) {
	return RetryWithResponse(ctx, c.retryPolicy, func(ctx context.Context) (*http.Response, error) {
		return c.httpClient.Do(req)
	})
}

func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	return c.doWithRetryPolicy(ctx, req)
}

// Close releases resources.
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// newRequestWithContext creates a new HTTP request with context.
func newRequestWithContext(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, body)
}

// decodeJSON decodes JSON from a reader into a value.
func decodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}
