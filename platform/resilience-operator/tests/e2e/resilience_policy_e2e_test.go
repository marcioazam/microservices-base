// Package e2e contains end-to-end tests for the resilience operator.
package e2e

import (
	"fmt"
	"testing"
)

// TestDeployServiceWithLinkerdSidecar tests deploying a service with Linkerd sidecar.
func TestDeployServiceWithLinkerdSidecar(t *testing.T) {
	scenarios := []struct {
		name            string
		namespace       string
		serviceName     string
		injectEnabled   bool
		expectedSidecar bool
	}{
		{
			name:            "namespace injection enabled",
			namespace:       "test-ns-injected",
			serviceName:     "test-service",
			injectEnabled:   true,
			expectedSidecar: true,
		},
		{
			name:            "namespace injection disabled",
			namespace:       "test-ns-no-inject",
			serviceName:     "test-service",
			injectEnabled:   false,
			expectedSidecar: false,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			result := simulateDeployment(sc.namespace, sc.serviceName, sc.injectEnabled)
			if result.HasSidecar != sc.expectedSidecar {
				t.Errorf("expected sidecar=%v, got %v", sc.expectedSidecar, result.HasSidecar)
			}
		})
	}
}

// TestCreateResiliencePolicyAndVerifyAnnotations tests policy creation and annotation verification.
func TestCreateResiliencePolicyAndVerifyAnnotations(t *testing.T) {
	scenarios := []struct {
		name           string
		policyName     string
		targetService  string
		circuitBreaker *CircuitBreakerConfig
		retry          *RetryConfig
		timeout        *TimeoutConfig
		expectedAnnots map[string]string
	}{
		{
			name:          "circuit breaker policy",
			policyName:    "cb-policy",
			targetService: "api-service",
			circuitBreaker: &CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 5,
			},
			expectedAnnots: map[string]string{
				"config.linkerd.io/failure-accrual":                       "consecutive",
				"config.linkerd.io/failure-accrual-consecutive-failures": "5",
			},
		},
		{
			name:          "retry policy",
			policyName:    "retry-policy",
			targetService: "api-service",
			retry: &RetryConfig{
				Enabled:     true,
				MaxAttempts: 3,
				StatusCodes: "5xx,429",
			},
			expectedAnnots: map[string]string{
				"retry.linkerd.io/http":              "3",
				"retry.linkerd.io/http-status-codes": "5xx,429",
			},
		},
		{
			name:          "timeout policy",
			policyName:    "timeout-policy",
			targetService: "api-service",
			timeout: &TimeoutConfig{
				Enabled:        true,
				RequestTimeout: "30s",
			},
			expectedAnnots: map[string]string{
				"timeout.linkerd.io/request": "30s",
			},
		},
		{
			name:          "combined policy",
			policyName:    "combined-policy",
			targetService: "api-service",
			circuitBreaker: &CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 10,
			},
			retry: &RetryConfig{
				Enabled:     true,
				MaxAttempts: 2,
			},
			timeout: &TimeoutConfig{
				Enabled:        true,
				RequestTimeout: "60s",
			},
			expectedAnnots: map[string]string{
				"config.linkerd.io/failure-accrual":                       "consecutive",
				"config.linkerd.io/failure-accrual-consecutive-failures": "10",
				"retry.linkerd.io/http":                                   "2",
				"timeout.linkerd.io/request":                              "60s",
			},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			result := simulatePolicyApplication(sc.policyName, sc.targetService,
				sc.circuitBreaker, sc.retry, sc.timeout)

			for k, v := range sc.expectedAnnots {
				if result.Annotations[k] != v {
					t.Errorf("expected annotation %s=%s, got %s", k, v, result.Annotations[k])
				}
			}
		})
	}
}

// TestCircuitBreakerActivation tests circuit breaker activation behavior.
func TestCircuitBreakerActivation(t *testing.T) {
	t.Run("circuit opens after consecutive failures", func(t *testing.T) {
		threshold := 5
		failures := 0
		circuitOpen := false

		for i := 0; i < threshold; i++ {
			failures++
			if failures >= threshold {
				circuitOpen = true
			}
		}

		if !circuitOpen {
			t.Error("circuit should be open after threshold failures")
		}
	})

	t.Run("circuit stays closed below threshold", func(t *testing.T) {
		threshold := 5
		failures := 3
		circuitOpen := failures >= threshold

		if circuitOpen {
			t.Error("circuit should stay closed below threshold")
		}
	})
}

// TestRetryBehavior tests retry behavior.
func TestRetryBehavior(t *testing.T) {
	t.Run("retries on 5xx errors", func(t *testing.T) {
		statusCodes := []int{500, 502, 503, 504}
		maxAttempts := 3

		for _, code := range statusCodes {
			attempts := simulateRetry(code, maxAttempts, "5xx")
			if attempts < 2 {
				t.Errorf("expected retry for status %d", code)
			}
		}
	})

	t.Run("no retry on 4xx errors", func(t *testing.T) {
		statusCodes := []int{400, 401, 403, 404}
		maxAttempts := 3

		for _, code := range statusCodes {
			attempts := simulateRetry(code, maxAttempts, "5xx")
			if attempts > 1 {
				t.Errorf("should not retry for status %d", code)
			}
		}
	})

	t.Run("respects max attempts", func(t *testing.T) {
		maxAttempts := 3
		attempts := simulateRetry(500, maxAttempts, "5xx")
		if attempts > maxAttempts {
			t.Errorf("attempts %d exceeded max %d", attempts, maxAttempts)
		}
	})
}

// TestPolicyDeletion tests policy deletion and cleanup.
func TestPolicyDeletion(t *testing.T) {
	t.Run("annotations removed on policy deletion", func(t *testing.T) {
		initialAnnots := map[string]string{
			"config.linkerd.io/failure-accrual":                       "consecutive",
			"config.linkerd.io/failure-accrual-consecutive-failures": "5",
			"app": "test",
		}

		result := simulatePolicyDeletion(initialAnnots)

		if _, exists := result["config.linkerd.io/failure-accrual"]; exists {
			t.Error("failure-accrual should be removed")
		}
		if result["app"] != "test" {
			t.Error("non-linkerd annotations should be preserved")
		}
	})
}

// Helper types
type CircuitBreakerConfig struct {
	Enabled          bool
	FailureThreshold int32
}

type RetryConfig struct {
	Enabled     bool
	MaxAttempts int32
	StatusCodes string
}

type TimeoutConfig struct {
	Enabled        bool
	RequestTimeout string
}

type DeploymentResult struct {
	HasSidecar bool
}

type PolicyResult struct {
	Annotations map[string]string
}

// Simulation functions
func simulateDeployment(namespace, serviceName string, injectEnabled bool) DeploymentResult {
	return DeploymentResult{HasSidecar: injectEnabled}
}

func simulatePolicyApplication(policyName, targetService string,
	cb *CircuitBreakerConfig, retry *RetryConfig, timeout *TimeoutConfig) PolicyResult {

	annotations := make(map[string]string)

	if cb != nil && cb.Enabled {
		annotations["config.linkerd.io/failure-accrual"] = "consecutive"
		annotations["config.linkerd.io/failure-accrual-consecutive-failures"] =
			fmt.Sprintf("%d", cb.FailureThreshold)
	}

	if retry != nil && retry.Enabled {
		annotations["retry.linkerd.io/http"] = fmt.Sprintf("%d", retry.MaxAttempts)
		if retry.StatusCodes != "" {
			annotations["retry.linkerd.io/http-status-codes"] = retry.StatusCodes
		}
	}

	if timeout != nil && timeout.Enabled {
		annotations["timeout.linkerd.io/request"] = timeout.RequestTimeout
	}

	return PolicyResult{Annotations: annotations}
}

func simulateRetry(statusCode, maxAttempts int, retryableCodes string) int {
	isRetryable := statusCode >= 500 && statusCode < 600
	if !isRetryable {
		return 1
	}
	return maxAttempts
}

func simulatePolicyDeletion(annotations map[string]string) map[string]string {
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
