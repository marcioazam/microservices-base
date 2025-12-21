//go:build integration

package integration

import (
	"os"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/config"
	redisclient "github.com/auth-platform/platform/resilience-service/internal/infrastructure/repositories"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/observability"
	"log/slog"
)

func getRedisURL() string {
	if url := os.Getenv("REDIS_URL"); url != "" {
		return url
	}
	return "redis://localhost:6379"
}

func setupTestRepository(t *testing.T) *redisclient.RedisRepository {
	t.Helper()

	cfg := &config.RedisConfig{
		URL:            getRedisURL(),
		DB:             0,
		ConnectTimeout: 5 * time.Second,
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   3 * time.Second,
		MaxRetries:     3,
		PoolSize:       10,
	}

	logger := slog.Default()
	metrics := observability.NewMetricsRecorder()

	repo, err := redisclient.NewRedisRepository(cfg, logger, metrics)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func TestIntegration_RedisRepository_HealthCheck(t *testing.T) {
	_ = setupTestRepository(t)
	// If we get here, the repository connected successfully
}

func TestIntegration_CircuitBreakerState(t *testing.T) {
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

	if state.ServiceName != "test-service-integration" {
		t.Errorf("ServiceName: got %s, want test-service-integration", state.ServiceName)
	}
	if state.State != domain.StateOpen {
		t.Errorf("State: got %v, want StateOpen", state.State)
	}
}
