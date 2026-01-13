// Package client provides property-based tests for configuration.
package client

import (
	"testing"
	"time"

	"github.com/auth-platform/sdk-go/src/client"
	"pgregory.net/rapid"
)

// Property 26: Config Environment Loading
func TestProperty_ConfigEnvironmentLoading(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// LoadFromEnv should always return a non-nil config
		c := client.LoadFromEnv()
		if c == nil {
			t.Fatal("LoadFromEnv should never return nil")
		}
	})
}

// Property 27: Config Validation
func TestProperty_ConfigValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseURL := "https://" + rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "domain") + ".com"
		clientID := rapid.StringMatching(`[a-z0-9-]{5,20}`).Draw(t, "clientID")

		c := &client.Config{
			BaseURL:      baseURL,
			ClientID:     clientID,
			Timeout:      30 * time.Second,
			JWKSCacheTTL: time.Hour,
			MaxRetries:   3,
			BaseDelay:    time.Second,
			MaxDelay:     30 * time.Second,
		}

		err := c.Validate()
		if err != nil {
			t.Fatalf("valid config should pass validation: %v", err)
		}
	})
}

// Property 28: Config Defaults
func TestProperty_ConfigDefaults(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := client.DefaultConfig()

		// All defaults should be set
		if c.Timeout <= 0 {
			t.Fatal("default Timeout should be positive")
		}
		if c.JWKSCacheTTL <= 0 {
			t.Fatal("default JWKSCacheTTL should be positive")
		}
		if c.MaxRetries < 0 {
			t.Fatal("default MaxRetries should be non-negative")
		}
		if c.BaseDelay <= 0 {
			t.Fatal("default BaseDelay should be positive")
		}
		if c.MaxDelay <= 0 {
			t.Fatal("default MaxDelay should be positive")
		}
	})
}

// Property 29: Config Functional Options
func TestProperty_ConfigFunctionalOptions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseURL := "https://" + rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "domain") + ".com"
		clientID := rapid.StringMatching(`[a-z0-9-]{5,20}`).Draw(t, "clientID")
		timeoutSec := rapid.IntRange(10, 120).Draw(t, "timeoutSec")
		maxRetries := rapid.IntRange(0, 10).Draw(t, "maxRetries")

		c := client.NewConfig(
			client.WithBaseURL(baseURL),
			client.WithClientID(clientID),
			client.WithTimeout(time.Duration(timeoutSec)*time.Second),
			client.WithMaxRetries(maxRetries),
		)

		if c.BaseURL != baseURL {
			t.Fatalf("BaseURL = %s, want %s", c.BaseURL, baseURL)
		}
		if c.ClientID != clientID {
			t.Fatalf("ClientID = %s, want %s", c.ClientID, clientID)
		}
		if c.Timeout != time.Duration(timeoutSec)*time.Second {
			t.Fatalf("Timeout = %v, want %ds", c.Timeout, timeoutSec)
		}
		if c.MaxRetries != maxRetries {
			t.Fatalf("MaxRetries = %d, want %d", c.MaxRetries, maxRetries)
		}
	})
}

// Property: ApplyDefaults doesn't override set values
func TestProperty_ApplyDefaultsPreservesSetValues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		timeout := time.Duration(rapid.IntRange(10, 120).Draw(t, "timeout")) * time.Second

		c := &client.Config{
			Timeout: timeout,
		}
		c.ApplyDefaults()

		if c.Timeout != timeout {
			t.Fatalf("ApplyDefaults should preserve set Timeout: got %v, want %v", c.Timeout, timeout)
		}
	})
}

// Property: Validation fails for invalid JWKSCacheTTL
func TestProperty_ValidationFailsInvalidJWKSCacheTTL(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate TTL outside valid range (< 1 minute or > 24 hours)
		invalidTTL := rapid.SampledFrom([]time.Duration{
			30 * time.Second,
			25 * time.Hour,
		}).Draw(t, "invalidTTL")

		c := &client.Config{
			BaseURL:      "https://example.com",
			ClientID:     "test",
			Timeout:      30 * time.Second,
			JWKSCacheTTL: invalidTTL,
			BaseDelay:    time.Second,
			MaxDelay:     30 * time.Second,
		}

		err := c.Validate()
		if err == nil {
			t.Fatalf("validation should fail for JWKSCacheTTL = %v", invalidTTL)
		}
	})
}

// Property: Validation fails when MaxDelay < BaseDelay
func TestProperty_ValidationFailsMaxDelayLessThanBase(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseDelay := time.Duration(rapid.IntRange(10, 60).Draw(t, "baseDelay")) * time.Second
		maxDelay := time.Duration(rapid.IntRange(1, 9).Draw(t, "maxDelay")) * time.Second

		c := &client.Config{
			BaseURL:      "https://example.com",
			ClientID:     "test",
			Timeout:      30 * time.Second,
			JWKSCacheTTL: time.Hour,
			BaseDelay:    baseDelay,
			MaxDelay:     maxDelay,
		}

		err := c.Validate()
		if err == nil {
			t.Fatal("validation should fail when MaxDelay < BaseDelay")
		}
	})
}

// Property: DPoP option is correctly applied
func TestProperty_DPoPOptionApplied(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		enabled := rapid.Bool().Draw(t, "enabled")

		c := client.NewConfig(client.WithDPoP(enabled))

		if c.DPoPEnabled != enabled {
			t.Fatalf("DPoPEnabled = %v, want %v", c.DPoPEnabled, enabled)
		}
	})
}

// Property: All options can be combined
func TestProperty_AllOptionsCombined(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseURL := "https://example.com"
		clientID := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "clientID")
		clientSecret := rapid.StringMatching(`[a-z0-9]{20,40}`).Draw(t, "clientSecret")
		timeout := time.Duration(rapid.IntRange(10, 60).Draw(t, "timeout")) * time.Second
		jwksTTL := time.Duration(rapid.IntRange(1, 24).Draw(t, "jwksTTL")) * time.Hour
		maxRetries := rapid.IntRange(0, 5).Draw(t, "maxRetries")
		baseDelay := time.Duration(rapid.IntRange(1, 5).Draw(t, "baseDelay")) * time.Second
		maxDelay := time.Duration(rapid.IntRange(10, 60).Draw(t, "maxDelay")) * time.Second
		dpop := rapid.Bool().Draw(t, "dpop")

		c := client.NewConfig(
			client.WithBaseURL(baseURL),
			client.WithClientID(clientID),
			client.WithClientSecret(clientSecret),
			client.WithTimeout(timeout),
			client.WithJWKSCacheTTL(jwksTTL),
			client.WithMaxRetries(maxRetries),
			client.WithBaseDelay(baseDelay),
			client.WithMaxDelay(maxDelay),
			client.WithDPoP(dpop),
		)

		if c.BaseURL != baseURL {
			t.Fatal("BaseURL not set correctly")
		}
		if c.ClientID != clientID {
			t.Fatal("ClientID not set correctly")
		}
		if c.ClientSecret != clientSecret {
			t.Fatal("ClientSecret not set correctly")
		}
		if c.Timeout != timeout {
			t.Fatal("Timeout not set correctly")
		}
		if c.JWKSCacheTTL != jwksTTL {
			t.Fatal("JWKSCacheTTL not set correctly")
		}
		if c.MaxRetries != maxRetries {
			t.Fatal("MaxRetries not set correctly")
		}
		if c.BaseDelay != baseDelay {
			t.Fatal("BaseDelay not set correctly")
		}
		if c.MaxDelay != maxDelay {
			t.Fatal("MaxDelay not set correctly")
		}
		if c.DPoPEnabled != dpop {
			t.Fatal("DPoPEnabled not set correctly")
		}
	})
}
