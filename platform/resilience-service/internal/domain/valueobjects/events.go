// Package valueobjects defines domain value objects.
package valueobjects

import (
	"time"
)

// HealthState represents component health state.
type HealthState string

const (
	HealthHealthy   HealthState = "healthy"
	HealthUnhealthy HealthState = "unhealthy"
	HealthDegraded  HealthState = "degraded"
	HealthUnknown   HealthState = "unknown"
)

// HealthStatus represents component health state.
type HealthStatus struct {
	Status    HealthState    `json:"status"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// NewHealthStatus creates a new health status.
func NewHealthStatus(status HealthState, message string) HealthStatus {
	return HealthStatus{
		Status:    status,
		Message:   message,
		Details:   make(map[string]any),
		Timestamp: time.Now().UTC(),
	}
}

// IsHealthy returns true if the status is healthy.
func (h HealthStatus) IsHealthy() bool {
	return h.Status == HealthHealthy
}

// WithDetail adds a detail to the health status.
func (h HealthStatus) WithDetail(key string, value any) HealthStatus {
	if h.Details == nil {
		h.Details = make(map[string]any)
	}
	h.Details[key] = value
	return h
}

// PolicyEventType represents policy lifecycle event types.
type PolicyEventType string

const (
	PolicyCreated PolicyEventType = "created"
	PolicyUpdated PolicyEventType = "updated"
	PolicyDeleted PolicyEventType = "deleted"
)

// PolicyEvent represents policy lifecycle events.
type PolicyEvent struct {
	ID         string          `json:"id"`
	Type       PolicyEventType `json:"type"`
	PolicyName string          `json:"policy_name"`
	Version    int             `json:"version"`
	OccurredAt time.Time       `json:"timestamp"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
}

// NewPolicyEvent creates a new policy event.
func NewPolicyEvent(eventType PolicyEventType, policyName string, version int) PolicyEvent {
	return PolicyEvent{
		ID:         generateEventID(),
		Type:       eventType,
		PolicyName: policyName,
		Version:    version,
		OccurredAt: time.Now().UTC(),
		Metadata:   make(map[string]any),
	}
}

// WithMetadata adds metadata to the policy event.
func (p PolicyEvent) WithMetadata(key string, value any) PolicyEvent {
	// Create a copy of the metadata map to ensure immutability
	newMetadata := make(map[string]any)
	for k, v := range p.Metadata {
		newMetadata[k] = v
	}
	newMetadata[key] = value
	
	p.Metadata = newMetadata
	return p
}

// EventID returns the event ID (implements DomainEvent).
func (p PolicyEvent) EventID() string {
	return p.ID
}

// EventType returns the event type (implements DomainEvent).
func (p PolicyEvent) EventType() string {
	return string(p.Type)
}

// Timestamp returns the event timestamp (implements DomainEvent).
func (p PolicyEvent) Timestamp() time.Time {
	return p.OccurredAt
}

// AggregateID returns the policy name as aggregate ID (implements DomainEvent).
func (p PolicyEvent) AggregateID() string {
	return p.PolicyName
}

// ExecutionMetrics captures resilience execution statistics.
type ExecutionMetrics struct {
	PolicyName      string        `json:"policy_name"`
	ExecutionTime   time.Duration `json:"execution_time"`
	Success         bool          `json:"success"`
	CircuitState    string        `json:"circuit_state,omitempty"`
	RetryAttempts   int           `json:"retry_attempts,omitempty"`
	RateLimited     bool          `json:"rate_limited,omitempty"`
	BulkheadQueued  bool          `json:"bulkhead_queued,omitempty"`
	OccurredAt      time.Time     `json:"timestamp"`
}

// NewExecutionMetrics creates new execution metrics.
func NewExecutionMetrics(policyName string, executionTime time.Duration, success bool) ExecutionMetrics {
	return ExecutionMetrics{
		PolicyName:    policyName,
		ExecutionTime: executionTime,
		Success:       success,
		OccurredAt:    time.Now().UTC(),
	}
}

// WithCircuitState adds circuit breaker state to metrics.
func (e ExecutionMetrics) WithCircuitState(state string) ExecutionMetrics {
	e.CircuitState = state
	return e
}

// WithRetryAttempts adds retry attempts to metrics.
func (e ExecutionMetrics) WithRetryAttempts(attempts int) ExecutionMetrics {
	e.RetryAttempts = attempts
	return e
}

// WithRateLimit adds rate limiting information to metrics.
func (e ExecutionMetrics) WithRateLimit(limited bool) ExecutionMetrics {
	e.RateLimited = limited
	return e
}

// WithBulkheadQueue adds bulkhead queue information to metrics.
func (e ExecutionMetrics) WithBulkheadQueue(queued bool) ExecutionMetrics {
	e.BulkheadQueued = queued
	return e
}

// DomainEvent represents a generic domain event.
type DomainEvent interface {
	EventID() string
	EventType() string
	Timestamp() time.Time
	AggregateID() string
}

// BaseDomainEvent provides common domain event functionality.
type BaseDomainEvent struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	AggregateId string    `json:"aggregate_id"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// EventID returns the event ID.
func (e BaseDomainEvent) EventID() string {
	return e.ID
}

// EventType returns the event type.
func (e BaseDomainEvent) EventType() string {
	return e.Type
}

// Timestamp returns the event timestamp.
func (e BaseDomainEvent) Timestamp() time.Time {
	return e.OccurredAt
}

// AggregateID returns the aggregate ID.
func (e BaseDomainEvent) AggregateID() string {
	return e.AggregateId
}

// NewBaseDomainEvent creates a new base domain event.
func NewBaseDomainEvent(eventType, aggregateID string) BaseDomainEvent {
	return BaseDomainEvent{
		ID:          generateEventID(),
		Type:        eventType,
		AggregateId: aggregateID,
		OccurredAt:  time.Now().UTC(),
	}
}

// generateEventID generates a unique event ID.
func generateEventID() string {
	// Simple implementation - in production, use UUID or similar
	return time.Now().Format("20060102150405.000000")
}