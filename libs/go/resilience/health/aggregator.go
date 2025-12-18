// Package health implements health aggregation and monitoring.
package health

import (
	"context"
	"sync"
	"time"

	"github.com/auth-platform/libs/go/resilience"
)

// Aggregator implements the HealthAggregator interface.
type Aggregator struct {
	mu            sync.RWMutex
	services      map[string]*serviceEntry
	eventEmitter  resilience.EventEmitter
	correlationFn func() string
}

type serviceEntry struct {
	checker   HealthChecker
	status    HealthStatus
	message   string
	lastCheck time.Time
}

// Config holds aggregator creation options.
type Config struct {
	EventEmitter  resilience.EventEmitter
	CorrelationFn func() string
}

// NewAggregator creates a new health aggregator.
func NewAggregator(cfg Config) *Aggregator {
	return &Aggregator{
		services:      make(map[string]*serviceEntry),
		eventEmitter:  cfg.EventEmitter,
		correlationFn: resilience.EnsureCorrelationFunc(cfg.CorrelationFn),
	}
}

// GetAggregatedHealth returns overall system health.
func (a *Aggregator) GetAggregatedHealth(ctx context.Context) (*AggregatedHealth, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	services := make(map[string]ServiceHealth, len(a.services))
	overallStatus := HealthHealthy

	for name, entry := range a.services {
		services[name] = ServiceHealth{
			Name:      name,
			Status:    entry.status,
			Message:   entry.message,
			LastCheck: entry.lastCheck,
		}
		overallStatus = aggregateStatus(overallStatus, entry.status)
	}

	return &AggregatedHealth{
		Status:    overallStatus,
		Services:  services,
		Timestamp: resilience.NowUTC(),
	}, nil
}

// RegisterService adds a service to monitor.
func (a *Aggregator) RegisterService(name string, checker HealthChecker) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.services[name] = &serviceEntry{
		checker:   checker,
		status:    HealthHealthy,
		lastCheck: resilience.NowUTC(),
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
func (a *Aggregator) UpdateHealth(name string, status HealthStatus, message string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	entry, ok := a.services[name]
	if !ok {
		entry = &serviceEntry{}
		a.services[name] = entry
	}

	previousStatus := entry.status
	entry.status = status
	entry.message = message
	entry.lastCheck = resilience.NowUTC()

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
		status = HealthUnhealthy
		message = err.Error()
	}

	a.UpdateHealth(name, status, message)
}

// emitHealthChangeEvent emits a health change event.
func (a *Aggregator) emitHealthChangeEvent(name string, prev, new HealthStatus, message string) {
	event := resilience.NewEvent(resilience.EventHealthChange, name).
		WithCorrelationID(a.correlationFn()).
		WithMetadata("previous_status", string(prev)).
		WithMetadata("new_status", string(new)).
		WithMetadata("message", message)

	resilience.EmitEvent(a.eventEmitter, *event)
}

// aggregateStatus returns the worst status between two statuses.
func aggregateStatus(current, new HealthStatus) HealthStatus {
	statusPriority := map[HealthStatus]int{
		HealthHealthy:   0,
		HealthDegraded:  1,
		HealthUnhealthy: 2,
	}

	if statusPriority[new] > statusPriority[current] {
		return new
	}
	return current
}

// AggregateStatuses aggregates multiple health statuses into one.
func AggregateStatuses(statuses []HealthStatus) HealthStatus {
	result := HealthHealthy
	for _, status := range statuses {
		result = aggregateStatus(result, status)
	}
	return result
}

// Services returns an iterator over all registered services.
func (a *Aggregator) Services() func(yield func(ServiceHealth) bool) {
	return func(yield func(ServiceHealth) bool) {
		a.mu.RLock()
		defer a.mu.RUnlock()
		for name, entry := range a.services {
			sh := ServiceHealth{
				Name:      name,
				Status:    entry.status,
				Message:   entry.message,
				LastCheck: entry.lastCheck,
			}
			if !yield(sh) {
				return
			}
		}
	}
}
