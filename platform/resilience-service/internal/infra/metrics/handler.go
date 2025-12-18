// Package metrics provides Prometheus metrics for resilience components.
package metrics

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Handler returns an HTTP handler for Prometheus metrics.
func Handler(m *Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		snapshot := m.Snapshot()
		writeMetrics(w, snapshot)
	}
}

func writeMetrics(w io.Writer, s MetricsSnapshot) {
	// Circuit Breaker metrics
	writeHelp(w, "resilience_circuit_breaker_state", "Current state of circuit breaker (0=closed, 1=open, 2=half_open)")
	for service, state := range s.CircuitBreakerState {
		value := stateToValue(state)
		writeMetric(w, "resilience_circuit_breaker_state", value, "service", service, "state", state)
	}

	writeHelp(w, "resilience_circuit_breaker_transitions_total", "Total circuit breaker state transitions")
	for key, count := range s.CircuitBreakerTransitions {
		parts := strings.SplitN(key, ":", 3)
		if len(parts) == 3 {
			writeMetric(w, "resilience_circuit_breaker_transitions_total", float64(count),
				"service", parts[0], "from", parts[1], "to", parts[2])
		}
	}

	writeHelp(w, "resilience_circuit_breaker_failures_total", "Total circuit breaker failures")
	for service, count := range s.CircuitBreakerFailures {
		writeMetric(w, "resilience_circuit_breaker_failures_total", float64(count), "service", service)
	}

	writeHelp(w, "resilience_circuit_breaker_successes_total", "Total circuit breaker successes")
	for service, count := range s.CircuitBreakerSuccesses {
		writeMetric(w, "resilience_circuit_breaker_successes_total", float64(count), "service", service)
	}

	// Retry metrics
	writeHelp(w, "resilience_retry_attempts_total", "Total retry attempts")
	for service, count := range s.RetryAttempts {
		writeMetric(w, "resilience_retry_attempts_total", float64(count), "service", service)
	}

	writeHelp(w, "resilience_retry_successes_total", "Total successful retries")
	for service, count := range s.RetrySuccesses {
		writeMetric(w, "resilience_retry_successes_total", float64(count), "service", service)
	}

	writeHelp(w, "resilience_retry_exhausted_total", "Total exhausted retries")
	for service, count := range s.RetryExhausted {
		writeMetric(w, "resilience_retry_exhausted_total", float64(count), "service", service)
	}

	writeHelp(w, "resilience_retry_delay_seconds_total", "Total retry delay in seconds")
	for service, duration := range s.RetryDelaySum {
		writeMetric(w, "resilience_retry_delay_seconds_total", duration.Seconds(), "service", service)
	}

	// Rate Limiter metrics
	writeHelp(w, "resilience_rate_limit_allowed_total", "Total allowed requests")
	for key, count := range s.RateLimitAllowed {
		writeMetric(w, "resilience_rate_limit_allowed_total", float64(count), "key", key)
	}

	writeHelp(w, "resilience_rate_limit_rejected_total", "Total rejected requests")
	for key, count := range s.RateLimitRejected {
		writeMetric(w, "resilience_rate_limit_rejected_total", float64(count), "key", key)
	}

	writeHelp(w, "resilience_rate_limit_tokens", "Current token count")
	for key, tokens := range s.RateLimitTokens {
		writeMetric(w, "resilience_rate_limit_tokens", float64(tokens), "key", key)
	}

	// Bulkhead metrics
	writeHelp(w, "resilience_bulkhead_active", "Current active requests")
	for partition, count := range s.BulkheadActive {
		writeMetric(w, "resilience_bulkhead_active", float64(count), "partition", partition)
	}

	writeHelp(w, "resilience_bulkhead_queued", "Current queued requests")
	for partition, count := range s.BulkheadQueued {
		writeMetric(w, "resilience_bulkhead_queued", float64(count), "partition", partition)
	}

	writeHelp(w, "resilience_bulkhead_rejected_total", "Total rejected requests")
	for partition, count := range s.BulkheadRejected {
		writeMetric(w, "resilience_bulkhead_rejected_total", float64(count), "partition", partition)
	}

	writeHelp(w, "resilience_bulkhead_max", "Maximum concurrent requests")
	for partition, max := range s.BulkheadMax {
		writeMetric(w, "resilience_bulkhead_max", float64(max), "partition", partition)
	}

	// Timeout metrics
	writeHelp(w, "resilience_timeout_total", "Total timeout operations")
	for operation, count := range s.TimeoutTotal {
		writeMetric(w, "resilience_timeout_total", float64(count), "operation", operation)
	}

	writeHelp(w, "resilience_timeout_exceeded_total", "Total exceeded timeouts")
	for operation, count := range s.TimeoutExceeded {
		writeMetric(w, "resilience_timeout_exceeded_total", float64(count), "operation", operation)
	}

	writeHelp(w, "resilience_timeout_duration_seconds_total", "Total timeout duration in seconds")
	for operation, duration := range s.TimeoutDurations {
		writeMetric(w, "resilience_timeout_duration_seconds_total", duration.Seconds(), "operation", operation)
	}

	// Health metrics
	writeHelp(w, "resilience_health_status", "Health status (0=unhealthy, 1=degraded, 2=healthy)")
	for service, status := range s.HealthStatus {
		value := healthToValue(status)
		writeMetric(w, "resilience_health_status", value, "service", service, "status", status)
	}

	// Latency histograms
	writeHistograms(w, "resilience_circuit_breaker_latency_seconds", "Circuit breaker operation latency", s.CircuitBreakerLatency, "service")
	writeHistograms(w, "resilience_retry_latency_seconds", "Retry operation latency", s.RetryLatency, "service")
	writeHistograms(w, "resilience_rate_limit_latency_seconds", "Rate limit check latency", s.RateLimitLatency, "key")
	writeHistograms(w, "resilience_bulkhead_latency_seconds", "Bulkhead wait latency", s.BulkheadLatency, "partition")
	writeHistograms(w, "resilience_operation_latency_seconds", "Operation latency", s.OperationLatency, "operation")
}

func writeHelp(w io.Writer, name, help string) {
	fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	fmt.Fprintf(w, "# TYPE %s gauge\n", name)
}

func writeMetric(w io.Writer, name string, value float64, labels ...string) {
	if len(labels) == 0 {
		fmt.Fprintf(w, "%s %g\n", name, value)
		return
	}

	var labelPairs []string
	for i := 0; i < len(labels); i += 2 {
		labelPairs = append(labelPairs, fmt.Sprintf(`%s="%s"`, labels[i], labels[i+1]))
	}

	fmt.Fprintf(w, "%s{%s} %g\n", name, strings.Join(labelPairs, ","), value)
}

func stateToValue(state string) float64 {
	switch state {
	case "CLOSED":
		return 0
	case "OPEN":
		return 1
	case "HALF_OPEN":
		return 2
	default:
		return -1
	}
}

func healthToValue(status string) float64 {
	switch status {
	case "unhealthy":
		return 0
	case "degraded":
		return 1
	case "healthy":
		return 2
	default:
		return -1
	}
}

// writeHistograms writes histogram metrics in Prometheus format.
func writeHistograms(w io.Writer, name, help string, histograms map[string]HistogramData, labelName string) {
	if len(histograms) == 0 {
		return
	}

	fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	fmt.Fprintf(w, "# TYPE %s histogram\n", name)

	for key, data := range histograms {
		// Write bucket counts (cumulative)
		cumulative := uint64(0)
		for i, bound := range data.Buckets {
			cumulative += data.BucketCount[i]
			fmt.Fprintf(w, "%s_bucket{%s=\"%s\",le=\"%g\"} %d\n",
				name, labelName, key, bound, cumulative)
		}
		// +Inf bucket
		cumulative += data.BucketCount[len(data.Buckets)]
		fmt.Fprintf(w, "%s_bucket{%s=\"%s\",le=\"+Inf\"} %d\n",
			name, labelName, key, cumulative)

		// Write sum and count
		fmt.Fprintf(w, "%s_sum{%s=\"%s\"} %g\n", name, labelName, key, data.Sum)
		fmt.Fprintf(w, "%s_count{%s=\"%s\"} %d\n", name, labelName, key, data.Count)
	}
}
