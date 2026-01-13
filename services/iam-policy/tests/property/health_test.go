package property

import (
	"context"
	"errors"
	"testing"

	"github.com/auth-platform/iam-policy-service/internal/health"
	"pgregory.net/rapid"
)

// TestHealthCheckDegradedStatus validates Property 11: Health Check Degraded Status
// For any health check when crypto-service is unavailable, the status SHALL be
// DEGRADED (not UNHEALTHY).
// **Validates: Requirements 7.2**
func TestHealthCheckDegradedStatus(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Simulate crypto service unavailable
		unavailableChecker := func(ctx context.Context) (*health.CryptoHealthStatus, error) {
			return &health.CryptoHealthStatus{
				Connected: false,
				LatencyMs: 0,
			}, nil
		}

		// Create health check
		check := health.CryptoHealthCheck(unavailableChecker)

		// Run check
		result := check(context.Background())

		// Status should be DEGRADED, not UNHEALTHY
		if result.Status != health.StatusDegraded {
			t.Errorf("expected status DEGRADED when crypto service unavailable, got %s", result.Status)
		}

		// Should not be UNHEALTHY
		if result.Status == health.StatusUnhealthy {
			t.Error("crypto service unavailability should not cause UNHEALTHY status")
		}
	})
}

func TestHealthCheckDegradedOnError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random error message
		errorMsg := rapid.StringMatching(`[a-zA-Z0-9 ]{5,50}`).Draw(t, "errorMsg")

		// Simulate crypto service error
		errorChecker := func(ctx context.Context) (*health.CryptoHealthStatus, error) {
			return nil, errors.New(errorMsg)
		}

		// Create health check
		check := health.CryptoHealthCheck(errorChecker)

		// Run check
		result := check(context.Background())

		// Status should be DEGRADED on error
		if result.Status != health.StatusDegraded {
			t.Errorf("expected status DEGRADED on error, got %s", result.Status)
		}

		// Message should contain error info
		if result.Message == "" {
			t.Error("result message should not be empty on error")
		}
	})
}

func TestHealthCheckHealthyWhenConnected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random latency
		latencyMs := rapid.Int64Range(1, 100).Draw(t, "latencyMs")

		// Simulate crypto service connected
		connectedChecker := func(ctx context.Context) (*health.CryptoHealthStatus, error) {
			return &health.CryptoHealthStatus{
				Connected:    true,
				LatencyMs:    latencyMs,
				HSMConnected: rapid.Bool().Draw(t, "hsmConnected"),
				KMSConnected: rapid.Bool().Draw(t, "kmsConnected"),
			}, nil
		}

		// Create health check
		check := health.CryptoHealthCheck(connectedChecker)

		// Run check
		result := check(context.Background())

		// Status should be HEALTHY when connected
		if result.Status != health.StatusHealthy {
			t.Errorf("expected status HEALTHY when crypto service connected, got %s", result.Status)
		}
	})
}

func TestOverallStatusWithCryptoCheck(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		manager := health.NewManager()

		// Register crypto health check that returns degraded
		cryptoConnected := rapid.Bool().Draw(t, "cryptoConnected")

		cryptoChecker := func(ctx context.Context) (*health.CryptoHealthStatus, error) {
			return &health.CryptoHealthStatus{
				Connected: cryptoConnected,
				LatencyMs: 10,
			}, nil
		}

		manager.RegisterCheck("crypto", health.CryptoHealthCheck(cryptoChecker))

		// Get overall status
		status := manager.GetOverallStatus(context.Background())

		// If crypto is not connected, overall should be DEGRADED
		if !cryptoConnected && status != health.StatusDegraded {
			t.Errorf("expected DEGRADED when crypto not connected, got %s", status)
		}

		// If crypto is connected, overall should be HEALTHY
		if cryptoConnected && status != health.StatusHealthy {
			t.Errorf("expected HEALTHY when crypto connected, got %s", status)
		}
	})
}
