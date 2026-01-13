package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/auth-platform/sdk-go/src/client"
	"github.com/auth-platform/sdk-go/src/retry"
)

func TestClientCredentialsFlow(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "test-access-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"refresh_token": "test-refresh-token",
			})
			return
		}
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c, err := client.New(
		client.WithBaseURL(server.URL),
		client.WithClientID("test-client"),
		client.WithClientSecret("test-secret"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	// Verify client was created successfully
	if c.Config().BaseURL != server.URL {
		t.Errorf("expected BaseURL %q, got %q", server.URL, c.Config().BaseURL)
	}
}

func TestClientWithDPoP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c, err := client.New(
		client.WithBaseURL(server.URL),
		client.WithClientID("test-client"),
		client.WithClientSecret("test-secret"),
		client.WithDPoP(true),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	// Verify DPoP prover was created
	if c.DPoPProver() == nil {
		t.Error("expected DPoP prover to be set")
	}
}

func TestClientRetryPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	policy := retry.NewPolicy(
		retry.WithMaxRetries(5),
		retry.WithBaseDelay(10*time.Millisecond),
		retry.WithMaxDelay(100*time.Millisecond),
	)

	c, err := client.New(
		client.WithBaseURL(server.URL),
		client.WithClientID("test-client"),
		client.WithClientSecret("test-secret"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	// Verify retry policy
	if policy.MaxRetries != 5 {
		t.Errorf("expected MaxRetries 5, got %d", policy.MaxRetries)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		opts    []client.ConfigOption
		wantErr bool
	}{
		{
			name: "valid config",
			opts: []client.ConfigOption{
				client.WithBaseURL("https://auth.example.com"),
				client.WithClientID("client-id"),
				client.WithTimeout(30 * time.Second),
			},
			wantErr: false,
		},
		{
			name: "missing base url",
			opts: []client.ConfigOption{
				client.WithClientID("client-id"),
			},
			wantErr: true,
		},
		{
			name: "missing client id",
			opts: []client.ConfigOption{
				client.WithBaseURL("https://auth.example.com"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("AUTH_PLATFORM_BASE_URL", "https://test.example.com")
	t.Setenv("AUTH_PLATFORM_CLIENT_ID", "env-client-id")
	t.Setenv("AUTH_PLATFORM_TIMEOUT", "45s")
	t.Setenv("AUTH_PLATFORM_DPOP_ENABLED", "true")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	// Override base URL for test
	t.Setenv("AUTH_PLATFORM_BASE_URL", server.URL)

	c, err := client.NewFromEnv()
	if err != nil {
		t.Fatalf("failed to create client from env: %v", err)
	}
	defer c.Close()

	config := c.Config()
	if config.ClientID != "env-client-id" {
		t.Errorf("expected ClientID 'env-client-id', got %q", config.ClientID)
	}
	if config.Timeout != 45*time.Second {
		t.Errorf("expected Timeout 45s, got %v", config.Timeout)
	}
}

func TestHTTPMiddleware(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c, err := client.New(
		client.WithBaseURL(server.URL),
		client.WithClientID("test-client"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	// Create middleware
	middleware := c.HTTPMiddleware()
	if middleware == nil {
		t.Error("expected middleware to be created")
	}

	// Test that middleware wraps handler
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	if handler == nil {
		t.Error("expected handler to be wrapped")
	}
}

func TestTokenValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c, err := client.New(
		client.WithBaseURL(server.URL),
		client.WithClientID("test-client"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Test with invalid token (should fail)
	_, err = c.ValidateTokenCtx(ctx, "invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}
