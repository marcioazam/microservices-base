// Package health provides health check management for IAM Policy Service.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Status represents health status.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckResult represents the result of a health check.
type CheckResult struct {
	Status    Status        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration_ms"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthCheck is a function that performs a health check.
type HealthCheck func(ctx context.Context) CheckResult

// Manager manages health checks.
type Manager struct {
	mu           sync.RWMutex
	checks       map[string]HealthCheck
	results      map[string]CheckResult
	shuttingDown atomic.Bool
}

// NewManager creates a new health manager.
func NewManager() *Manager {
	return &Manager{
		checks:  make(map[string]HealthCheck),
		results: make(map[string]CheckResult),
	}
}

// RegisterCheck registers a health check.
func (m *Manager) RegisterCheck(name string, check HealthCheck) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checks[name] = check
}

// SetShuttingDown marks the service as shutting down.
func (m *Manager) SetShuttingDown() {
	m.shuttingDown.Store(true)
}

// IsShuttingDown returns whether the service is shutting down.
func (m *Manager) IsShuttingDown() bool {
	return m.shuttingDown.Load()
}

// RunChecks runs all registered health checks.
func (m *Manager) RunChecks(ctx context.Context) map[string]CheckResult {
	m.mu.RLock()
	checks := make(map[string]HealthCheck, len(m.checks))
	for name, check := range m.checks {
		checks[name] = check
	}
	m.mu.RUnlock()

	results := make(map[string]CheckResult, len(checks))
	var wg sync.WaitGroup

	for name, check := range checks {
		wg.Add(1)
		go func(name string, check HealthCheck) {
			defer wg.Done()
			start := time.Now()
			result := check(ctx)
			result.Duration = time.Since(start)
			result.Timestamp = time.Now()

			m.mu.Lock()
			m.results[name] = result
			results[name] = result
			m.mu.Unlock()
		}(name, check)
	}

	wg.Wait()
	return results
}

// GetOverallStatus returns the overall health status.
func (m *Manager) GetOverallStatus(ctx context.Context) Status {
	if m.shuttingDown.Load() {
		return StatusUnhealthy
	}

	results := m.RunChecks(ctx)

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range results {
		switch result.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}
	return StatusHealthy
}

// LivenessHandler returns an HTTP handler for liveness checks.
func (m *Manager) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := map[string]interface{}{
			"status":    "alive",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// ReadinessHandler returns an HTTP handler for readiness checks.
func (m *Manager) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		status := m.GetOverallStatus(ctx)

		m.mu.RLock()
		results := make(map[string]CheckResult, len(m.results))
		for k, v := range m.results {
			results[k] = v
		}
		m.mu.RUnlock()

		response := map[string]interface{}{
			"status":    status,
			"checks":    results,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		if status == StatusHealthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(response)
	}
}

// CacheHealthCheck creates a health check for cache connectivity.
func CacheHealthCheck(pingFunc func(context.Context) error) HealthCheck {
	return func(ctx context.Context) CheckResult {
		if err := pingFunc(ctx); err != nil {
			return CheckResult{Status: StatusDegraded, Message: err.Error()}
		}
		return CheckResult{Status: StatusHealthy}
	}
}

// LoggingHealthCheck creates a health check for logging connectivity.
func LoggingHealthCheck(pingFunc func(context.Context) error) HealthCheck {
	return func(ctx context.Context) CheckResult {
		if err := pingFunc(ctx); err != nil {
			return CheckResult{Status: StatusDegraded, Message: err.Error()}
		}
		return CheckResult{Status: StatusHealthy}
	}
}

// CryptoHealthStatus holds crypto service health information.
type CryptoHealthStatus struct {
	Connected  bool  `json:"crypto_service_connected"`
	LatencyMs  int64 `json:"crypto_service_latency_ms"`
	HSMConnected bool `json:"hsm_connected,omitempty"`
	KMSConnected bool `json:"kms_connected,omitempty"`
}

// CryptoHealthChecker is a function that checks crypto service health.
type CryptoHealthChecker func(ctx context.Context) (*CryptoHealthStatus, error)

// CryptoHealthCheck creates a health check for crypto service connectivity.
// Returns DEGRADED (not UNHEALTHY) when crypto service is unavailable.
func CryptoHealthCheck(checker CryptoHealthChecker) HealthCheck {
	return func(ctx context.Context) CheckResult {
		status, err := checker(ctx)
		if err != nil {
			return CheckResult{
				Status:  StatusDegraded,
				Message: "crypto service check failed: " + err.Error(),
			}
		}

		if status == nil || !status.Connected {
			return CheckResult{
				Status:  StatusDegraded,
				Message: "crypto service not connected",
			}
		}

		return CheckResult{
			Status:  StatusHealthy,
			Message: "crypto service connected",
		}
	}
}

// ExtendedReadinessResponse includes crypto service status.
type ExtendedReadinessResponse struct {
	Status                 Status                   `json:"status"`
	Checks                 map[string]CheckResult   `json:"checks"`
	CryptoServiceConnected bool                     `json:"crypto_service_connected"`
	CryptoServiceLatencyMs int64                    `json:"crypto_service_latency_ms"`
	Timestamp              string                   `json:"timestamp"`
}

// SetCryptoHealthCheck registers a crypto service health check.
func (m *Manager) SetCryptoHealthCheck(checker func(ctx context.Context) *CryptoHealthStatus) {
	m.RegisterCheck("crypto", func(ctx context.Context) CheckResult {
		status := checker(ctx)
		if status == nil || !status.Connected {
			return CheckResult{
				Status:  StatusDegraded,
				Message: "crypto service not connected",
			}
		}
		return CheckResult{
			Status:  StatusHealthy,
			Message: "crypto service connected",
		}
	})
}
