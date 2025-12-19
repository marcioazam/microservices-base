// Package domain provides health types for the resilience service.
// This package re-exports types from libs/go/health for backward compatibility.
package domain

import (
	"context"
	"time"

	libhealth "github.com/auth-platform/libs/go/server/health"
)

// HealthStatus represents service health status.
type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
)

// ToLibStatus converts to library health status.
func (s HealthStatus) ToLibStatus() libhealth.Status {
	switch s {
	case HealthHealthy:
		return libhealth.Healthy
	case HealthDegraded:
		return libhealth.Degraded
	case HealthUnhealthy:
		return libhealth.Unhealthy
	default:
		return libhealth.Unhealthy
	}
}

// FromLibStatus converts from library health status.
func FromLibStatus(s libhealth.Status) HealthStatus {
	switch s {
	case libhealth.Healthy:
		return HealthHealthy
	case libhealth.Degraded:
		return HealthDegraded
	case libhealth.Unhealthy:
		return HealthUnhealthy
	default:
		return HealthUnhealthy
	}
}

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

// LibAggregator is a type alias for the library aggregator.
type LibAggregator = libhealth.Aggregator

// NewLibAggregator creates a new library health aggregator.
func NewLibAggregator() *LibAggregator {
	return libhealth.NewAggregator()
}

// AggregateHealthStatuses aggregates multiple health statuses using library function.
func AggregateHealthStatuses(statuses []HealthStatus) HealthStatus {
	libStatuses := make([]libhealth.Status, len(statuses))
	for i, s := range statuses {
		libStatuses[i] = s.ToLibStatus()
	}
	return FromLibStatus(libhealth.AggregateStatusValues(libStatuses))
}
