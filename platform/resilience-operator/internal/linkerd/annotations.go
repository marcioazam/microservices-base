// Package linkerd provides Linkerd-specific annotation mapping.
package linkerd

import (
	"fmt"

	resiliencev1 "github.com/auth-platform/platform/resilience-operator/api/v1"
)

// Linkerd annotation keys
const (
	// Circuit breaker annotations
	AnnotationFailureAccrual            = "config.linkerd.io/failure-accrual"
	AnnotationFailureAccrualConsecutive = "config.linkerd.io/failure-accrual-consecutive-failures"

	// Retry annotations
	AnnotationRetryHTTP        = "retry.linkerd.io/http"
	AnnotationRetryStatusCodes = "retry.linkerd.io/http-status-codes"
	AnnotationRetryTimeout     = "retry.linkerd.io/timeout"

	// Timeout annotations
	AnnotationTimeoutRequest  = "timeout.linkerd.io/request"
	AnnotationTimeoutResponse = "timeout.linkerd.io/response"

	// Sidecar injection
	AnnotationInject = "linkerd.io/inject"
)

// AnnotationMapper maps ResiliencePolicy configs to Linkerd annotations.
type AnnotationMapper struct{}

// NewAnnotationMapper creates a new AnnotationMapper.
func NewAnnotationMapper() *AnnotationMapper {
	return &AnnotationMapper{}
}

// CircuitBreakerAnnotations returns Linkerd annotations for circuit breaker config.
func (m *AnnotationMapper) CircuitBreakerAnnotations(config *resiliencev1.CircuitBreakerConfig) map[string]string {
	if config == nil || !config.Enabled {
		return nil
	}

	return map[string]string{
		AnnotationFailureAccrual:            "consecutive",
		AnnotationFailureAccrualConsecutive: fmt.Sprintf("%d", config.FailureThreshold),
	}
}

// RetryAnnotations returns Linkerd annotations for retry config.
func (m *AnnotationMapper) RetryAnnotations(config *resiliencev1.RetryConfig) map[string]string {
	if config == nil || !config.Enabled {
		return nil
	}

	annotations := map[string]string{
		AnnotationRetryHTTP: fmt.Sprintf("%d", config.MaxAttempts),
	}

	if config.RetryableStatusCodes != "" {
		annotations[AnnotationRetryStatusCodes] = config.RetryableStatusCodes
	}

	if config.RetryTimeout != "" {
		annotations[AnnotationRetryTimeout] = config.RetryTimeout
	}

	return annotations
}

// TimeoutAnnotations returns Linkerd annotations for timeout config.
func (m *AnnotationMapper) TimeoutAnnotations(config *resiliencev1.TimeoutConfig) map[string]string {
	if config == nil || !config.Enabled {
		return nil
	}

	annotations := map[string]string{
		AnnotationTimeoutRequest: config.RequestTimeout,
	}

	if config.ResponseTimeout != "" {
		annotations[AnnotationTimeoutResponse] = config.ResponseTimeout
	}

	return annotations
}

// MergeAnnotations merges multiple annotation maps into one.
func MergeAnnotations(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// RemoveCircuitBreakerAnnotations removes circuit breaker annotations from a map.
func RemoveCircuitBreakerAnnotations(annotations map[string]string) {
	delete(annotations, AnnotationFailureAccrual)
	delete(annotations, AnnotationFailureAccrualConsecutive)
}

// RemoveRetryAnnotations removes retry annotations from a map.
func RemoveRetryAnnotations(annotations map[string]string) {
	delete(annotations, AnnotationRetryHTTP)
	delete(annotations, AnnotationRetryStatusCodes)
	delete(annotations, AnnotationRetryTimeout)
}

// RemoveTimeoutAnnotations removes timeout annotations from a map.
func RemoveTimeoutAnnotations(annotations map[string]string) {
	delete(annotations, AnnotationTimeoutRequest)
	delete(annotations, AnnotationTimeoutResponse)
}
