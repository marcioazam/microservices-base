package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents health check status.
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
)

// HealthCheck represents a single health check.
type HealthCheck struct {
	Name    string            `json:"name"`
	Status  HealthStatus      `json:"status"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
	Latency time.Duration     `json:"latency_ms"`
}

// HealthResponse represents the overall health response.
type HealthResponse struct {
	Status    HealthStatus  `json:"status"`
	Timestamp time.Time     `json:"timestamp"`
	Checks    []HealthCheck `json:"checks"`
}

// Checker performs a health check.
type Checker func(ctx context.Context) HealthCheck

// HealthChecker manages health checks.
type HealthChecker struct {
	checks  map[string]Checker
	mu      sync.RWMutex
	timeout time.Duration
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks:  make(map[string]Checker),
		timeout: time.Second * 5,
	}
}

// WithTimeout sets the check timeout.
func (h *HealthChecker) WithTimeout(d time.Duration) *HealthChecker {
	h.timeout = d
	return h
}

// Register adds a health check.
func (h *HealthChecker) Register(name string, checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = checker
}

// Unregister removes a health check.
func (h *HealthChecker) Unregister(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.checks, name)
}

// Check runs all health checks.
func (h *HealthChecker) Check(ctx context.Context) HealthResponse {
	h.mu.RLock()
	checks := make(map[string]Checker, len(h.checks))
	for k, v := range h.checks {
		checks[k] = v
	}
	h.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	var wg sync.WaitGroup
	results := make(chan HealthCheck, len(checks))

	for name, checker := range checks {
		wg.Add(1)
		go func(n string, c Checker) {
			defer wg.Done()
			start := time.Now()
			check := c(ctx)
			check.Name = n
			check.Latency = time.Since(start)
			results <- check
		}(name, checker)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var checkResults []HealthCheck
	overallStatus := StatusHealthy

	for check := range results {
		checkResults = append(checkResults, check)
		if check.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if check.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	return HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Checks:    checkResults,
	}
}

// Handler returns an HTTP handler for health checks.
func (h *HealthChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := h.Check(r.Context())

		w.Header().Set("Content-Type", "application/json")
		if response.Status == StatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else if response.Status == StatusDegraded {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		json.NewEncoder(w).Encode(response)
	}
}

// LivenessHandler returns a simple liveness probe handler.
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

// ReadinessHandler returns a readiness probe handler.
func (h *HealthChecker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := h.Check(r.Context())
		if response.Status == StatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(response)
	}
}

// HealthAggregator aggregates health checks from multiple sources.
type HealthAggregator struct {
	mu       sync.RWMutex
	checkers []Checker
	results  map[string]HealthCheck
	onChange func(HealthCheck)
}

// NewHealthAggregator creates a new health aggregator.
func NewHealthAggregator() *HealthAggregator {
	return &HealthAggregator{
		checkers: make([]Checker, 0),
		results:  make(map[string]HealthCheck),
	}
}

// RegisterChecker adds a health checker.
func (a *HealthAggregator) RegisterChecker(name string, checker Checker) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.checkers = append(a.checkers, checker)
}

// OnChange sets a callback for status changes.
func (a *HealthAggregator) OnChange(fn func(HealthCheck)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onChange = fn
}

// GetStatus returns the current aggregated status without running checks.
func (a *HealthAggregator) GetStatus() HealthStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, check := range a.results {
		if check.Status == StatusUnhealthy {
			return StatusUnhealthy
		}
	}
	return StatusHealthy
}

// NewHealthyCheck creates a healthy check result.
func NewHealthyCheck(name string) HealthCheck {
	return HealthCheck{Name: name, Status: StatusHealthy}
}

// NewDegradedCheck creates a degraded check result.
func NewDegradedCheck(name, message string) HealthCheck {
	return HealthCheck{Name: name, Status: StatusDegraded, Message: message}
}

// NewUnhealthyCheck creates an unhealthy check result.
func NewUnhealthyCheck(name, message string) HealthCheck {
	return HealthCheck{Name: name, Status: StatusUnhealthy, Message: message}
}
