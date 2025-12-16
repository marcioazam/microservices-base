package domain

import (
	"context"
	"time"
)

// HealthStatus represents service health status.
type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
)

// ServiceHealth represents health of a single service.
type ServiceHealth struct {
	Name      string       `json:"name"`
	Status    HealthStatus `json:"status"`
	Message   string       `json:"message,omitempty"`
	LastCheck time.Time    `json:"last_check"`
}

// AggregatedHealth represents system-wide health status.
type AggregatedHealth struct {
	Status    HealthStatus             `json:"status"`
	Services  map[string]ServiceHealth `json:"services"`
	Timestamp time.Time                `json:"timestamp"`
}

// HealthAggregator collects and aggregates service health.
type HealthAggregator interface {
	// GetAggregatedHealth returns overall system health.
	GetAggregatedHealth(ctx context.Context) (*AggregatedHealth, error)

	// RegisterService adds a service to monitor.
	RegisterService(name string, checker HealthChecker) error

	// UnregisterService removes a service from monitoring.
	UnregisterService(name string) error

	// UpdateHealth updates health status for a service.
	UpdateHealth(name string, status HealthStatus, message string)
}

// HealthChecker checks health of a service.
type HealthChecker interface {
	Check(ctx context.Context) (HealthStatus, string, error)
}

// HealthChangeEvent represents a health status change for CAEP.
type HealthChangeEvent struct {
	ServiceName    string       `json:"service_name"`
	PreviousStatus HealthStatus `json:"previous_status"`
	NewStatus      HealthStatus `json:"new_status"`
	Message        string       `json:"message,omitempty"`
	CorrelationID  string       `json:"correlation_id"`
	Timestamp      time.Time    `json:"timestamp"`
}
