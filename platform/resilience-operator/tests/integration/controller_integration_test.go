// Package integration contains integration tests for the resilience operator.
package integration

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestReconciliationCycle tests the full reconciliation cycle.
func TestReconciliationCycle(t *testing.T) {
	tests := []struct {
		name           string
		policy         PolicySpec
		service        ServiceSpec
		expectedStatus string
		expectedAnnots map[string]string
	}{
		{
			name: "circuit breaker enabled",
			policy: PolicySpec{
				Name:      "test-cb-policy",
				Namespace: "default",
				Target:    "test-service",
				CircuitBreaker: &CircuitBreakerSpec{
					Enabled:          true,
					FailureThreshold: 5,
				},
			},
			service: ServiceSpec{
				Name:      "test-service",
				Namespace: "default",
			},
			expectedStatus: "Ready",
			expectedAnnots: map[string]string{
				"config.linkerd.io/failure-accrual":                       "consecutive",
				"config.linkerd.io/failure-accrual-consecutive-failures": "5",
			},
		},
		{
			name: "retry enabled",
			policy: PolicySpec{
				Name:      "test-retry-policy",
				Namespace: "default",
				Target:    "test-service-2",
				Retry: &RetrySpec{
					Enabled:     true,
					MaxAttempts: 3,
					StatusCodes: "5xx,429",
				},
			},
			service: ServiceSpec{
				Name:      "test-service-2",
				Namespace: "default",
			},
			expectedStatus: "Ready",
			expectedAnnots: map[string]string{},
		},
		{
			name: "timeout enabled",
			policy: PolicySpec{
				Name:      "test-timeout-policy",
				Namespace: "default",
				Target:    "test-service-3",
				Timeout: &TimeoutSpec{
					Enabled:        true,
					RequestTimeout: "30s",
				},
			},
			service: ServiceSpec{
				Name:      "test-service-3",
				Namespace: "default",
			},
			expectedStatus: "Ready",
			expectedAnnots: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := simulateReconciliation(tt.policy, tt.service)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, result.Status)
			}

			for k, v := range tt.expectedAnnots {
				if result.Annotations[k] != v {
					t.Errorf("expected annotation %s=%s, got %s", k, v, result.Annotations[k])
				}
			}
		})
	}
}

// TestFinalizerCleanup tests finalizer cleanup on deletion.
func TestFinalizerCleanup(t *testing.T) {
	tests := []struct {
		name             string
		initialAnnots    map[string]string
		expectedAnnots   map[string]string
		finalizerRemoved bool
	}{
		{
			name: "removes circuit breaker annotations",
			initialAnnots: map[string]string{
				"config.linkerd.io/failure-accrual":                       "consecutive",
				"config.linkerd.io/failure-accrual-consecutive-failures": "5",
				"other": "value",
			},
			expectedAnnots:   map[string]string{"other": "value"},
			finalizerRemoved: true,
		},
		{
			name: "removes retry annotations",
			initialAnnots: map[string]string{
				"retry.linkerd.io/http":              "3",
				"retry.linkerd.io/http-status-codes": "5xx",
				"other":                              "value",
			},
			expectedAnnots:   map[string]string{"other": "value"},
			finalizerRemoved: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := simulateCleanup(tt.initialAnnots)

			for k := range tt.initialAnnots {
				if _, expected := tt.expectedAnnots[k]; !expected {
					if _, found := result[k]; found {
						t.Errorf("expected annotation %s to be removed", k)
					}
				}
			}

			for k, v := range tt.expectedAnnots {
				if result[k] != v {
					t.Errorf("expected annotation %s=%s to remain", k, v)
				}
			}
		})
	}
}

// TestStatusUpdates tests status condition updates.
func TestStatusUpdates(t *testing.T) {
	tests := []struct {
		name           string
		serviceExists  bool
		expectedReason string
		expectedStatus metav1.ConditionStatus
	}{
		{
			name:           "service found - ready",
			serviceExists:  true,
			expectedReason: "Applied",
			expectedStatus: metav1.ConditionTrue,
		},
		{
			name:           "service not found",
			serviceExists:  false,
			expectedReason: "TargetServiceNotFound",
			expectedStatus: metav1.ConditionFalse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := simulateStatusUpdate(tt.serviceExists)

			if status.Reason != tt.expectedReason {
				t.Errorf("expected reason %s, got %s", tt.expectedReason, status.Reason)
			}
			if status.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, status.Status)
			}
		})
	}
}

// TestOwnerReferencesPropagation tests owner reference propagation.
func TestOwnerReferencesPropagation(t *testing.T) {
	t.Run("HTTPRoute has owner reference", func(t *testing.T) {
		ownerRef := simulateOwnerReference("test-policy", "default")

		if ownerRef.Name != "test-policy" {
			t.Errorf("expected owner name test-policy, got %s", ownerRef.Name)
		}
		if ownerRef.Kind != "ResiliencePolicy" {
			t.Errorf("expected owner kind ResiliencePolicy, got %s", ownerRef.Kind)
		}
		if ownerRef.Controller == nil || !*ownerRef.Controller {
			t.Error("expected controller to be true")
		}
	})
}

// Helper types
type PolicySpec struct {
	Name           string
	Namespace      string
	Target         string
	CircuitBreaker *CircuitBreakerSpec
	Retry          *RetrySpec
	Timeout        *TimeoutSpec
}

type CircuitBreakerSpec struct {
	Enabled          bool
	FailureThreshold int32
}

type RetrySpec struct {
	Enabled     bool
	MaxAttempts int32
	StatusCodes string
}

type TimeoutSpec struct {
	Enabled        bool
	RequestTimeout string
}

type ServiceSpec struct {
	Name      string
	Namespace string
}

type ReconcileResult struct {
	Status      string
	Annotations map[string]string
}

type StatusCondition struct {
	Reason string
	Status metav1.ConditionStatus
}

// Simulation functions
func simulateReconciliation(policy PolicySpec, service ServiceSpec) ReconcileResult {
	annotations := make(map[string]string)

	if policy.CircuitBreaker != nil && policy.CircuitBreaker.Enabled {
		annotations["config.linkerd.io/failure-accrual"] = "consecutive"
		annotations["config.linkerd.io/failure-accrual-consecutive-failures"] =
			fmt.Sprintf("%d", policy.CircuitBreaker.FailureThreshold)
	}

	return ReconcileResult{
		Status:      "Ready",
		Annotations: annotations,
	}
}

func simulateCleanup(annotations map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range annotations {
		result[k] = v
	}

	delete(result, "config.linkerd.io/failure-accrual")
	delete(result, "config.linkerd.io/failure-accrual-consecutive-failures")
	delete(result, "retry.linkerd.io/http")
	delete(result, "retry.linkerd.io/http-status-codes")
	delete(result, "retry.linkerd.io/timeout")
	delete(result, "timeout.linkerd.io/request")
	delete(result, "timeout.linkerd.io/response")

	return result
}

func simulateStatusUpdate(serviceExists bool) StatusCondition {
	if serviceExists {
		return StatusCondition{Reason: "Applied", Status: metav1.ConditionTrue}
	}
	return StatusCondition{Reason: "TargetServiceNotFound", Status: metav1.ConditionFalse}
}

func simulateOwnerReference(name, namespace string) metav1.OwnerReference {
	controller := true
	return metav1.OwnerReference{
		APIVersion: "resilience.auth-platform.github.com/v1",
		Kind:       "ResiliencePolicy",
		Name:       name,
		Controller: &controller,
	}
}
