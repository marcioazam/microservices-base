// Package grpc provides gRPC types for the resilience service.
// These types mirror the protobuf definitions until proto generation is configured.
package grpc

import (
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Policy represents a resilience policy configuration.
type Policy struct {
	Name           string                 `json:"name"`
	Version        int32                  `json:"version"`
	CircuitBreaker *CircuitBreakerConfig  `json:"circuit_breaker,omitempty"`
	Retry          *RetryConfig           `json:"retry,omitempty"`
	Timeout        *TimeoutConfig         `json:"timeout,omitempty"`
	RateLimit      *RateLimitConfig       `json:"rate_limit,omitempty"`
	Bulkhead       *BulkheadConfig        `json:"bulkhead,omitempty"`
	CreatedAt      *timestamppb.Timestamp `json:"created_at,omitempty"`
	UpdatedAt      *timestamppb.Timestamp `json:"updated_at,omitempty"`
}

// CircuitBreakerConfig defines circuit breaker parameters.
type CircuitBreakerConfig struct {
	FailureThreshold int32                `json:"failure_threshold"`
	SuccessThreshold int32                `json:"success_threshold"`
	Timeout          *durationpb.Duration `json:"timeout"`
	ProbeCount       int32                `json:"probe_count"`
}

// RetryConfig defines retry behavior parameters.
type RetryConfig struct {
	MaxAttempts   int32                `json:"max_attempts"`
	BaseDelay     *durationpb.Duration `json:"base_delay"`
	MaxDelay      *durationpb.Duration `json:"max_delay"`
	Multiplier    float64              `json:"multiplier"`
	JitterPercent float64              `json:"jitter_percent"`
}

// TimeoutConfig defines timeout parameters.
type TimeoutConfig struct {
	DefaultTimeout *durationpb.Duration `json:"default_timeout"`
	MaxTimeout     *durationpb.Duration `json:"max_timeout"`
}

// RateLimitConfig defines rate limiting parameters.
type RateLimitConfig struct {
	Algorithm string               `json:"algorithm"`
	Limit     int32                `json:"limit"`
	Window    *durationpb.Duration `json:"window"`
	BurstSize int32                `json:"burst_size"`
}

// BulkheadConfig defines bulkhead isolation parameters.
type BulkheadConfig struct {
	MaxConcurrent int32                `json:"max_concurrent"`
	MaxQueue      int32                `json:"max_queue"`
	QueueTimeout  *durationpb.Duration `json:"queue_timeout"`
}

// Request/Response types

// CreatePolicyRequest creates a new policy.
type CreatePolicyRequest struct {
	Name           string               `json:"name"`
	CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker,omitempty"`
	Retry          *RetryConfig          `json:"retry,omitempty"`
	Timeout        *TimeoutConfig        `json:"timeout,omitempty"`
	RateLimit      *RateLimitConfig      `json:"rate_limit,omitempty"`
	Bulkhead       *BulkheadConfig       `json:"bulkhead,omitempty"`
}

// CreatePolicyResponse returns the created policy.
type CreatePolicyResponse struct {
	Policy *Policy `json:"policy"`
}

// GetPolicyRequest retrieves a policy by name.
type GetPolicyRequest struct {
	Name string `json:"name"`
}

// GetPolicyResponse returns the requested policy.
type GetPolicyResponse struct {
	Policy *Policy `json:"policy"`
}

// UpdatePolicyRequest updates an existing policy.
type UpdatePolicyRequest struct {
	Name           string               `json:"name"`
	CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker,omitempty"`
	Retry          *RetryConfig          `json:"retry,omitempty"`
	Timeout        *TimeoutConfig        `json:"timeout,omitempty"`
	RateLimit      *RateLimitConfig      `json:"rate_limit,omitempty"`
	Bulkhead       *BulkheadConfig       `json:"bulkhead,omitempty"`
}

// UpdatePolicyResponse returns the updated policy.
type UpdatePolicyResponse struct {
	Policy *Policy `json:"policy"`
}

// DeletePolicyRequest deletes a policy by name.
type DeletePolicyRequest struct {
	Name string `json:"name"`
}

// DeletePolicyResponse confirms deletion.
type DeletePolicyResponse struct {
	Success bool `json:"success"`
}

// ListPoliciesRequest lists all policies.
type ListPoliciesRequest struct {
	PageSize  int32  `json:"page_size"`
	PageToken string `json:"page_token"`
}

// ListPoliciesResponse returns a list of policies.
type ListPoliciesResponse struct {
	Policies      []*Policy `json:"policies"`
	NextPageToken string    `json:"next_page_token"`
	TotalCount    int32     `json:"total_count"`
}

// WatchPoliciesRequest starts watching policy changes.
type WatchPoliciesRequest struct {
	PolicyNames []string `json:"policy_names"`
}

// PolicyEvent represents a policy change event.
type PolicyEvent struct {
	EventId    string                 `json:"event_id"`
	Type       PolicyEventType        `json:"type"`
	PolicyName string                 `json:"policy_name"`
	Version    int32                  `json:"version"`
	Timestamp  *timestamppb.Timestamp `json:"timestamp"`
	Policy     *Policy                `json:"policy,omitempty"`
}

// PolicyEventType represents the type of policy event.
type PolicyEventType int32

const (
	PolicyEventType_POLICY_EVENT_TYPE_UNSPECIFIED PolicyEventType = 0
	PolicyEventType_POLICY_EVENT_TYPE_CREATED     PolicyEventType = 1
	PolicyEventType_POLICY_EVENT_TYPE_UPDATED     PolicyEventType = 2
	PolicyEventType_POLICY_EVENT_TYPE_DELETED     PolicyEventType = 3
)

// ResilienceService_WatchPoliciesServer is the server stream interface.
type ResilienceService_WatchPoliciesServer interface {
	Send(*PolicyEvent) error
	Context() interface{ Done() <-chan struct{} }
}
