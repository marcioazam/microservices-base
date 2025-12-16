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
)

// Config holds client configuration.
type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Timeout      time.Duration
	JWKSCacheTTL time.Duration
}

// Option is a functional option for configuring the client.
type Option func(*Client)

// WithTimeout sets the HTTP timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithJWKSCacheTTL sets the JWKS cache TTL.
func WithJWKSCacheTTL(d time.Duration) Option {
	return func(c *Client) {
		c.jwksCache.ttl = d
	}
}

// Client is the Auth Platform SDK client.
type Client struct {
	config     Config
	httpClient *http.Client
	jwksCache  *JWKSCache
	tokens     *TokenData
	tokensMu   sync.RWMutex
}

// New creates a new Auth Platform client.
func New(config Config, opts ...Option) (*Client, error) {
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
		config:     config,
		httpClient: &http.Client{Timeout: config.Timeout},
		jwksCache:  NewJWKSCache(config.BaseURL+"/.well-known/jwks.json", config.JWKSCacheTTL),
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
	if c.config.ClientSecret == "" {
		return nil, fmt.Errorf("%w: client_secret required", ErrInvalidConfig)
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
		return nil, fmt.Errorf("%w: %v", ErrNetwork, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
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

	return &tokenResp, nil
}

// ValidateToken validates a JWT and returns claims.
func (c *Client) ValidateToken(ctx context.Context, token string) (*Claims, error) {
	return c.jwksCache.ValidateToken(ctx, token, c.config.ClientID)
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
	c.tokensMu.RLock()
	refreshToken := c.tokens.RefreshToken
	c.tokensMu.RUnlock()

	if refreshToken == "" {
		return ErrTokenRefresh
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
		return fmt.Errorf("%w: %v", ErrNetwork, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		c.tokensMu.Lock()
		c.tokens = nil
		c.tokensMu.Unlock()
		return fmt.Errorf("%w: %v", ErrTokenRefresh, err)
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
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

	return nil
}

func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%w: %v", ErrNetwork, err)
			time.Sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			retryAfter := resp.Header.Get("Retry-After")
			delay := time.Duration(1<<attempt) * time.Second
			if retryAfter != "" {
				if d, err := time.ParseDuration(retryAfter + "s"); err == nil {
					delay = d
				}
			}
			lastErr = ErrRateLimited
			time.Sleep(delay)
			continue
		}

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("%w: status %d: %s", ErrNetwork, resp.StatusCode, string(body))
		}

		return resp, nil
	}

	return nil, lastErr
}

// Close releases resources.
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
