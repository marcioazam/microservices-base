// Package client provides unit tests for configuration.
package client

import (
	"os"
	"testing"
	"time"

	"github.com/auth-platform/sdk-go/src/client"
)

func TestDefaultConfig(t *testing.T) {
	c := client.DefaultConfig()

	if c.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", c.Timeout)
	}
	if c.JWKSCacheTTL != time.Hour {
		t.Errorf("JWKSCacheTTL = %v, want 1h", c.JWKSCacheTTL)
	}
	if c.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", c.MaxRetries)
	}
	if c.BaseDelay != time.Second {
		t.Errorf("BaseDelay = %v, want 1s", c.BaseDelay)
	}
	if c.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", c.MaxDelay)
	}
	if c.DPoPEnabled {
		t.Error("DPoPEnabled should be false by default")
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	c := &client.Config{
		BaseURL:      "https://auth.example.com",
		ClientID:     "my-client",
		Timeout:      30 * time.Second,
		JWKSCacheTTL: time.Hour,
		MaxRetries:   3,
		BaseDelay:    time.Second,
		MaxDelay:     30 * time.Second,
	}

	if err := c.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfig_Validate_MissingBaseURL(t *testing.T) {
	c := &client.Config{
		ClientID: "my-client",
		Timeout:  30 * time.Second,
	}
	c.ApplyDefaults()

	if err := c.Validate(); err == nil {
		t.Error("expected error for missing BaseURL")
	}
}

func TestConfig_Validate_MissingClientID(t *testing.T) {
	c := &client.Config{
		BaseURL: "https://auth.example.com",
		Timeout: 30 * time.Second,
	}
	c.ApplyDefaults()

	if err := c.Validate(); err == nil {
		t.Error("expected error for missing ClientID")
	}
}

func TestConfig_Validate_InvalidTimeout(t *testing.T) {
	c := &client.Config{
		BaseURL:  "https://auth.example.com",
		ClientID: "my-client",
		Timeout:  0,
	}
	c.ApplyDefaults()
	c.Timeout = 0 // Override default

	if err := c.Validate(); err == nil {
		t.Error("expected error for zero Timeout")
	}
}

func TestConfig_Validate_JWKSCacheTTLTooShort(t *testing.T) {
	c := &client.Config{
		BaseURL:      "https://auth.example.com",
		ClientID:     "my-client",
		Timeout:      30 * time.Second,
		JWKSCacheTTL: 30 * time.Second, // Less than 1 minute
		BaseDelay:    time.Second,
		MaxDelay:     30 * time.Second,
	}

	if err := c.Validate(); err == nil {
		t.Error("expected error for JWKSCacheTTL < 1 minute")
	}
}

func TestConfig_Validate_JWKSCacheTTLTooLong(t *testing.T) {
	c := &client.Config{
		BaseURL:      "https://auth.example.com",
		ClientID:     "my-client",
		Timeout:      30 * time.Second,
		JWKSCacheTTL: 25 * time.Hour, // More than 24 hours
		BaseDelay:    time.Second,
		MaxDelay:     30 * time.Second,
	}

	if err := c.Validate(); err == nil {
		t.Error("expected error for JWKSCacheTTL > 24 hours")
	}
}

func TestConfig_ApplyDefaults(t *testing.T) {
	c := &client.Config{
		BaseURL:  "https://auth.example.com",
		ClientID: "my-client",
	}

	c.ApplyDefaults()

	if c.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", c.Timeout)
	}
	if c.JWKSCacheTTL != time.Hour {
		t.Errorf("JWKSCacheTTL = %v, want 1h", c.JWKSCacheTTL)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("AUTH_PLATFORM_BASE_URL", "https://test.example.com")
	os.Setenv("AUTH_PLATFORM_CLIENT_ID", "test-client")
	os.Setenv("AUTH_PLATFORM_CLIENT_SECRET", "test-secret")
	os.Setenv("AUTH_PLATFORM_TIMEOUT", "60s")
	os.Setenv("AUTH_PLATFORM_DPOP_ENABLED", "true")
	defer func() {
		os.Unsetenv("AUTH_PLATFORM_BASE_URL")
		os.Unsetenv("AUTH_PLATFORM_CLIENT_ID")
		os.Unsetenv("AUTH_PLATFORM_CLIENT_SECRET")
		os.Unsetenv("AUTH_PLATFORM_TIMEOUT")
		os.Unsetenv("AUTH_PLATFORM_DPOP_ENABLED")
	}()

	c := client.LoadFromEnv()

	if c.BaseURL != "https://test.example.com" {
		t.Errorf("BaseURL = %s, want https://test.example.com", c.BaseURL)
	}
	if c.ClientID != "test-client" {
		t.Errorf("ClientID = %s, want test-client", c.ClientID)
	}
	if c.ClientSecret != "test-secret" {
		t.Errorf("ClientSecret = %s, want test-secret", c.ClientSecret)
	}
	if c.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want 60s", c.Timeout)
	}
	if !c.DPoPEnabled {
		t.Error("DPoPEnabled should be true")
	}
}

func TestNewConfig_WithOptions(t *testing.T) {
	c := client.NewConfig(
		client.WithBaseURL("https://custom.example.com"),
		client.WithClientID("custom-client"),
		client.WithTimeout(45*time.Second),
		client.WithDPoP(true),
	)

	if c.BaseURL != "https://custom.example.com" {
		t.Errorf("BaseURL = %s, want https://custom.example.com", c.BaseURL)
	}
	if c.ClientID != "custom-client" {
		t.Errorf("ClientID = %s, want custom-client", c.ClientID)
	}
	if c.Timeout != 45*time.Second {
		t.Errorf("Timeout = %v, want 45s", c.Timeout)
	}
	if !c.DPoPEnabled {
		t.Error("DPoPEnabled should be true")
	}
}

func TestNewConfigFromEnv_WithOverrides(t *testing.T) {
	os.Setenv("AUTH_PLATFORM_BASE_URL", "https://env.example.com")
	defer os.Unsetenv("AUTH_PLATFORM_BASE_URL")

	c := client.NewConfigFromEnv(
		client.WithClientID("override-client"),
	)

	if c.BaseURL != "https://env.example.com" {
		t.Errorf("BaseURL = %s, want https://env.example.com", c.BaseURL)
	}
	if c.ClientID != "override-client" {
		t.Errorf("ClientID = %s, want override-client", c.ClientID)
	}
}
