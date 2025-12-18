//go:build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/infra/redis"
)

func getRedisURL() string {
	if url := os.Getenv("REDIS_URL"); url != "" {
		return url
	}
	return "redis://localhost:6379"
}

func setupTestClient(t *testing.T) *redis.Client {
	t.Helper()
	client, err := redis.NewClient(redis.Config{
		URL:    getRedisURL(),
		Prefix: "test:resilience:",
	})
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestIntegration_CircuitStateRoundTrip(t *testing.T) {
	client := setupTestClient(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Millisecond)
	failureTime := now.Add(-time.Minute)
	state := domain.CircuitBreakerState{
		ServiceName:     "test-service-integration",
		State:           domain.StateOpen,
		FailureCount:    5,
		SuccessCount:    0,
		LastFailureTime: &failureTime,
		LastStateChange: now,
		Version:         1,
	}

	err := client.SaveCircuitState(ctx, state)
	if err != nil {
		t.Fatalf("SaveCircuitState failed: %v", err)
	}

	loaded, err := client.LoadCircuitState(ctx, state.ServiceName)
	if err != nil {
		t.Fatalf("LoadCircuitState failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadCircuitState returned nil")
	}
	if loaded.ServiceName != state.ServiceName {
		t.Errorf("ServiceName: got %s, want %s", loaded.ServiceName, state.ServiceName)
	}
}

func TestIntegration_HealthCheck(t *testing.T) {
	client := setupTestClient(t)
	ctx := context.Background()
	err := client.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}
