package domain

import (
	"context"
	"time"
)

// BulkheadConfig defines bulkhead behavior.
type BulkheadConfig struct {
	MaxConcurrent int           `json:"max_concurrent" yaml:"maxConcurrent"`
	MaxQueue      int           `json:"max_queue" yaml:"maxQueue"`
	QueueTimeout  time.Duration `json:"queue_timeout" yaml:"queueTimeout"`
}

// BulkheadMetrics reports bulkhead utilization.
type BulkheadMetrics struct {
	ActiveCount   int
	QueuedCount   int
	RejectedCount int64
}

// Bulkhead provides isolation through concurrency limits.
type Bulkhead interface {
	// Acquire attempts to acquire a permit.
	Acquire(ctx context.Context) error

	// Release returns a permit.
	Release()

	// GetMetrics returns current utilization.
	GetMetrics() BulkheadMetrics
}

// BulkheadManager manages multiple bulkhead partitions.
type BulkheadManager interface {
	// GetBulkhead returns the bulkhead for a partition.
	GetBulkhead(partition string) Bulkhead

	// GetAllMetrics returns metrics for all partitions.
	GetAllMetrics() map[string]BulkheadMetrics
}

// BulkheadRejectionEvent represents a bulkhead rejection for observability.
type BulkheadRejectionEvent struct {
	Partition     string    `json:"partition"`
	ActiveCount   int       `json:"active_count"`
	QueuedCount   int       `json:"queued_count"`
	CorrelationID string    `json:"correlation_id"`
	Timestamp     time.Time `json:"timestamp"`
}
