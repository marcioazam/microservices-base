// Package v1 contains API Schema definitions for the resilience v1 API group.
// +kubebuilder:object:generate=true
// +groupName=resilience.auth-platform.github.com
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResiliencePolicySpec defines the desired state of ResiliencePolicy.
type ResiliencePolicySpec struct {
	// TargetRef identifies the target service for this policy.
	// +kubebuilder:validation:Required
	TargetRef TargetReference `json:"targetRef"`

	// CircuitBreaker configuration for failure isolation.
	// +optional
	CircuitBreaker *CircuitBreakerConfig `json:"circuitBreaker,omitempty"`

	// Retry configuration for automatic retries.
	// +optional
	Retry *RetryConfig `json:"retry,omitempty"`

	// Timeout configuration for request timeouts.
	// +optional
	Timeout *TimeoutConfig `json:"timeout,omitempty"`

	// RateLimit configuration for rate limiting (future extension).
	// +optional
	RateLimit *RateLimitConfig `json:"rateLimit,omitempty"`
}

// TargetReference identifies a Kubernetes Service.
type TargetReference struct {
	// Name of the target Service.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// Namespace of the target Service (defaults to policy namespace).
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Port name or number (optional, applies to all ports if not specified).
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port *int32 `json:"port,omitempty"`
}


// CircuitBreakerConfig configures circuit breaking behavior.
type CircuitBreakerConfig struct {
	// Enabled controls whether circuit breaking is active.
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// FailureThreshold is the number of consecutive failures before circuit opens.
	// Maps to Linkerd annotation: config.linkerd.io/failure-accrual-consecutive-failures
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=5
	FailureThreshold int32 `json:"failureThreshold"`

	// FailureAccrualMethod: "consecutive" (only supported value in Linkerd 2.16).
	// +kubebuilder:validation:Enum=consecutive
	// +kubebuilder:default=consecutive
	// +optional
	FailureAccrualMethod string `json:"failureAccrualMethod,omitempty"`
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	// Enabled controls whether retries are active.
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// MaxAttempts is the maximum number of retry attempts.
	// Maps to Linkerd annotation: retry.linkerd.io/http
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=3
	MaxAttempts int32 `json:"maxAttempts"`

	// RetryableStatusCodes defines which HTTP status codes trigger retries.
	// Example: "5xx,429"
	// +kubebuilder:validation:Pattern=`^[0-9x,]+$`
	// +optional
	RetryableStatusCodes string `json:"retryableStatusCodes,omitempty"`

	// RetryTimeout is the timeout per retry attempt.
	// Maps to Linkerd annotation: retry.linkerd.io/timeout
	// +kubebuilder:validation:Pattern=`^[0-9]+(ms|s|m)$`
	// +optional
	RetryTimeout string `json:"retryTimeout,omitempty"`
}

// TimeoutConfig configures request timeout behavior.
type TimeoutConfig struct {
	// Enabled controls whether timeouts are active.
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// RequestTimeout is the maximum time for a request to complete.
	// Maps to Linkerd annotation: timeout.linkerd.io/request
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[0-9]+(ms|s|m)$`
	RequestTimeout string `json:"requestTimeout"`

	// ResponseTimeout is the maximum time to wait for response headers.
	// Maps to Linkerd annotation: timeout.linkerd.io/response
	// +kubebuilder:validation:Pattern=`^[0-9]+(ms|s|m)$`
	// +optional
	ResponseTimeout string `json:"responseTimeout,omitempty"`
}

// RateLimitConfig configures rate limiting (placeholder for future).
type RateLimitConfig struct {
	// Enabled controls whether rate limiting is active.
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// RequestsPerSecond is the maximum requests per second.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100000
	RequestsPerSecond int32 `json:"requestsPerSecond"`

	// BurstSize is the maximum burst size.
	// +kubebuilder:validation:Minimum=1
	// +optional
	BurstSize int32 `json:"burstSize,omitempty"`
}


// ResiliencePolicyStatus defines the observed state of ResiliencePolicy.
type ResiliencePolicyStatus struct {
	// Conditions represent the latest available observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation most recently observed.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// AppliedToServices lists the services this policy is applied to.
	// +optional
	AppliedToServices []string `json:"appliedToServices,omitempty"`

	// LastUpdateTime is when the policy was last updated.
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=respol
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.targetRef.name`
// +kubebuilder:printcolumn:name="Circuit Breaker",type=boolean,JSONPath=`.spec.circuitBreaker.enabled`
// +kubebuilder:printcolumn:name="Retry",type=boolean,JSONPath=`.spec.retry.enabled`
// +kubebuilder:printcolumn:name="Timeout",type=string,JSONPath=`.spec.timeout.requestTimeout`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ResiliencePolicy is the Schema for the resiliencepolicies API.
type ResiliencePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResiliencePolicySpec   `json:"spec,omitempty"`
	Status ResiliencePolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ResiliencePolicyList contains a list of ResiliencePolicy.
type ResiliencePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResiliencePolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResiliencePolicy{}, &ResiliencePolicyList{})
}
