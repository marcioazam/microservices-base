package crypto

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds Prometheus metrics for crypto operations.
type Metrics struct {
	encryptTotal  *prometheus.CounterVec
	decryptTotal  *prometheus.CounterVec
	signTotal     *prometheus.CounterVec
	verifyTotal   *prometheus.CounterVec
	latency       *prometheus.HistogramVec
	errorsTotal   *prometheus.CounterVec
	fallbackTotal prometheus.Counter
}

// NewMetrics creates and registers crypto metrics.
func NewMetrics(registry prometheus.Registerer) *Metrics {
	m := &Metrics{
		encryptTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "iam_crypto_encrypt_total",
				Help: "Total number of encryption operations",
			},
			[]string{"status"},
		),
		decryptTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "iam_crypto_decrypt_total",
				Help: "Total number of decryption operations",
			},
			[]string{"status"},
		),
		signTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "iam_crypto_sign_total",
				Help: "Total number of signing operations",
			},
			[]string{"status"},
		),
		verifyTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "iam_crypto_verify_total",
				Help: "Total number of verification operations",
			},
			[]string{"status"},
		),
		latency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "iam_crypto_latency_seconds",
				Help:    "Latency of crypto operations in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
			[]string{"operation"},
		),
		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "iam_crypto_errors_total",
				Help: "Total number of crypto errors by error code",
			},
			[]string{"error_code"},
		),
		fallbackTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "iam_crypto_fallback_total",
				Help: "Total number of fallbacks to unencrypted mode",
			},
		),
	}

	if registry != nil {
		registry.MustRegister(
			m.encryptTotal,
			m.decryptTotal,
			m.signTotal,
			m.verifyTotal,
			m.latency,
			m.errorsTotal,
			m.fallbackTotal,
		)
	}

	return m
}

// RecordEncrypt records an encryption operation.
func (m *Metrics) RecordEncrypt(status string, duration time.Duration) {
	if m == nil {
		return
	}
	m.encryptTotal.WithLabelValues(status).Inc()
	m.latency.WithLabelValues("encrypt").Observe(duration.Seconds())
}

// RecordDecrypt records a decryption operation.
func (m *Metrics) RecordDecrypt(status string, duration time.Duration) {
	if m == nil {
		return
	}
	m.decryptTotal.WithLabelValues(status).Inc()
	m.latency.WithLabelValues("decrypt").Observe(duration.Seconds())
}

// RecordSign records a signing operation.
func (m *Metrics) RecordSign(status string, duration time.Duration) {
	if m == nil {
		return
	}
	m.signTotal.WithLabelValues(status).Inc()
	m.latency.WithLabelValues("sign").Observe(duration.Seconds())
}

// RecordVerify records a verification operation.
func (m *Metrics) RecordVerify(status string, duration time.Duration) {
	if m == nil {
		return
	}
	m.verifyTotal.WithLabelValues(status).Inc()
	m.latency.WithLabelValues("verify").Observe(duration.Seconds())
}

// RecordError records a crypto error.
func (m *Metrics) RecordError(errorCode string) {
	if m == nil {
		return
	}
	m.errorsTotal.WithLabelValues(errorCode).Inc()
}

// RecordFallback records a fallback to unencrypted mode.
func (m *Metrics) RecordFallback() {
	if m == nil {
		return
	}
	m.fallbackTotal.Inc()
}

// GetEncryptTotal returns the encrypt counter for testing.
func (m *Metrics) GetEncryptTotal() *prometheus.CounterVec {
	return m.encryptTotal
}

// GetDecryptTotal returns the decrypt counter for testing.
func (m *Metrics) GetDecryptTotal() *prometheus.CounterVec {
	return m.decryptTotal
}

// GetSignTotal returns the sign counter for testing.
func (m *Metrics) GetSignTotal() *prometheus.CounterVec {
	return m.signTotal
}

// GetVerifyTotal returns the verify counter for testing.
func (m *Metrics) GetVerifyTotal() *prometheus.CounterVec {
	return m.verifyTotal
}
