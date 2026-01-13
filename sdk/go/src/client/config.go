// Package client provides the main SDK client and configuration.
package client

import (
	"os"
	"strconv"
	"time"

	"github.com/auth-platform/sdk-go/src/errors"
)

// Config holds SDK configuration.
type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Timeout      time.Duration
	JWKSCacheTTL time.Duration
	MaxRetries   int
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	DPoPEnabled  bool
}

// DefaultConfig returns default configuration values.
func DefaultConfig() *Config {
	return &Config{
		Timeout:      30 * time.Second,
		JWKSCacheTTL: time.Hour,
		MaxRetries:   3,
		BaseDelay:    time.Second,
		MaxDelay:     30 * time.Second,
		DPoPEnabled:  false,
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return errors.NewError(errors.ErrCodeInvalidConfig, "BaseURL is required")
	}
	if c.ClientID == "" {
		return errors.NewError(errors.ErrCodeInvalidConfig, "ClientID is required")
	}
	if c.Timeout <= 0 {
		return errors.NewError(errors.ErrCodeInvalidConfig, "Timeout must be positive")
	}
	if c.JWKSCacheTTL < time.Minute {
		return errors.NewError(errors.ErrCodeInvalidConfig, "JWKSCacheTTL must be at least 1 minute")
	}
	if c.JWKSCacheTTL > 24*time.Hour {
		return errors.NewError(errors.ErrCodeInvalidConfig, "JWKSCacheTTL must not exceed 24 hours")
	}
	if c.MaxRetries < 0 {
		return errors.NewError(errors.ErrCodeInvalidConfig, "MaxRetries must be non-negative")
	}
	if c.BaseDelay <= 0 {
		return errors.NewError(errors.ErrCodeInvalidConfig, "BaseDelay must be positive")
	}
	if c.MaxDelay < c.BaseDelay {
		return errors.NewError(errors.ErrCodeInvalidConfig, "MaxDelay must be >= BaseDelay")
	}
	return nil
}

// ApplyDefaults applies default values for unset fields.
func (c *Config) ApplyDefaults() {
	defaults := DefaultConfig()
	if c.Timeout == 0 {
		c.Timeout = defaults.Timeout
	}
	if c.JWKSCacheTTL == 0 {
		c.JWKSCacheTTL = defaults.JWKSCacheTTL
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = defaults.MaxRetries
	}
	if c.BaseDelay == 0 {
		c.BaseDelay = defaults.BaseDelay
	}
	if c.MaxDelay == 0 {
		c.MaxDelay = defaults.MaxDelay
	}
}

// LoadFromEnv loads configuration from environment variables.
func LoadFromEnv() *Config {
	c := DefaultConfig()

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

	return c
}
