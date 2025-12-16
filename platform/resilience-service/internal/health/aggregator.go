// Package health implements health aggregation and monitoring.
package health

import (
	"context"
	"sync"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
)

// Aggregator implements the HealthAggregator interface.
type Aggregator struct {
	mu            sync.RWMutex
	services      map[string]*serviceEntry
	eventEmitter  domain.EventEmitter
	correlationFn func() string
}

type serviceEntry struct {
	checker   domain.HealthChecker
	status    domain.HealthStatus
	message   string
	lastCheck time.Time
}

// Config holds aggregator creation options.
type Config struct {
	EventEmitter  domain.EventEmitter
	CorrelationFn func() string
}

// NewAggregator creates a new health aggregator.
func NewAggregator(cfg Config) *Aggregator {
	correlationFn := cfg.CorrelationFn
	if correlationFn == nil {
		correlationFn = func() string { return "" }
	}

	return &Aggregator{
		services:      make(map[string]*serviceEntry),
		eventEmitter:  cfg.EventEmitter,
		correlationFn: correlationFn,
	}
}

// GetAggregatedHealth returns overall system health.
func (a *Aggregator) GetAggregatedHealth(ctx context.Context) (*domain.AggregatedHealth, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	services := make(map[string]domain.ServiceHealth, len(a.services))
	overallStatus := domain.HealthHealthy

	for name, entry := range a.services {
		services[name] = domain.ServiceHealth{
			Name:      name,
			Status:    entry.status,
			Message:   entry.message,
			LastCheck: entry.lastCheck,
		}

		// Update overall status based on worst service status
		overallStatus = aggregateStatus(overallStatus, entry.status)
	}

	return &domain.AggregatedHealth{
		Status:    overallStatus,
		Services:  services,
		Timestamp: time.Now(),
	}, nil
}

// RegisterService adds a service to monitor.
func (a *Aggregator) RegisterService(name string, checker domain.HealthChecker) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.services[name] = &serviceEntry{
		checker:   checker,
		status:    domain.HealthHealthy,
		lastCheck: time.Now(),
	}

	return nil
}

// UnregisterService removes a service from monitoring.
func (a *Aggregator) UnregisterService(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.services, name)
	return nil
}

// UpdateHealth updates health status for a service.
func (a *Aggregator) UpdateHealth(name string, status domain.HealthStatus, message string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	entry, ok := a.services[name]
	if !ok {
		// Create entry if it doesn't exist
		entry = &serviceEntry{}
		a.services[name] = entry
	}

	previousStatus := entry.status
	entry.status = status
	entry.message = message
	entry.lastCheck = time.Now()

	// Emit event if status changed
	if previousStatus != status {
		a.emitHealthChangeEvent(name, previousStatus, status, message)
	}
}

// CheckAll runs health checks for all registered services.
func (a *Aggregator) CheckAll(ctx context.Context) {
	a.mu.RLock()
	names := make([]string, 0, len(a.services))
	for name := range a.services {
		names = append(names, name)
	}
	a.mu.RUnlock()

	for _, name := range names {
		a.checkService(ctx, name)
	}
}

// checkService runs health check for a single service.
func (a *Aggregator) checkService(ctx context.Context, name string) {
	a.mu.RLock()
	entry, ok := a.services[name]
	if !ok || entry.checker == nil {
		a.mu.RUnlock()
		return
	}
	checker := entry.checker
	a.mu.RUnlock()

	status, message, err := checker.Check(ctx)
	if err != nil {
		status = domain.HealthUnhealthy
		message = err.Error()
	}

	a.UpdateHealth(name, status, message)
}

// emitHealthChangeEvent emits a health change event.
func (a *Aggregator) emitHealthChangeEvent(name string, prev, new domain.HealthStatus, message string) {
	if a.eventEmitter == nil {
		return
	}

	event := domain.ResilienceEvent{
		ID:            generateEventID(),
		Type:          domain.EventHealthChange,
		ServiceName:   name,
		Timestamp:     time.Now(),
		CorrelationID: a.correlationFn(),
		Metadata: map[string]any{
			"previous_status": string(prev),
			"new_status":      string(new),
			"message":         message,
		},
	}

	a.eventEmitter.Emit(event)
}

// aggregateStatus returns the worst status between two statuses.
func aggregateStatus(current, new domain.HealthStatus) domain.HealthStatus {
	// Priority: unhealthy > degraded > healthy
	statusPriority := map[domain.HealthStatus]int{
		domain.HealthHealthy:   0,
		domain.HealthDegraded:  1,
		domain.HealthUnhealthy: 2,
	}

	if statusPriority[new] > statusPriority[current] {
		return new
	}
	return current
}

// AggregateStatuses aggregates multiple health statuses into one.
func AggregateStatuses(statuses []domain.HealthStatus) domain.HealthStatus {
	result := domain.HealthHealthy

	for _, status := range statuses {
		result = aggregateStatus(result, status)
	}

	return result
}

// generateEventID generates a unique event ID.
func generateEventID() string {
	return time.Now().Format("20060102150405.000000000")
}
