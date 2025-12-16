package domain

import "context"

// ResiliencePolicy combines all resilience settings.
type ResiliencePolicy struct {
	Name           string                `json:"name" yaml:"name"`
	Version        int                   `json:"version" yaml:"version"`
	CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker,omitempty" yaml:"circuitBreaker,omitempty"`
	Retry          *RetryConfig          `json:"retry,omitempty" yaml:"retry,omitempty"`
	Timeout        *TimeoutConfig        `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	RateLimit      *RateLimitConfig      `json:"rate_limit,omitempty" yaml:"rateLimit,omitempty"`
	Bulkhead       *BulkheadConfig       `json:"bulkhead,omitempty" yaml:"bulkhead,omitempty"`
}

// PolicyEvent represents a policy change event.
type PolicyEvent struct {
	Type   PolicyEventType
	Policy *ResiliencePolicy
}

// PolicyEventType represents the type of policy event.
type PolicyEventType string

const (
	PolicyCreated PolicyEventType = "created"
	PolicyUpdated PolicyEventType = "updated"
	PolicyDeleted PolicyEventType = "deleted"
)

// PolicyEngine manages resilience policies.
type PolicyEngine interface {
	// GetPolicy retrieves policy by name.
	GetPolicy(name string) (*ResiliencePolicy, error)

	// UpdatePolicy updates or creates a policy.
	UpdatePolicy(policy *ResiliencePolicy) error

	// DeletePolicy removes a policy.
	DeletePolicy(name string) error

	// ListPolicies returns all policies.
	ListPolicies() ([]*ResiliencePolicy, error)

	// WatchPolicies streams policy changes.
	WatchPolicies(ctx context.Context) (<-chan PolicyEvent, error)

	// Validate validates a policy configuration.
	Validate(policy *ResiliencePolicy) error
}
