// Package observability provides metrics recording using shared types from libs/go.
package observability

import (
	"context"
	"sync"

	"github.com/authcorp/libs/go/src/fault"
	"github.com/auth-platform/platform/resilience-service/internal/domain/interfaces"
)

// MetricsRecorder records resilience execution metrics.
type MetricsRecorder struct {
	executions     int64
	successes      int64
	failures       int64
	retries        int64
	circuitOpens   int64
	rateLimits     int64
	bulkheadQueues int64
	cacheHits      int64
	cacheMisses    int64
	cacheEvictions int64
	mu             sync.RWMutex
}

// NewMetricsRecorder creates a new metrics recorder.
func NewMetricsRecorder() *MetricsRecorder {
	return &MetricsRecorder{}
}

// RecordExecution records metrics for a completed execution.
func (r *MetricsRecorder) RecordExecution(ctx context.Context, metrics fault.ExecutionMetrics) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.executions++
	if metrics.Success {
		r.successes++
	} else {
		r.failures++
	}

	if metrics.RetryAttempts > 0 {
		r.retries += int64(metrics.RetryAttempts)
	}

	if metrics.RateLimited {
		r.rateLimits++
	}

	if metrics.BulkheadQueued {
		r.bulkheadQueues++
	}
}

// RecordCircuitState records a circuit breaker state change.
func (r *MetricsRecorder) RecordCircuitState(ctx context.Context, policyName string, state string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if state == "open" || state == "OPEN" {
		r.circuitOpens++
	}
}

// RecordRetryAttempt records a retry attempt.
func (r *MetricsRecorder) RecordRetryAttempt(ctx context.Context, policyName string, attempt int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.retries++
}

// RecordRateLimit records a rate limit event.
func (r *MetricsRecorder) RecordRateLimit(ctx context.Context, policyName string, limited bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if limited {
		r.rateLimits++
	}
}

// RecordBulkheadQueue records a bulkhead queue event.
func (r *MetricsRecorder) RecordBulkheadQueue(ctx context.Context, policyName string, queued bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if queued {
		r.bulkheadQueues++
	}
}

// RecordCacheStats records cache statistics.
func (r *MetricsRecorder) RecordCacheStats(ctx context.Context, hits, misses, evictions int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cacheHits = hits
	r.cacheMisses = misses
	r.cacheEvictions = evictions
}

// Stats returns current metrics statistics.
type Stats struct {
	Executions     int64   `json:"executions"`
	Successes      int64   `json:"successes"`
	Failures       int64   `json:"failures"`
	SuccessRate    float64 `json:"success_rate"`
	Retries        int64   `json:"retries"`
	CircuitOpens   int64   `json:"circuit_opens"`
	RateLimits     int64   `json:"rate_limits"`
	BulkheadQueues int64   `json:"bulkhead_queues"`
	CacheHits      int64   `json:"cache_hits"`
	CacheMisses    int64   `json:"cache_misses"`
	CacheEvictions int64   `json:"cache_evictions"`
	CacheHitRate   float64 `json:"cache_hit_rate"`
}

// GetStats returns current metrics statistics.
func (r *MetricsRecorder) GetStats() Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := Stats{
		Executions:     r.executions,
		Successes:      r.successes,
		Failures:       r.failures,
		Retries:        r.retries,
		CircuitOpens:   r.circuitOpens,
		RateLimits:     r.rateLimits,
		BulkheadQueues: r.bulkheadQueues,
		CacheHits:      r.cacheHits,
		CacheMisses:    r.cacheMisses,
		CacheEvictions: r.cacheEvictions,
	}

	if r.executions > 0 {
		stats.SuccessRate = float64(r.successes) / float64(r.executions)
	}

	totalCacheOps := r.cacheHits + r.cacheMisses
	if totalCacheOps > 0 {
		stats.CacheHitRate = float64(r.cacheHits) / float64(totalCacheOps)
	}

	return stats
}

// Reset resets all metrics to zero.
func (r *MetricsRecorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.executions = 0
	r.successes = 0
	r.failures = 0
	r.retries = 0
	r.circuitOpens = 0
	r.rateLimits = 0
	r.bulkheadQueues = 0
	r.cacheHits = 0
	r.cacheMisses = 0
	r.cacheEvictions = 0
}

// Ensure MetricsRecorder implements interfaces.MetricsRecorder.
var _ interfaces.MetricsRecorder = (*MetricsRecorder)(nil)
