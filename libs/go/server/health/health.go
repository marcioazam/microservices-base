// Package health provides health status types and aggregation.
package health

import (
	"context"
	"sync"
	"time"
)

// Status represents the health status of a component.
type Status int

const (
	Healthy Status = iota
	Degraded
	Unhealthy
)

func (s Status) String() string {
	switch s {
	case Healthy:
		return "healthy"
	case Degraded:
		return "degraded"
	case Unhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// Check represents a health check result.
type Check struct {
	Name      string
	Status    Status
	Message   string
	Timestamp time.Time
	Details   map[string]interface{}
}

// Checker is the interface for health checkers.
type Checker interface {
	Name() string
	Check(ctx context.Context) Check
}

// CheckerFunc is a function adapter for Checker.
type CheckerFunc struct {
	name string
	fn   func(ctx context.Context) Check
}

// NewCheckerFunc creates a new CheckerFunc.
func NewCheckerFunc(name string, fn func(ctx context.Context) Check) *CheckerFunc {
	return &CheckerFunc{name: name, fn: fn}
}

func (c *CheckerFunc) Name() string {
	return c.name
}

func (c *CheckerFunc) Check(ctx context.Context) Check {
	return c.fn(ctx)
}

// Aggregator aggregates health checks from multiple sources.
type Aggregator struct {
	mu       sync.RWMutex
	checkers []Checker
	results  map[string]Check
	onChange func(Check)
}

// NewAggregator creates a new health aggregator.
func NewAggregator() *Aggregator {
	return &Aggregator{
		checkers: make([]Checker, 0),
		results:  make(map[string]Check),
	}
}

// Register adds a health checker.
func (a *Aggregator) Register(checker Checker) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.checkers = append(a.checkers, checker)
}

// RegisterFunc adds a health check function.
func (a *Aggregator) RegisterFunc(name string, fn func(ctx context.Context) Check) {
	a.Register(NewCheckerFunc(name, fn))
}

// OnChange sets a callback for status changes.
func (a *Aggregator) OnChange(fn func(Check)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onChange = fn
}

// Check runs all health checks and returns the aggregated status.
func (a *Aggregator) Check(ctx context.Context) (Status, []Check) {
	a.mu.RLock()
	checkers := make([]Checker, len(a.checkers))
	copy(checkers, a.checkers)
	a.mu.RUnlock()

	checks := make([]Check, 0, len(checkers))
	for _, checker := range checkers {
		check := checker.Check(ctx)
		checks = append(checks, check)
		a.updateResult(check)
	}

	return AggregateStatuses(checks), checks
}

// GetStatus returns the current aggregated status without running checks.
func (a *Aggregator) GetStatus() Status {
	a.mu.RLock()
	defer a.mu.RUnlock()

	checks := make([]Check, 0, len(a.results))
	for _, check := range a.results {
		checks = append(checks, check)
	}

	return AggregateStatuses(checks)
}

// GetChecks returns the current check results.
func (a *Aggregator) GetChecks() []Check {
	a.mu.RLock()
	defer a.mu.RUnlock()

	checks := make([]Check, 0, len(a.results))
	for _, check := range a.results {
		checks = append(checks, check)
	}
	return checks
}

func (a *Aggregator) updateResult(check Check) {
	a.mu.Lock()
	oldCheck, exists := a.results[check.Name]
	a.results[check.Name] = check
	onChange := a.onChange
	a.mu.Unlock()

	if onChange != nil && (!exists || oldCheck.Status != check.Status) {
		onChange(check)
	}
}

// AggregateStatuses returns the worst status from a list of checks.
// Returns Healthy if the list is empty.
func AggregateStatuses(checks []Check) Status {
	if len(checks) == 0 {
		return Healthy
	}

	worst := Healthy
	for _, check := range checks {
		if check.Status > worst {
			worst = check.Status
		}
	}
	return worst
}

// AggregateStatusValues returns the worst status from a list of status values.
// Returns Healthy if the list is empty.
func AggregateStatusValues(statuses []Status) Status {
	if len(statuses) == 0 {
		return Healthy
	}

	worst := Healthy
	for _, status := range statuses {
		if status > worst {
			worst = status
		}
	}
	return worst
}

// NewHealthyCheck creates a healthy check result.
func NewHealthyCheck(name string) Check {
	return Check{
		Name:      name,
		Status:    Healthy,
		Timestamp: time.Now(),
	}
}

// NewDegradedCheck creates a degraded check result.
func NewDegradedCheck(name, message string) Check {
	return Check{
		Name:      name,
		Status:    Degraded,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewUnhealthyCheck creates an unhealthy check result.
func NewUnhealthyCheck(name, message string) Check {
	return Check{
		Name:      name,
		Status:    Unhealthy,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// WithDetails adds details to a check.
func (c Check) WithDetails(details map[string]interface{}) Check {
	c.Details = details
	return c
}
