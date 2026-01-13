// Package client provides unit tests for the main client.
package client

import (
	"testing"

	"github.com/auth-platform/sdk-go/src/client"
)

func TestNew_ValidConfig(t *testing.T) {
	c, err := client.New(
		client.WithBaseURL("https://auth.example.com"),
		client.WithClientID("test-client"),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client should not be nil")
	}

	defer c.Close()

	if c.Config().BaseURL != "https://auth.example.com" {
		t.Errorf("BaseURL = %s, want https://auth.example.com", c.Config().BaseURL)
	}
	if c.Config().ClientID != "test-client" {
		t.Errorf("ClientID = %s, want test-client", c.Config().ClientID)
	}
}

func TestNew_MissingBaseURL(t *testing.T) {
	_, err := client.New(
		client.WithClientID("test-client"),
	)

	if err == nil {
		t.Fatal("expected error for missing BaseURL")
	}
}

func TestNew_MissingClientID(t *testing.T) {
	_, err := client.New(
		client.WithBaseURL("https://auth.example.com"),
	)

	if err == nil {
		t.Fatal("expected error for missing ClientID")
	}
}

func TestNew_WithDPoP(t *testing.T) {
	c, err := client.New(
		client.WithBaseURL("https://auth.example.com"),
		client.WithClientID("test-client"),
		client.WithDPoP(true),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer c.Close()

	if c.DPoPProver() == nil {
		t.Error("DPoPProver should not be nil when DPoP is enabled")
	}
}

func TestNew_WithoutDPoP(t *testing.T) {
	c, err := client.New(
		client.WithBaseURL("https://auth.example.com"),
		client.WithClientID("test-client"),
		client.WithDPoP(false),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer c.Close()

	if c.DPoPProver() != nil {
		t.Error("DPoPProver should be nil when DPoP is disabled")
	}
}

func TestNewFromConfig(t *testing.T) {
	config := &client.Config{
		BaseURL:  "https://auth.example.com",
		ClientID: "test-client",
	}

	c, err := client.NewFromConfig(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer c.Close()

	if c.Config().BaseURL != "https://auth.example.com" {
		t.Errorf("BaseURL = %s, want https://auth.example.com", c.Config().BaseURL)
	}
}

func TestClient_HTTPMiddleware(t *testing.T) {
	c, err := client.New(
		client.WithBaseURL("https://auth.example.com"),
		client.WithClientID("test-client"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer c.Close()

	mw := c.HTTPMiddleware()
	if mw == nil {
		t.Error("HTTPMiddleware should not return nil")
	}
}

func TestClient_Close(t *testing.T) {
	c, err := client.New(
		client.WithBaseURL("https://auth.example.com"),
		client.WithClientID("test-client"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Close should not panic
	c.Close()
}
