package client

import "time"

// ConfigOption configures the client.
type ConfigOption func(*Config)

// WithBaseURL sets the base URL.
func WithBaseURL(url string) ConfigOption {
	return func(c *Config) { c.BaseURL = url }
}

// WithClientID sets the client ID.
func WithClientID(id string) ConfigOption {
	return func(c *Config) { c.ClientID = id }
}

// WithClientSecret sets the client secret.
func WithClientSecret(secret string) ConfigOption {
	return func(c *Config) { c.ClientSecret = secret }
}

// WithTimeout sets the HTTP timeout.
func WithTimeout(d time.Duration) ConfigOption {
	return func(c *Config) { c.Timeout = d }
}

// WithJWKSCacheTTL sets the JWKS cache TTL.
func WithJWKSCacheTTL(d time.Duration) ConfigOption {
	return func(c *Config) { c.JWKSCacheTTL = d }
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(n int) ConfigOption {
	return func(c *Config) { c.MaxRetries = n }
}

// WithBaseDelay sets the base retry delay.
func WithBaseDelay(d time.Duration) ConfigOption {
	return func(c *Config) { c.BaseDelay = d }
}

// WithMaxDelay sets the maximum retry delay.
func WithMaxDelay(d time.Duration) ConfigOption {
	return func(c *Config) { c.MaxDelay = d }
}

// WithDPoP enables or disables DPoP.
func WithDPoP(enabled bool) ConfigOption {
	return func(c *Config) { c.DPoPEnabled = enabled }
}

// NewConfig creates a new configuration with options.
func NewConfig(opts ...ConfigOption) *Config {
	c := DefaultConfig()
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewConfigFromEnv creates configuration from environment with overrides.
func NewConfigFromEnv(opts ...ConfigOption) *Config {
	c := LoadFromEnv()
	for _, opt := range opts {
		opt(c)
	}
	return c
}
