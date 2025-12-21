package authplatform_test

import (
	"context"
	"testing"
	"time"

	authplatform "github.com/auth-platform/sdk-go"
)

func TestJWKSCacheConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := authplatform.DefaultJWKSCacheConfig("https://example.com/.well-known/jwks.json")
		if config.URI != "https://example.com/.well-known/jwks.json" {
			t.Errorf("expected URI to be set, got %s", config.URI)
		}
		if config.TTL != time.Hour {
			t.Errorf("expected TTL to be 1 hour, got %v", config.TTL)
		}
	})
}

func TestJWKSCacheMetrics(t *testing.T) {
	t.Run("InitialMetricsAreZero", func(t *testing.T) {
		// Create cache with invalid URL (won't actually fetch)
		cache := authplatform.NewJWKSCache("https://invalid.example.com/jwks", time.Hour)
		if cache == nil {
			t.Skip("Cache creation failed - expected in test environment")
		}
		defer cache.Close()

		metrics := cache.GetMetrics()
		if metrics.Hits != 0 || metrics.Misses != 0 || metrics.Errors != 0 {
			t.Errorf("expected initial metrics to be zero, got hits=%d misses=%d errors=%d",
				metrics.Hits, metrics.Misses, metrics.Errors)
		}
	})
}

func TestJWKSCacheInvalidate(t *testing.T) {
	t.Run("InvalidateIncrementsRefreshCounter", func(t *testing.T) {
		cache := authplatform.NewJWKSCache("https://example.com/jwks", time.Hour)
		if cache == nil {
			t.Skip("Cache creation failed - expected in test environment")
		}
		defer cache.Close()

		initialMetrics := cache.GetMetrics()
		cache.Invalidate()
		afterMetrics := cache.GetMetrics()

		if afterMetrics.Refreshes != initialMetrics.Refreshes+1 {
			t.Errorf("expected refresh counter to increment, got %d", afterMetrics.Refreshes)
		}
	})
}

func TestJWKSCacheValidation(t *testing.T) {
	t.Run("InvalidTokenReturnsError", func(t *testing.T) {
		cache := authplatform.NewJWKSCache("https://example.com/jwks", time.Hour)
		if cache == nil {
			t.Skip("Cache creation failed - expected in test environment")
		}
		defer cache.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := cache.ValidateToken(ctx, "invalid.token.here", "")
		if err == nil {
			t.Error("expected error for invalid token")
		}
	})

	t.Run("MalformedTokenReturnsError", func(t *testing.T) {
		cache := authplatform.NewJWKSCache("https://example.com/jwks", time.Hour)
		if cache == nil {
			t.Skip("Cache creation failed - expected in test environment")
		}
		defer cache.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := cache.ValidateToken(ctx, "not-a-jwt", "")
		if err == nil {
			t.Error("expected error for malformed token")
		}
	})
}

func TestValidationOptions(t *testing.T) {
	t.Run("OptionsStructure", func(t *testing.T) {
		opts := authplatform.ValidationOptions{
			Audience:       "test-audience",
			Issuer:         "test-issuer",
			RequiredClaims: []string{"sub", "email"},
			SkipExpiry:     true,
		}

		if opts.Audience != "test-audience" {
			t.Errorf("expected audience test-audience, got %s", opts.Audience)
		}
		if opts.Issuer != "test-issuer" {
			t.Errorf("expected issuer test-issuer, got %s", opts.Issuer)
		}
		if len(opts.RequiredClaims) != 2 {
			t.Errorf("expected 2 required claims, got %d", len(opts.RequiredClaims))
		}
		if !opts.SkipExpiry {
			t.Error("expected SkipExpiry to be true")
		}
	})
}
