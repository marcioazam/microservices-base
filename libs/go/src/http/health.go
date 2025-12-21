package http

import (
	"encoding/json"
	"net/http"
	"sync"
)

// HealthStatus represents health check status.
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
)

// HealthCheck is a health check function.
type HealthCheck func() error

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	mu     sync.RWMutex
	checks map[string]HealthCheck
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		checks: make(map[string]HealthCheck),
	}
}

// Register registers a health check.
func (h *HealthHandler) Register(name string, check HealthCheck) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = check
}

// HealthResponse is the health check response.
type HealthResponse struct {
	Status HealthStatus      `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// LivenessHandler returns a simple liveness probe handler.
func (h *HealthHandler) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthResponse{Status: StatusHealthy})
	}
}

// ReadinessHandler returns a readiness probe handler.
func (h *HealthHandler) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		defer h.mu.RUnlock()

		response := HealthResponse{
			Status: StatusHealthy,
			Checks: make(map[string]string),
		}

		for name, check := range h.checks {
			if err := check(); err != nil {
				response.Status = StatusUnhealthy
				response.Checks[name] = err.Error()
			} else {
				response.Checks[name] = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if response.Status != StatusHealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(response)
	}
}
