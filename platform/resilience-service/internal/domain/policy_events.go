package domain

import (
	"time"
)

// PolicyEventType represents the type of policy event.
type PolicyEventType string

const (
	// PolicyCreated indicates a policy was created.
	PolicyCreated PolicyEventType = "policy_created"
	// PolicyUpdated indicates a policy was updated.
	PolicyUpdated PolicyEventType = "policy_updated"
	// PolicyDeleted indicates a policy was deleted.
	PolicyDeleted PolicyEventType = "policy_deleted"
)

// PolicyEvent represents a service-specific policy change event.
type PolicyEvent struct {
	Type          PolicyEventType `json:"type"`
	PolicyName    string          `json:"policy_name"`
	Version       int64           `json:"version"`
	CorrelationID string          `json:"correlation_id"`
	Timestamp     time.Time       `json:"timestamp"`
}
