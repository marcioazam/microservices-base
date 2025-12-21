package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authplatform "github.com/auth-platform/sdk-go"
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
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := authplatform.New(authplatform.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.ClientCredentials(ctx)
	if err != nil {
		t.Fatalf("client credentials failed: %v", err)
	}

	if resp.AccessToken != "test-access-token" {
		t.Errorf("expected access_token 'test-access-token', got %q", resp.AccessToken)
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected token_type 'Bearer', got %q", resp.TokenType)
	}
}

func TestClientCredentialsWithDPoP(t *testing.T) {
	var dpopHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			dpopHeader = r.Header.Get("DPoP")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "dpop-bound-token",
				"token_type":   "DPoP",
				"expires_in":   3600,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := authplatform.New(authplatform.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		DPoPEnabled:  true,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.ClientCredentials(ctx)
	if err != nil {
		t.Fatalf("client credentials with DPoP failed: %v", err)
	}

	if dpopHeader == "" {
		t.Error("expected DPoP header to be set")
	}
	if resp.TokenType != "DPoP" {
		t.Errorf("expected token_type 'DPoP', got %q", resp.TokenType)
	}
}

func TestClientRetryOnRateLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "success-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	policy := authplatform.NewRetryPolicy(
		authplatform.WithMaxRetries(3),
		authplatform.WithBaseDelay(10*time.Millisecond),
		authplatform.WithMaxDelay(100*time.Millisecond),
	)

	client, err := authplatform.New(
		authplatform.Config{
			BaseURL:      server.URL,
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		},
		authplatform.WithRetryPolicy(policy),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	resp, err := client.ClientCredentials(ctx)
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}

	if resp.AccessToken != "success-token" {
		t.Errorf("expected 'success-token', got %q", resp.AccessToken)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestGetAccessTokenRefresh(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "new-access-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"refresh_token": "new-refresh-token",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := authplatform.New(authplatform.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// First get tokens
	_, err = client.ClientCredentials(ctx)
	if err != nil {
		t.Fatalf("client credentials failed: %v", err)
	}

	// Get access token (should not refresh yet)
	token, err := client.GetAccessToken(ctx)
	if err != nil {
		t.Fatalf("get access token failed: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  authplatform.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: authplatform.Config{
				BaseURL:      "https://auth.example.com",
				ClientID:     "client-id",
				Timeout:      30 * time.Second,
				JWKSCacheTTL: time.Hour,
				MaxRetries:   3,
				BaseDelay:    time.Second,
				MaxDelay:     30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing base url",
			config: authplatform.Config{
				ClientID: "client-id",
				Timeout:  30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "missing client id",
			config: authplatform.Config{
				BaseURL: "https://auth.example.com",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: authplatform.Config{
				BaseURL:  "https://auth.example.com",
				ClientID: "client-id",
				Timeout:  -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.ApplyDefaults()
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("AUTH_PLATFORM_BASE_URL", "https://test.example.com")
	t.Setenv("AUTH_PLATFORM_CLIENT_ID", "env-client-id")
	t.Setenv("AUTH_PLATFORM_TIMEOUT", "45s")
	t.Setenv("AUTH_PLATFORM_DPOP_ENABLED", "true")

	config := authplatform.LoadFromEnv()

	if config.BaseURL != "https://test.example.com" {
		t.Errorf("expected BaseURL 'https://test.example.com', got %q", config.BaseURL)
	}
	if config.ClientID != "env-client-id" {
		t.Errorf("expected ClientID 'env-client-id', got %q", config.ClientID)
	}
	if config.Timeout != 45*time.Second {
		t.Errorf("expected Timeout 45s, got %v", config.Timeout)
	}
	if !config.DPoPEnabled {
		t.Error("expected DPoPEnabled to be true")
	}
}
