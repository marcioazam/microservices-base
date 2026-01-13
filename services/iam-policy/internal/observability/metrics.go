// Package observability provides tracing and metrics for IAM Policy Service.
package observability

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds service metrics.
type Metrics struct {
	mu sync.RWMutex

	// Authorization metrics
	authDecisions     atomic.Int64
	authAllowed       atomic.Int64
	authDenied        atomic.Int64
	authErrors        atomic.Int64
	authDurationSum   atomic.Int64
	authDurationCount atomic.Int64

	// Cache metrics
	cacheHits   atomic.Int64
	cacheMisses atomic.Int64
	cacheErrors atomic.Int64

	// Policy metrics
	policyEvaluations atomic.Int64
	policyReloads     atomic.Int64
	policyCount       atomic.Int64

	// gRPC metrics
	grpcRequests      map[string]*atomic.Int64
	grpcErrors        map[string]*atomic.Int64
	grpcDurationSum   map[string]*atomic.Int64
	grpcDurationCount map[string]*atomic.Int64
}

// NewMetrics creates a new metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		grpcRequests:      make(map[string]*atomic.Int64),
		grpcErrors:        make(map[string]*atomic.Int64),
		grpcDurationSum:   make(map[string]*atomic.Int64),
		grpcDurationCount: make(map[string]*atomic.Int64),
	}
}

// RecordAuthDecision records an authorization decision.
func (m *Metrics) RecordAuthDecision(allowed bool, duration time.Duration) {
	m.authDecisions.Add(1)
	if allowed {
		m.authAllowed.Add(1)
	} else {
		m.authDenied.Add(1)
	}
	m.authDurationSum.Add(duration.Nanoseconds())
	m.authDurationCount.Add(1)
}

// RecordAuthError records an authorization error.
func (m *Metrics) RecordAuthError() {
	m.authErrors.Add(1)
}

// RecordCacheHit records a cache hit.
func (m *Metrics) RecordCacheHit() {
	m.cacheHits.Add(1)
}

// RecordCacheMiss records a cache miss.
func (m *Metrics) RecordCacheMiss() {
	m.cacheMisses.Add(1)
}

// RecordCacheError records a cache error.
func (m *Metrics) RecordCacheError() {
	m.cacheErrors.Add(1)
}

// RecordPolicyEvaluation records a policy evaluation.
func (m *Metrics) RecordPolicyEvaluation() {
	m.policyEvaluations.Add(1)
}

// RecordPolicyReload records a policy reload.
func (m *Metrics) RecordPolicyReload(count int) {
	m.policyReloads.Add(1)
	m.policyCount.Store(int64(count))
}

// RecordGRPCRequest records a gRPC request.
func (m *Metrics) RecordGRPCRequest(method string, duration time.Duration, err error) {
	m.mu.Lock()
	if m.grpcRequests[method] == nil {
		m.grpcRequests[method] = &atomic.Int64{}
		m.grpcErrors[method] = &atomic.Int64{}
		m.grpcDurationSum[method] = &atomic.Int64{}
		m.grpcDurationCount[method] = &atomic.Int64{}
	}
	m.mu.Unlock()

	m.grpcRequests[method].Add(1)
	m.grpcDurationSum[method].Add(duration.Nanoseconds())
	m.grpcDurationCount[method].Add(1)

	if err != nil {
		m.grpcErrors[method].Add(1)
	}
}

// GetSnapshot returns a snapshot of current metrics.
func (m *Metrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	grpcMethods := make(map[string]GRPCMethodMetrics)
	for method := range m.grpcRequests {
		count := m.grpcDurationCount[method].Load()
		avgDuration := time.Duration(0)
		if count > 0 {
			avgDuration = time.Duration(m.grpcDurationSum[method].Load() / count)
		}

		grpcMethods[method] = GRPCMethodMetrics{
			Requests:    m.grpcRequests[method].Load(),
			Errors:      m.grpcErrors[method].Load(),
			AvgDuration: avgDuration,
		}
	}

	authCount := m.authDurationCount.Load()
	avgAuthDuration := time.Duration(0)
	if authCount > 0 {
		avgAuthDuration = time.Duration(m.authDurationSum.Load() / authCount)
	}

	return MetricsSnapshot{
		AuthDecisions:   m.authDecisions.Load(),
		AuthAllowed:     m.authAllowed.Load(),
		AuthDenied:      m.authDenied.Load(),
		AuthErrors:      m.authErrors.Load(),
		AvgAuthDuration: avgAuthDuration,
		CacheHits:       m.cacheHits.Load(),
		CacheMisses:     m.cacheMisses.Load(),
		CacheErrors:     m.cacheErrors.Load(),
		PolicyEvals:     m.policyEvaluations.Load(),
		PolicyReloads:   m.policyReloads.Load(),
		PolicyCount:     m.policyCount.Load(),
		GRPCMethods:     grpcMethods,
	}
}

// MetricsSnapshot holds a point-in-time snapshot of metrics.
type MetricsSnapshot struct {
	AuthDecisions   int64
	AuthAllowed     int64
	AuthDenied      int64
	AuthErrors      int64
	AvgAuthDuration time.Duration
	CacheHits       int64
	CacheMisses     int64
	CacheErrors     int64
	PolicyEvals     int64
	PolicyReloads   int64
	PolicyCount     int64
	GRPCMethods     map[string]GRPCMethodMetrics
}

// GRPCMethodMetrics holds metrics for a gRPC method.
type GRPCMethodMetrics struct {
	Requests    int64
	Errors      int64
	AvgDuration time.Duration
}

// Handler returns an HTTP handler for metrics endpoint.
func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot := m.GetSnapshot()

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		// Prometheus format
		fmt := "# HELP %s %s\n# TYPE %s %s\n%s %d\n"

		w.Write([]byte(formatMetric(fmt, "iam_auth_decisions_total", "Total authorization decisions", "counter", snapshot.AuthDecisions)))
		w.Write([]byte(formatMetric(fmt, "iam_auth_allowed_total", "Total allowed decisions", "counter", snapshot.AuthAllowed)))
		w.Write([]byte(formatMetric(fmt, "iam_auth_denied_total", "Total denied decisions", "counter", snapshot.AuthDenied)))
		w.Write([]byte(formatMetric(fmt, "iam_auth_errors_total", "Total authorization errors", "counter", snapshot.AuthErrors)))
		w.Write([]byte(formatMetric(fmt, "iam_cache_hits_total", "Total cache hits", "counter", snapshot.CacheHits)))
		w.Write([]byte(formatMetric(fmt, "iam_cache_misses_total", "Total cache misses", "counter", snapshot.CacheMisses)))
		w.Write([]byte(formatMetric(fmt, "iam_policy_evaluations_total", "Total policy evaluations", "counter", snapshot.PolicyEvals)))
		w.Write([]byte(formatMetric(fmt, "iam_policy_count", "Current policy count", "gauge", snapshot.PolicyCount)))
	}
}

func formatMetric(format, name, help, metricType string, value int64) string {
	return fmt.Sprintf(format, name, help, name, metricType, name, value)
}
