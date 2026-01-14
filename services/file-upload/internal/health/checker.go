// Package health provides health check functionality.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents health check status.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckResult represents a single health check result.
type CheckResult struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status  Status        `json:"status"`
	Checks  []CheckResult `json:"checks,omitempty"`
	Version string        `json:"version,omitempty"`
}

// Checker defines a health check function.
type Checker func(ctx context.Context) CheckResult

// HealthChecker manages health checks.
type HealthChecker struct {
	checkers map[string]Checker
	version  string
	mu       sync.RWMutex
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(version string) *HealthChecker {
	return &HealthChecker{
		checkers: make(map[string]Checker),
		version:  version,
	}
}

// Register registers a health checker.
func (h *HealthChecker) Register(name string, checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[name] = checker
}

// CheckLiveness performs liveness check.
func (h *HealthChecker) CheckLiveness() HealthResponse {
	return HealthResponse{
		Status:  StatusHealthy,
		Version: h.version,
	}
}

// CheckReadiness performs readiness check.
func (h *HealthChecker) CheckReadiness(ctx context.Context) HealthResponse {
	h.mu.RLock()
	checkers := make(map[string]Checker, len(h.checkers))
	for k, v := range h.checkers {
		checkers[k] = v
	}
	h.mu.RUnlock()

	results := make([]CheckResult, 0, len(checkers))
	overallStatus := StatusHealthy

	for name, checker := range checkers {
		start := time.Now()
		result := checker(ctx)
		result.Name = name
		result.Latency = time.Since(start).String()
		results = append(results, result)

		if result.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if result.Status == StatusDegraded && overallStatus != StatusUnhealthy {
			overallStatus = StatusDegraded
		}
	}

	return HealthResponse{
		Status:  overallStatus,
		Checks:  results,
		Version: h.version,
	}
}

// LivenessHandler returns HTTP handler for liveness endpoint.
func (h *HealthChecker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := h.CheckLiveness()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// ReadinessHandler returns HTTP handler for readiness endpoint.
func (h *HealthChecker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		response := h.CheckReadiness(ctx)

		w.Header().Set("Content-Type", "application/json")
		switch response.Status {
		case StatusHealthy:
			w.WriteHeader(http.StatusOK)
		case StatusDegraded:
			w.WriteHeader(http.StatusOK) // Still serving, but degraded
		case StatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(response)
	}
}

// DatabaseChecker creates a database health checker.
func DatabaseChecker(pingFunc func(ctx context.Context) error) Checker {
	return func(ctx context.Context) CheckResult {
		if err := pingFunc(ctx); err != nil {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "database unavailable: " + err.Error(),
			}
		}
		return CheckResult{Status: StatusHealthy, Message: "database connected"}
	}
}

// CacheChecker creates a cache health checker.
func CacheChecker(pingFunc func(ctx context.Context) error) Checker {
	return func(ctx context.Context) CheckResult {
		if err := pingFunc(ctx); err != nil {
			return CheckResult{
				Status:  StatusDegraded,
				Message: "cache unavailable: " + err.Error(),
			}
		}
		return CheckResult{Status: StatusHealthy, Message: "cache connected"}
	}
}

// StorageChecker creates a storage health checker.
func StorageChecker(checkFunc func(ctx context.Context) error) Checker {
	return func(ctx context.Context) CheckResult {
		if err := checkFunc(ctx); err != nil {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "storage unavailable: " + err.Error(),
			}
		}
		return CheckResult{Status: StatusHealthy, Message: "storage accessible"}
	}
}
