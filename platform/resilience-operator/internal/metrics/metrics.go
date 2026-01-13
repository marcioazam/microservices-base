// Package metrics provides Prometheus metrics for the resilience operator.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ReconciliationsTotal counts total reconciliations by result.
	ReconciliationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "resilience_operator_reconciliations_total",
			Help: "Total number of reconciliations by result",
		},
		[]string{"result", "namespace"},
	)

	// ReconciliationDuration measures reconciliation duration.
	ReconciliationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "resilience_operator_reconciliation_duration_seconds",
			Help:    "Duration of reconciliation in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"namespace"},
	)

	// PoliciesTotal counts total policies by status.
	PoliciesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "resilience_operator_policies_total",
			Help: "Total number of policies by status",
		},
		[]string{"status", "namespace"},
	)

	// CircuitBreakersConfigured counts configured circuit breakers.
	CircuitBreakersConfigured = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "resilience_operator_circuit_breakers_configured",
			Help: "Number of circuit breakers configured",
		},
		[]string{"namespace"},
	)

	// RetriesConfigured counts configured retry policies.
	RetriesConfigured = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "resilience_operator_retries_configured",
			Help: "Number of retry policies configured",
		},
		[]string{"namespace"},
	)

	// TimeoutsConfigured counts configured timeout policies.
	TimeoutsConfigured = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "resilience_operator_timeouts_configured",
			Help: "Number of timeout policies configured",
		},
		[]string{"namespace"},
	)

	// HTTPRoutesCreated counts HTTPRoutes created by the operator.
	HTTPRoutesCreated = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "resilience_operator_httproutes_created",
			Help: "Number of HTTPRoutes created",
		},
		[]string{"namespace"},
	)

	// ErrorsTotal counts errors by type.
	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "resilience_operator_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type", "namespace"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		ReconciliationsTotal,
		ReconciliationDuration,
		PoliciesTotal,
		CircuitBreakersConfigured,
		RetriesConfigured,
		TimeoutsConfigured,
		HTTPRoutesCreated,
		ErrorsTotal,
	)
}

// RecordReconciliation records a reconciliation result.
func RecordReconciliation(result, namespace string) {
	ReconciliationsTotal.WithLabelValues(result, namespace).Inc()
}

// RecordReconciliationDuration records reconciliation duration.
func RecordReconciliationDuration(duration float64, namespace string) {
	ReconciliationDuration.WithLabelValues(namespace).Observe(duration)
}

// RecordPolicyStatus records policy status.
func RecordPolicyStatus(status, namespace string, count float64) {
	PoliciesTotal.WithLabelValues(status, namespace).Set(count)
}

// RecordCircuitBreaker records circuit breaker configuration.
func RecordCircuitBreaker(namespace string, count float64) {
	CircuitBreakersConfigured.WithLabelValues(namespace).Set(count)
}

// RecordRetry records retry configuration.
func RecordRetry(namespace string, count float64) {
	RetriesConfigured.WithLabelValues(namespace).Set(count)
}

// RecordTimeout records timeout configuration.
func RecordTimeout(namespace string, count float64) {
	TimeoutsConfigured.WithLabelValues(namespace).Set(count)
}

// RecordHTTPRoute records HTTPRoute creation.
func RecordHTTPRoute(namespace string, count float64) {
	HTTPRoutesCreated.WithLabelValues(namespace).Set(count)
}

// RecordError records an error.
func RecordError(errorType, namespace string) {
	ErrorsTotal.WithLabelValues(errorType, namespace).Inc()
}
