package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the cache service.
type Metrics struct {
	CacheHitTotal         *prometheus.CounterVec
	CacheMissTotal        *prometheus.CounterVec
	CacheLatencySeconds   *prometheus.HistogramVec
	CacheMemoryUsageBytes prometheus.Gauge
	CacheEvictionTotal    prometheus.Counter
	CircuitBreakerState   *prometheus.GaugeVec
	RequestsTotal         *prometheus.CounterVec
	ErrorsTotal           *prometheus.CounterVec
	LogEntriesSent        prometheus.Counter
	LogEntriesDropped     prometheus.Counter
	LogBatchesSent        prometheus.Counter
	LogBatchesFailed      prometheus.Counter
}

// NewMetrics creates a new Metrics instance with all metrics registered.
func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		CacheHitTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hit_total",
				Help:      "Total number of cache hits",
			},
			[]string{"source", "namespace"},
		),
		CacheMissTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_miss_total",
				Help:      "Total number of cache misses",
			},
			[]string{"namespace"},
		),
		CacheLatencySeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "cache_latency_seconds",
				Help:      "Cache operation latency in seconds",
				Buckets:   []float64{.0001, .0005, .001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"operation", "source"},
		),
		CacheMemoryUsageBytes: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "cache_memory_usage_bytes",
				Help:      "Current memory usage of the local cache in bytes",
			},
		),
		CacheEvictionTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_eviction_total",
				Help:      "Total number of cache evictions",
			},
		),
		CircuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "circuit_breaker_state",
				Help:      "Current state of the circuit breaker (0=closed, 1=open, 2=half-open)",
			},
			[]string{"name"},
		),
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "requests_total",
				Help:      "Total number of requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors_total",
				Help:      "Total number of errors",
			},
			[]string{"type"},
		),
		LogEntriesSent: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "log_entries_sent_total",
				Help:      "Total number of log entries sent to logging-service",
			},
		),
		LogEntriesDropped: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "log_entries_dropped_total",
				Help:      "Total number of log entries dropped due to buffer overflow",
			},
		),
		LogBatchesSent: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "log_batches_sent_total",
				Help:      "Total number of log batches sent to logging-service",
			},
		),
		LogBatchesFailed: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "log_batches_failed_total",
				Help:      "Total number of log batches that failed to send",
			},
		),
	}
}

// RecordHit records a cache hit.
func (m *Metrics) RecordHit(source, namespace string) {
	m.CacheHitTotal.WithLabelValues(source, namespace).Inc()
}

// RecordMiss records a cache miss.
func (m *Metrics) RecordMiss(namespace string) {
	m.CacheMissTotal.WithLabelValues(namespace).Inc()
}

// RecordLatency records operation latency.
func (m *Metrics) RecordLatency(operation, source string, seconds float64) {
	m.CacheLatencySeconds.WithLabelValues(operation, source).Observe(seconds)
}

// SetMemoryUsage sets the current memory usage.
func (m *Metrics) SetMemoryUsage(bytes float64) {
	m.CacheMemoryUsageBytes.Set(bytes)
}

// RecordEviction records a cache eviction.
func (m *Metrics) RecordEviction() {
	m.CacheEvictionTotal.Inc()
}

// SetCircuitBreakerState sets the circuit breaker state.
func (m *Metrics) SetCircuitBreakerState(name string, state int) {
	m.CircuitBreakerState.WithLabelValues(name).Set(float64(state))
}

// RecordRequest records a request.
func (m *Metrics) RecordRequest(method, endpoint, status string) {
	m.RequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

// RecordError records an error.
func (m *Metrics) RecordError(errorType string) {
	m.ErrorsTotal.WithLabelValues(errorType).Inc()
}
