package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the service
type Metrics struct {
	UploadTotal       *prometheus.CounterVec
	UploadDuration    *prometheus.HistogramVec
	UploadSize        *prometheus.HistogramVec
	UploadErrors      *prometheus.CounterVec
	ActiveUploads     prometheus.Gauge
	StorageOperations *prometheus.CounterVec
	StorageDuration   *prometheus.HistogramVec
	ScanTotal         *prometheus.CounterVec
	ScanDuration      *prometheus.HistogramVec
	RateLimitHits     *prometheus.CounterVec
	AuthFailures      *prometheus.CounterVec
}

// NewMetrics creates and registers all metrics
func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		UploadTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "upload_total",
				Help:      "Total number of file uploads",
			},
			[]string{"tenant_id", "status", "mime_type"},
		),
		UploadDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "upload_duration_seconds",
				Help:      "Duration of file uploads in seconds",
				Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
			},
			[]string{"tenant_id"},
		),
		UploadSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "upload_size_bytes",
				Help:      "Size of uploaded files in bytes",
				Buckets:   prometheus.ExponentialBuckets(1024, 4, 10), // 1KB to ~1GB
			},
			[]string{"tenant_id", "mime_type"},
		),
		UploadErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "upload_errors_total",
				Help:      "Total number of upload errors",
			},
			[]string{"tenant_id", "error_type"},
		),
		ActiveUploads: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_uploads",
				Help:      "Number of currently active uploads",
			},
		),
		StorageOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "storage_operations_total",
				Help:      "Total number of storage operations",
			},
			[]string{"operation", "status"},
		),
		StorageDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "storage_duration_seconds",
				Help:      "Duration of storage operations in seconds",
				Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
			},
			[]string{"operation"},
		),
		ScanTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "scan_total",
				Help:      "Total number of malware scans",
			},
			[]string{"result"},
		),
		ScanDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "scan_duration_seconds",
				Help:      "Duration of malware scans in seconds",
				Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30},
			},
			[]string{},
		),
		RateLimitHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rate_limit_hits_total",
				Help:      "Total number of rate limit hits",
			},
			[]string{"tenant_id"},
		),
		AuthFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "auth_failures_total",
				Help:      "Total number of authentication failures",
			},
			[]string{"reason"},
		),
	}
}

// RecordUpload records a successful upload
func (m *Metrics) RecordUpload(tenantID, mimeType string, size int64, duration float64) {
	m.UploadTotal.WithLabelValues(tenantID, "success", mimeType).Inc()
	m.UploadDuration.WithLabelValues(tenantID).Observe(duration)
	m.UploadSize.WithLabelValues(tenantID, mimeType).Observe(float64(size))
}

// RecordUploadError records an upload error
func (m *Metrics) RecordUploadError(tenantID, errorType string) {
	m.UploadTotal.WithLabelValues(tenantID, "error", "").Inc()
	m.UploadErrors.WithLabelValues(tenantID, errorType).Inc()
}

// RecordStorageOperation records a storage operation
func (m *Metrics) RecordStorageOperation(operation, status string, duration float64) {
	m.StorageOperations.WithLabelValues(operation, status).Inc()
	m.StorageDuration.WithLabelValues(operation).Observe(duration)
}

// RecordScan records a malware scan
func (m *Metrics) RecordScan(result string, duration float64) {
	m.ScanTotal.WithLabelValues(result).Inc()
	m.ScanDuration.WithLabelValues().Observe(duration)
}

// RecordRateLimitHit records a rate limit hit
func (m *Metrics) RecordRateLimitHit(tenantID string) {
	m.RateLimitHits.WithLabelValues(tenantID).Inc()
}

// RecordAuthFailure records an authentication failure
func (m *Metrics) RecordAuthFailure(reason string) {
	m.AuthFailures.WithLabelValues(reason).Inc()
}

// IncrementActiveUploads increments active uploads counter
func (m *Metrics) IncrementActiveUploads() {
	m.ActiveUploads.Inc()
}

// DecrementActiveUploads decrements active uploads counter
func (m *Metrics) DecrementActiveUploads() {
	m.ActiveUploads.Dec()
}
