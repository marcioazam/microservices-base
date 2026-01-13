// Package integration contains integration tests for IAM Policy Service.
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/cache"
	"github.com/auth-platform/iam-policy-service/internal/config"
	"github.com/auth-platform/iam-policy-service/internal/logging"
)

// TestCacheServiceIntegration tests cache-service integration with fallback.
func TestCacheServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := config.CacheConfig{
		Address:        "localhost:50051", // May not be running
		Namespace:      "iam-policy-test",
		Timeout:        time.Second,
		TTL:            5 * time.Minute,
		LocalFallback:  true,
		LocalCacheSize: 1000,
	}

	dc, err := cache.NewDecisionCache(cfg)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer dc.Close()

	ctx := context.Background()

	// Test set and get
	input := map[string]interface{}{
		"subject": map[string]interface{}{"id": "user123"},
		"action":  "read",
	}

	decision := &cache.Decision{
		Allowed: true,
		Reason:  "test",
	}

	err = dc.Set(ctx, input, decision)
	if err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	retrieved, found := dc.Get(ctx, input)
	if !found {
		t.Fatal("decision not found")
	}

	if retrieved.Allowed != decision.Allowed {
		t.Errorf("allowed mismatch: expected %v, got %v", decision.Allowed, retrieved.Allowed)
	}
}

// TestLoggingServiceIntegration tests logging-service integration with fallback.
func TestLoggingServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := config.LoggingConfig{
		Address:       "localhost:50052", // May not be running
		ServiceName:   "iam-policy-test",
		MinLevel:      "debug",
		LocalFallback: true,
		BufferSize:    100,
		FlushInterval: time.Second,
	}

	logger, err := logging.NewLogger(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	ctx := context.Background()

	// Test logging at different levels
	logger.Debug(ctx, "debug message", logging.String("key", "value"))
	logger.Info(ctx, "info message", logging.Int("count", 42))
	logger.Warn(ctx, "warn message", logging.Bool("flag", true))

	// Flush and verify no errors
	err = logger.Flush()
	if err != nil {
		t.Errorf("flush failed: %v", err)
	}
}

// TestEndToEndAuthorizationFlow tests the complete authorization flow.
func TestEndToEndAuthorizationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test requires the full service to be running
	// Skip if not in integration test environment
	t.Skip("requires running service - run with integration test suite")
}

// TestCacheFallbackBehavior tests cache fallback when remote is unavailable.
func TestCacheFallbackBehavior(t *testing.T) {
	cfg := config.CacheConfig{
		Address:        "localhost:99999", // Invalid port
		Namespace:      "iam-policy-test",
		Timeout:        100 * time.Millisecond,
		TTL:            5 * time.Minute,
		LocalFallback:  true,
		LocalCacheSize: 1000,
	}

	dc, err := cache.NewDecisionCache(cfg)
	if err != nil {
		t.Fatalf("failed to create cache with fallback: %v", err)
	}
	defer dc.Close()

	ctx := context.Background()

	// Should work with local fallback
	input := map[string]interface{}{"test": "value"}
	decision := &cache.Decision{Allowed: true}

	err = dc.Set(ctx, input, decision)
	if err != nil {
		t.Errorf("set should succeed with local fallback: %v", err)
	}

	retrieved, found := dc.Get(ctx, input)
	if !found {
		t.Error("should find in local cache")
	}

	if retrieved.Allowed != decision.Allowed {
		t.Error("retrieved decision should match")
	}
}

// TestLoggingFallbackBehavior tests logging fallback when remote is unavailable.
func TestLoggingFallbackBehavior(t *testing.T) {
	cfg := config.LoggingConfig{
		Address:       "localhost:99999", // Invalid port
		ServiceName:   "iam-policy-test",
		MinLevel:      "info",
		LocalFallback: true,
		BufferSize:    100,
		FlushInterval: time.Second,
	}

	logger, err := logging.NewLogger(cfg)
	if err != nil {
		t.Fatalf("failed to create logger with fallback: %v", err)
	}
	defer logger.Close()

	ctx := context.Background()

	// Should not panic with local fallback
	logger.Info(ctx, "test message")
	logger.Error(ctx, "error message", logging.Error(nil))

	err = logger.Flush()
	if err != nil {
		t.Logf("flush returned error (expected with fallback): %v", err)
	}
}
