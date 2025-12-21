package authplatform

import (
	"net/url"
	"os"
	"strconv"
	"time"
)

// Config holds client configuration.
type Config struct {
	// Required fields
	BaseURL  string `env:"AUTH_PLATFORM_BASE_URL"`
	ClientID string `env:"AUTH_PLATFORM_CLIENT_ID"`

	// Optional fields with defaults
	ClientSecret string        `env:"AUTH_PLATFORM_CLIENT_SECRET"`
	Timeout      time.Duration `env:"AUTH_PLATFORM_TIMEOUT" default:"30s"`
	JWKSCacheTTL time.Duration `env:"AUTH_PLATFORM_JWKS_CACHE_TTL" default:"1h"`

	// Retry configuration
	MaxRetries int           `env:"AUTH_PLATFORM_MAX_RETRIES" default:"3"`
	BaseDelay  time.Duration `env:"AUTH_PLATFORM_BASE_DELAY" default:"1s"`
	MaxDelay   time.Duration `env:"AUTH_PLATFORM_MAX_DELAY" default:"30s"`

	// DPoP configuration
	DPoPEnabled bool   `env:"AUTH_PLATFORM_DPOP_ENABLED" default:"false"`
	DPoPKeyPath string `env:"AUTH_PLATFORM_DPOP_KEY_PATH"`
}

// Validate checks configuration validity.
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return &SDKError{Code: ErrCodeInvalidConfig, Message: "BaseURL is required"}
	}
	if _, err := url.Parse(c.BaseURL); err != nil {
		return &SDKError{Code: ErrCodeInvalidConfig, Message: "BaseURL is invalid", Cause: err}
	}
	if c.ClientID == "" {
		return &SDKError{Code: ErrCodeInvalidConfig, Message: "ClientID is required"}
	}
	if c.Timeout <= 0 {
		return &SDKError{Code: ErrCodeInvalidConfig, Message: "Timeout must be positive"}
	}
	if c.JWKSCacheTTL < time.Minute || c.JWKSCacheTTL > 24*time.Hour {
		return &SDKError{Code: ErrCodeInvalidConfig, Message: "JWKSCacheTTL must be between 1 minute and 24 hours"}
	}
	if c.MaxRetries < 0 {
		return &SDKError{Code: ErrCodeInvalidConfig, Message: "MaxRetries must be non-negative"}
	}
	if c.BaseDelay <= 0 {
		return &SDKError{Code: ErrCodeInvalidConfig, Message: "BaseDelay must be positive"}
	}
	if c.MaxDelay < c.BaseDelay {
		return &SDKError{Code: ErrCodeInvalidConfig, Message: "MaxDelay must be >= BaseDelay"}
	}
	return nil
}

// ApplyDefaults sets default values for unset fields.
func (c *Config) ApplyDefaults() {
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.JWKSCacheTTL == 0 {
		c.JWKSCacheTTL = time.Hour
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.BaseDelay == 0 {
		c.BaseDelay = time.Second
	}
	if c.MaxDelay == 0 {
		c.MaxDelay = 30 * time.Second
	}
}

// LoadFromEnv loads configuration from environment variables.
func LoadFromEnv() *Config {
	c := &Config{}

	if v := os.Getenv("AUTH_PLATFORM_BASE_URL"); v != "" {
		c.BaseURL = v
	}
	if v := os.Getenv("AUTH_PLATFORM_CLIENT_ID"); v != "" {
		c.ClientID = v
	}
	if v := os.Getenv("AUTH_PLATFORM_CLIENT_SECRET"); v != "" {
		c.ClientSecret = v
	}
	if v := os.Getenv("AUTH_PLATFORM_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		}
	}
	if v := os.Getenv("AUTH_PLATFORM_JWKS_CACHE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.JWKSCacheTTL = d
		}
	}
	if v := os.Getenv("AUTH_PLATFORM_MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxRetries = n
		}
	}
	if v := os.Getenv("AUTH_PLATFORM_BASE_DELAY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.BaseDelay = d
		}
	}
	if v := os.Getenv("AUTH_PLATFORM_MAX_DELAY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.MaxDelay = d
		}
	}
	if v := os.Getenv("AUTH_PLATFORM_DPOP_ENABLED"); v != "" {
		c.DPoPEnabled = v == "true" || v == "1"
	}
	if v := os.Getenv("AUTH_PLATFORM_DPOP_KEY_PATH"); v != "" {
		c.DPoPKeyPath = v
	}

	c.ApplyDefaults()
	return c
}
