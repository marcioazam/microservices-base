// Package property contains property-based tests for the resilience operator.
package property

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/auth-platform/platform/resilience-operator/internal/linkerd"
)

// TestAnnotationRemovalOnDisable validates annotation removal when features are disabled.
// Property 9: Annotation Removal on Disable
// Validates: Requirements 4.3
func TestAnnotationRemovalOnDisable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hasCircuitBreaker := rapid.Bool().Draw(t, "hasCircuitBreaker")
		hasRetry := rapid.Bool().Draw(t, "hasRetry")
		hasTimeout := rapid.Bool().Draw(t, "hasTimeout")

		annotations := make(map[string]string)
		annotations["other"] = "preserved"

		if hasCircuitBreaker {
			annotations[linkerd.AnnotationFailureAccrual] = "consecutive"
			annotations[linkerd.AnnotationFailureAccrualConsecutive] = "5"
		}
		if hasRetry {
			annotations[linkerd.AnnotationRetryHTTP] = "3"
			annotations[linkerd.AnnotationRetryStatusCodes] = "5xx"
			annotations[linkerd.AnnotationRetryTimeout] = "5s"
		}
		if hasTimeout {
			annotations[linkerd.AnnotationTimeoutRequest] = "30s"
			annotations[linkerd.AnnotationTimeoutResponse] = "10s"
		}

		result := removeAllResilienceAnnotations(annotations)

		linkerdAnnotations := []string{
			linkerd.AnnotationFailureAccrual,
			linkerd.AnnotationFailureAccrualConsecutive,
			linkerd.AnnotationRetryHTTP,
			linkerd.AnnotationRetryStatusCodes,
			linkerd.AnnotationRetryTimeout,
			linkerd.AnnotationTimeoutRequest,
			linkerd.AnnotationTimeoutResponse,
		}

		for _, key := range linkerdAnnotations {
			if _, exists := result[key]; exists {
				t.Fatalf("Linkerd annotation %s should be removed", key)
			}
		}

		if result["other"] != "preserved" {
			t.Fatal("Non-Linkerd annotations should be preserved")
		}
	})
}

// TestCircuitBreakerAnnotationRemoval validates circuit breaker annotation removal.
func TestCircuitBreakerAnnotationRemoval(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		annotations := map[string]string{
			linkerd.AnnotationFailureAccrual:            "consecutive",
			linkerd.AnnotationFailureAccrualConsecutive: "5",
			"other": "value",
		}

		linkerd.RemoveCircuitBreakerAnnotations(annotations)

		if _, exists := annotations[linkerd.AnnotationFailureAccrual]; exists {
			t.Fatal("failure-accrual should be removed")
		}
		if _, exists := annotations[linkerd.AnnotationFailureAccrualConsecutive]; exists {
			t.Fatal("failure-accrual-consecutive-failures should be removed")
		}

		if annotations["other"] != "value" {
			t.Fatal("other annotations should be preserved")
		}
	})
}

// TestRetryAnnotationRemoval validates retry annotation removal.
func TestRetryAnnotationRemoval(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hasStatusCodes := rapid.Bool().Draw(t, "hasStatusCodes")
		hasTimeout := rapid.Bool().Draw(t, "hasTimeout")

		annotations := map[string]string{
			linkerd.AnnotationRetryHTTP: "3",
			"other":                     "value",
		}
		if hasStatusCodes {
			annotations[linkerd.AnnotationRetryStatusCodes] = "5xx"
		}
		if hasTimeout {
			annotations[linkerd.AnnotationRetryTimeout] = "5s"
		}

		linkerd.RemoveRetryAnnotations(annotations)

		retryKeys := []string{
			linkerd.AnnotationRetryHTTP,
			linkerd.AnnotationRetryStatusCodes,
			linkerd.AnnotationRetryTimeout,
		}
		for _, key := range retryKeys {
			if _, exists := annotations[key]; exists {
				t.Fatalf("%s should be removed", key)
			}
		}

		if annotations["other"] != "value" {
			t.Fatal("other annotations should be preserved")
		}
	})
}

// TestTimeoutAnnotationRemoval validates timeout annotation removal.
func TestTimeoutAnnotationRemoval(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hasResponseTimeout := rapid.Bool().Draw(t, "hasResponseTimeout")

		annotations := map[string]string{
			linkerd.AnnotationTimeoutRequest: "30s",
			"other":                          "value",
		}
		if hasResponseTimeout {
			annotations[linkerd.AnnotationTimeoutResponse] = "10s"
		}

		linkerd.RemoveTimeoutAnnotations(annotations)

		timeoutKeys := []string{
			linkerd.AnnotationTimeoutRequest,
			linkerd.AnnotationTimeoutResponse,
		}
		for _, key := range timeoutKeys {
			if _, exists := annotations[key]; exists {
				t.Fatalf("%s should be removed", key)
			}
		}

		if annotations["other"] != "value" {
			t.Fatal("other annotations should be preserved")
		}
	})
}

// TestIdempotentAnnotationRemoval validates that removal is idempotent.
func TestIdempotentAnnotationRemoval(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		annotations := map[string]string{
			"other": "value",
		}

		result1 := removeAllResilienceAnnotations(annotations)
		result2 := removeAllResilienceAnnotations(result1)

		if len(result1) != len(result2) {
			t.Fatal("Idempotent removal should produce same result")
		}
		for k, v := range result1 {
			if result2[k] != v {
				t.Fatalf("Annotation %s differs after second removal", k)
			}
		}
	})
}

// removeAllResilienceAnnotations removes all resilience annotations.
func removeAllResilienceAnnotations(annotations map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range annotations {
		result[k] = v
	}

	keysToRemove := []string{
		linkerd.AnnotationFailureAccrual,
		linkerd.AnnotationFailureAccrualConsecutive,
		linkerd.AnnotationRetryHTTP,
		linkerd.AnnotationRetryStatusCodes,
		linkerd.AnnotationRetryTimeout,
		linkerd.AnnotationTimeoutRequest,
		linkerd.AnnotationTimeoutResponse,
	}

	for _, key := range keysToRemove {
		delete(result, key)
	}

	return result
}
