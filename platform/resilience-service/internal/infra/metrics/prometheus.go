// Package metrics provides Prometheus metrics for resilience components.
package metrics

import (
	"sync"
	"time"
)

// Metrics holds all Prometheus metrics for the resilience service.
type Metrics struct {
	mu sync.RWMutex

	// Circuit Breaker metrics
	circuitBreakerState       map[string]string // service -> state
	circuitBreakerTransitions map[string]int64  // service:from:to -> count
	circuitBreakerFailures    map[string]int64  // service -> count
	circuitBreakerSuccesses   map[string]int64  // service -> count

	// Retry metrics
	retryAttempts  map[string]int64         // service -> total attempts
	retrySuccesses map[string]int64         // service -> successful retries
	retryExhausted map[string]int64         // service -> exhausted retries
	retryDelaySum  map[string]time.Duration // service -> total delay

	// Rate Limiter metrics
	rateLimitAllowed  map[string]int64 // key -> allowed count
	rateLimitRejected map[string]int64 // key -> rejected count
	rateLimitTokens   map[string]int64 // key -> current tokens

	// Bulkhead metrics
	bulkheadActive   map[string]int64 // partition -> active count
	bulkheadQueued   map[string]int64 // partition -> queued count
	bulkheadRejected map[string]int64 // partition -> rejected count
	bulkheadMax      map[string]int64 // partition -> max concurrent

	// Timeout metrics
	timeoutTotal     map[string]int64         // operation -> total count
	timeoutExceeded  map[string]int64         // operation -> exceeded count
	timeoutDurations map[string]time.Duration // operation -> total duration

	// Health metrics
	healthStatus map[string]string // service -> status

	// Latency histograms for p50/p95/p99
	circuitBreakerLatency *LatencyHistograms // service -> latency histogram
	retryLatency          *LatencyHistograms // service -> latency histogram
	rateLimitLatency      *LatencyHistograms // key -> latency histogram
	bulkheadLatency       *LatencyHistograms // partition -> latency histogram
	operationLatency      *LatencyHistograms // operation -> latency histogram
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		circuitBreakerState:       make(map[string]string),
		circuitBreakerTransitions: make(map[string]int64),
		circuitBreakerFailures:    make(map[string]int64),
		circuitBreakerSuccesses:   make(map[string]int64),
		retryAttempts:             make(map[string]int64),
		retrySuccesses:            make(map[string]int64),
		retryExhausted:            make(map[string]int64),
		retryDelaySum:             make(map[string]time.Duration),
		rateLimitAllowed:          make(map[string]int64),
		rateLimitRejected:         make(map[string]int64),
		rateLimitTokens:           make(map[string]int64),
		bulkheadActive:            make(map[string]int64),
		bulkheadQueued:            make(map[string]int64),
		bulkheadRejected:          make(map[string]int64),
		bulkheadMax:               make(map[string]int64),
		timeoutTotal:              make(map[string]int64),
		timeoutExceeded:           make(map[string]int64),
		timeoutDurations:          make(map[string]time.Duration),
		healthStatus:              make(map[string]string),
		circuitBreakerLatency:     NewLatencyHistograms(),
		retryLatency:              NewLatencyHistograms(),
		rateLimitLatency:          NewLatencyHistograms(),
		bulkheadLatency:           NewLatencyHistograms(),
		operationLatency:          NewLatencyHistograms(),
	}
}

// Circuit Breaker metrics

// SetCircuitBreakerState sets the current state of a circuit breaker.
func (m *Metrics) SetCircuitBreakerState(service, state string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.circuitBreakerState[service] = state
}

// RecordCircuitBreakerTransition records a state transition.
func (m *Metrics) RecordCircuitBreakerTransition(service, from, to string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := service + ":" + from + ":" + to
	m.circuitBreakerTransitions[key]++
}

// RecordCircuitBreakerFailure records a failure.
func (m *Metrics) RecordCircuitBreakerFailure(service string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.circuitBreakerFailures[service]++
}

// RecordCircuitBreakerSuccess records a success.
func (m *Metrics) RecordCircuitBreakerSuccess(service string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.circuitBreakerSuccesses[service]++
}

// Retry metrics

// RecordRetryAttempt records a retry attempt.
func (m *Metrics) RecordRetryAttempt(service string, delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retryAttempts[service]++
	m.retryDelaySum[service] += delay
}

// RecordRetrySuccess records a successful retry.
func (m *Metrics) RecordRetrySuccess(service string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retrySuccesses[service]++
}

// RecordRetryExhausted records an exhausted retry.
func (m *Metrics) RecordRetryExhausted(service string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retryExhausted[service]++
}

// Rate Limiter metrics

// RecordRateLimitAllowed records an allowed request.
func (m *Metrics) RecordRateLimitAllowed(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rateLimitAllowed[key]++
}

// RecordRateLimitRejected records a rejected request.
func (m *Metrics) RecordRateLimitRejected(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rateLimitRejected[key]++
}

// SetRateLimitTokens sets the current token count.
func (m *Metrics) SetRateLimitTokens(key string, tokens int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rateLimitTokens[key] = tokens
}

// Bulkhead metrics

// SetBulkheadActive sets the active count for a partition.
func (m *Metrics) SetBulkheadActive(partition string, count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bulkheadActive[partition] = count
}

// SetBulkheadQueued sets the queued count for a partition.
func (m *Metrics) SetBulkheadQueued(partition string, count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bulkheadQueued[partition] = count
}

// RecordBulkheadRejected records a rejected request.
func (m *Metrics) RecordBulkheadRejected(partition string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bulkheadRejected[partition]++
}

// SetBulkheadMax sets the max concurrent for a partition.
func (m *Metrics) SetBulkheadMax(partition string, max int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bulkheadMax[partition] = max
}

// Timeout metrics

// RecordTimeout records a timeout operation.
func (m *Metrics) RecordTimeout(operation string, duration time.Duration, exceeded bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeoutTotal[operation]++
	m.timeoutDurations[operation] += duration
	if exceeded {
		m.timeoutExceeded[operation]++
	}
}

// Health metrics

// SetHealthStatus sets the health status for a service.
func (m *Metrics) SetHealthStatus(service, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthStatus[service] = status
}

// Latency histogram methods

// ObserveCircuitBreakerLatency records circuit breaker operation latency.
func (m *Metrics) ObserveCircuitBreakerLatency(service string, duration time.Duration) {
	m.circuitBreakerLatency.Observe(service, duration)
}

// ObserveRetryLatency records retry operation latency.
func (m *Metrics) ObserveRetryLatency(service string, duration time.Duration) {
	m.retryLatency.Observe(service, duration)
}

// ObserveRateLimitLatency records rate limit check latency.
func (m *Metrics) ObserveRateLimitLatency(key string, duration time.Duration) {
	m.rateLimitLatency.Observe(key, duration)
}

// ObserveBulkheadLatency records bulkhead wait latency.
func (m *Metrics) ObserveBulkheadLatency(partition string, duration time.Duration) {
	m.bulkheadLatency.Observe(partition, duration)
}

// ObserveOperationLatency records general operation latency.
func (m *Metrics) ObserveOperationLatency(operation string, duration time.Duration) {
	m.operationLatency.Observe(operation, duration)
}

// GetCircuitBreakerPercentiles returns p50, p95, p99 for circuit breaker.
func (m *Metrics) GetCircuitBreakerPercentiles(service string) (p50, p95, p99 float64) {
	return m.circuitBreakerLatency.GetPercentiles(service)
}

// GetRetryPercentiles returns p50, p95, p99 for retry operations.
func (m *Metrics) GetRetryPercentiles(service string) (p50, p95, p99 float64) {
	return m.retryLatency.GetPercentiles(service)
}

// GetRateLimitPercentiles returns p50, p95, p99 for rate limit checks.
func (m *Metrics) GetRateLimitPercentiles(key string) (p50, p95, p99 float64) {
	return m.rateLimitLatency.GetPercentiles(key)
}

// GetBulkheadPercentiles returns p50, p95, p99 for bulkhead waits.
func (m *Metrics) GetBulkheadPercentiles(partition string) (p50, p95, p99 float64) {
	return m.bulkheadLatency.GetPercentiles(partition)
}

// GetOperationPercentiles returns p50, p95, p99 for operations.
func (m *Metrics) GetOperationPercentiles(operation string) (p50, p95, p99 float64) {
	return m.operationLatency.GetPercentiles(operation)
}

// Snapshot returns a copy of all metrics for export.
type MetricsSnapshot struct {
	CircuitBreakerState       map[string]string
	CircuitBreakerTransitions map[string]int64
	CircuitBreakerFailures    map[string]int64
	CircuitBreakerSuccesses   map[string]int64
	RetryAttempts             map[string]int64
	RetrySuccesses            map[string]int64
	RetryExhausted            map[string]int64
	RetryDelaySum             map[string]time.Duration
	RateLimitAllowed          map[string]int64
	RateLimitRejected         map[string]int64
	RateLimitTokens           map[string]int64
	BulkheadActive            map[string]int64
	BulkheadQueued            map[string]int64
	BulkheadRejected          map[string]int64
	BulkheadMax               map[string]int64
	TimeoutTotal              map[string]int64
	TimeoutExceeded           map[string]int64
	TimeoutDurations          map[string]time.Duration
	HealthStatus              map[string]string

	// Latency histograms
	CircuitBreakerLatency map[string]HistogramData
	RetryLatency          map[string]HistogramData
	RateLimitLatency      map[string]HistogramData
	BulkheadLatency       map[string]HistogramData
	OperationLatency      map[string]HistogramData
}

// Snapshot returns a copy of all metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MetricsSnapshot{
		CircuitBreakerState:       copyStringMap(m.circuitBreakerState),
		CircuitBreakerTransitions: copyInt64Map(m.circuitBreakerTransitions),
		CircuitBreakerFailures:    copyInt64Map(m.circuitBreakerFailures),
		CircuitBreakerSuccesses:   copyInt64Map(m.circuitBreakerSuccesses),
		RetryAttempts:             copyInt64Map(m.retryAttempts),
		RetrySuccesses:            copyInt64Map(m.retrySuccesses),
		RetryExhausted:            copyInt64Map(m.retryExhausted),
		RetryDelaySum:             copyDurationMap(m.retryDelaySum),
		RateLimitAllowed:          copyInt64Map(m.rateLimitAllowed),
		RateLimitRejected:         copyInt64Map(m.rateLimitRejected),
		RateLimitTokens:           copyInt64Map(m.rateLimitTokens),
		BulkheadActive:            copyInt64Map(m.bulkheadActive),
		BulkheadQueued:            copyInt64Map(m.bulkheadQueued),
		BulkheadRejected:          copyInt64Map(m.bulkheadRejected),
		BulkheadMax:               copyInt64Map(m.bulkheadMax),
		TimeoutTotal:              copyInt64Map(m.timeoutTotal),
		TimeoutExceeded:           copyInt64Map(m.timeoutExceeded),
		TimeoutDurations:          copyDurationMap(m.timeoutDurations),
		HealthStatus:              copyStringMap(m.healthStatus),
		CircuitBreakerLatency:     m.circuitBreakerLatency.GetAllData(),
		RetryLatency:              m.retryLatency.GetAllData(),
		RateLimitLatency:          m.rateLimitLatency.GetAllData(),
		BulkheadLatency:           m.bulkheadLatency.GetAllData(),
		OperationLatency:          m.operationLatency.GetAllData(),
	}
}

func copyStringMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func copyInt64Map(m map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func copyDurationMap(m map[string]time.Duration) map[string]time.Duration {
	result := make(map[string]time.Duration, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
