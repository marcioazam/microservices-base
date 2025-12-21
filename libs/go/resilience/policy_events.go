package resilience

// PolicyEventType represents the type of policy event.
type PolicyEventType string

const (
	// PolicyCreated indicates a new policy was created.
	PolicyCreated PolicyEventType = "created"
	// PolicyUpdated indicates an existing policy was updated.
	PolicyUpdated PolicyEventType = "updated"
	// PolicyDeleted indicates a policy was deleted.
	PolicyDeleted PolicyEventType = "deleted"
)

// PolicyEvent represents a policy change event.
type PolicyEvent struct {
	Type   PolicyEventType
	Policy *ResiliencePolicy
}
