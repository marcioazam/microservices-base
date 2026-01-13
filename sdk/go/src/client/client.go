package client

import (
	"context"
	"net/http"

	"github.com/auth-platform/sdk-go/src/auth"
	"github.com/auth-platform/sdk-go/src/internal/observability"
	"github.com/auth-platform/sdk-go/src/middleware"
	"github.com/auth-platform/sdk-go/src/retry"
	"github.com/auth-platform/sdk-go/src/token"
	"github.com/auth-platform/sdk-go/src/types"
)

// Client is the main SDK client.
type Client struct {
	config      *Config
	httpClient  *http.Client
	jwksCache   *token.JWKSCache
	retryPolicy *retry.Policy
	dpopProver  auth.DPoPProver
	tracer      *observability.Tracer
	logger      *observability.Logger
}

// New creates a new SDK client with the given options.
func New(opts ...ConfigOption) (*Client, error) {
	config := NewConfig(opts...)
	config.ApplyDefaults()

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return newClient(config)
}

// NewFromConfig creates a new SDK client from a config.
func NewFromConfig(config *Config) (*Client, error) {
	config.ApplyDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return newClient(config)
}

// NewFromEnv creates a new SDK client from environment variables.
func NewFromEnv(opts ...ConfigOption) (*Client, error) {
	config := NewConfigFromEnv(opts...)
	config.ApplyDefaults()

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return newClient(config)
}

func newClient(config *Config) (*Client, error) {
	httpClient := &http.Client{Timeout: config.Timeout}

	jwksURI := config.BaseURL + "/.well-known/jwks.json"
	jwksCache := token.NewJWKSCache(jwksURI, config.JWKSCacheTTL)

	retryPolicy := retry.NewPolicy(
		retry.WithMaxRetries(config.MaxRetries),
		retry.WithBaseDelay(config.BaseDelay),
		retry.WithMaxDelay(config.MaxDelay),
	)

	client := &Client{
		config:      config,
		httpClient:  httpClient,
		jwksCache:   jwksCache,
		retryPolicy: retryPolicy,
		tracer:      observability.NewTracer(),
		logger:      observability.NewLogger(),
	}

	if config.DPoPEnabled {
		keyPair, err := auth.GenerateES256KeyPair()
		if err != nil {
			return nil, err
		}
		client.dpopProver = auth.NewDPoPProver(keyPair)
	}

	return client, nil
}

// Close releases resources held by the client.
func (c *Client) Close() {
	if c.jwksCache != nil {
		c.jwksCache.Close()
	}
}

// ValidateTokenCtx validates a JWT token with context.
func (c *Client) ValidateTokenCtx(ctx context.Context, tokenStr string) (*types.Claims, error) {
	ctx, span := c.tracer.TraceTokenValidation(ctx)
	defer span.End()

	claims, err := c.jwksCache.ValidateToken(ctx, tokenStr, c.config.ClientID)
	if err != nil {
		observability.SetSpanError(span, err)
		c.logger.LogTokenValidation(ctx, false, err)
		return nil, err
	}

	observability.SetSpanOK(span)
	c.logger.LogTokenValidation(ctx, true, nil)
	return claims, nil
}

// ValidateTokenWithOpts validates a JWT with custom options.
func (c *Client) ValidateTokenWithOpts(ctx context.Context, tokenStr string, opts types.ValidationOptions) (*types.Claims, error) {
	return c.jwksCache.ValidateTokenWithOpts(ctx, tokenStr, opts)
}

// HTTPMiddleware returns HTTP middleware for authentication.
func (c *Client) HTTPMiddleware(opts ...middleware.Option) func(http.Handler) http.Handler {
	mw := middleware.NewHTTPMiddleware(c, opts...)
	return mw.Middleware()
}

// Config returns the client configuration.
func (c *Client) Config() *Config {
	return c.config
}

// DPoPProver returns the DPoP prover if enabled.
func (c *Client) DPoPProver() auth.DPoPProver {
	return c.dpopProver
}
