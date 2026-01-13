// Package property contains property-based tests for the resilience operator.
package property

import (
	"testing"

	"pgregory.net/rapid"

	resiliencev1 "github.com/auth-platform/platform/resilience-operator/api/v1"
	"github.com/auth-platform/platform/resilience-operator/internal/linkerd"
)

// TestReconciliationIdempotency validates reconciliation produces identical results.
// Property 1: Reconciliation Idempotency
// Validates: Requirements 4.3
func TestReconciliationIdempotency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policy := generateResiliencePolicy(t)
		mapper := linkerd.NewAnnotationMapper()

		cbAnnotations1 := mapper.CircuitBreakerAnnotations(policy.Spec.CircuitBreaker)
		retryAnnotations1 := mapper.RetryAnnotations(policy.Spec.Retry)
		timeoutAnnotations1 := mapper.TimeoutAnnotations(policy.Spec.Timeout)

		cbAnnotations2 := mapper.CircuitBreakerAnnotations(policy.Spec.CircuitBreaker)
		retryAnnotations2 := mapper.RetryAnnotations(policy.Spec.Retry)
		timeoutAnnotations2 := mapper.TimeoutAnnotations(policy.Spec.Timeout)

		if !mapsEqual(cbAnnotations1, cbAnnotations2) {
			t.Fatalf("circuit breaker annotations not idempotent: %v != %v", cbAnnotations1, cbAnnotations2)
		}
		if !mapsEqual(retryAnnotations1, retryAnnotations2) {
			t.Fatalf("retry annotations not idempotent: %v != %v", retryAnnotations1, retryAnnotations2)
		}
		if !mapsEqual(timeoutAnnotations1, timeoutAnnotations2) {
			t.Fatalf("timeout annotations not idempotent: %v != %v", timeoutAnnotations1, timeoutAnnotations2)
		}
	})
}

// TestCircuitBreakerAnnotationConsistency validates circuit breaker annotations.
// Property 3: Circuit Breaker Annotation Consistency
// Validates: Requirements 5.1, 5.2
func TestCircuitBreakerAnnotationConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		enabled := rapid.Bool().Draw(t, "enabled")
		threshold := rapid.Int32Range(1, 100).Draw(t, "threshold")

		config := &resiliencev1.CircuitBreakerConfig{
			Enabled:          enabled,
			FailureThreshold: threshold,
		}

		mapper := linkerd.NewAnnotationMapper()
		annotations := mapper.CircuitBreakerAnnotations(config)

		if enabled {
			if len(annotations) != 2 {
				t.Fatalf("expected 2 annotations, got %d", len(annotations))
			}
			if annotations[linkerd.AnnotationFailureAccrual] != "consecutive" {
				t.Fatalf("expected failure-accrual=consecutive, got %s", annotations[linkerd.AnnotationFailureAccrual])
			}
		} else {
			if annotations != nil && len(annotations) > 0 {
				t.Fatalf("expected no annotations when disabled, got %v", annotations)
			}
		}
	})
}

// TestRetryConfigurationMapping validates retry configuration mapping.
// Property 4: Retry Configuration Mapping
// Validates: Requirements 6.1, 6.2, 6.3, 6.4
func TestRetryConfigurationMapping(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		enabled := rapid.Bool().Draw(t, "enabled")
		maxAttempts := rapid.Int32Range(1, 10).Draw(t, "maxAttempts")
		statusCodes := rapid.SampledFrom([]string{"", "5xx", "5xx,429", "500,502,503"}).Draw(t, "statusCodes")
		timeout := rapid.SampledFrom([]string{"", "1s", "500ms", "2m"}).Draw(t, "timeout")

		config := &resiliencev1.RetryConfig{
			Enabled:              enabled,
			MaxAttempts:          maxAttempts,
			RetryableStatusCodes: statusCodes,
			RetryTimeout:         timeout,
		}

		mapper := linkerd.NewAnnotationMapper()
		annotations := mapper.RetryAnnotations(config)

		if enabled {
			if annotations[linkerd.AnnotationRetryHTTP] == "" {
				t.Fatalf("retry annotation missing")
			}
			if statusCodes != "" && annotations[linkerd.AnnotationRetryStatusCodes] != statusCodes {
				t.Fatalf("status codes mismatch: expected %s, got %s", statusCodes, annotations[linkerd.AnnotationRetryStatusCodes])
			}
			if timeout != "" && annotations[linkerd.AnnotationRetryTimeout] != timeout {
				t.Fatalf("timeout mismatch: expected %s, got %s", timeout, annotations[linkerd.AnnotationRetryTimeout])
			}
		} else {
			if annotations != nil && len(annotations) > 0 {
				t.Fatalf("expected no annotations when disabled")
			}
		}
	})
}

// TestTimeoutConfigurationMapping validates timeout configuration mapping.
// Property 5: Timeout Configuration Mapping
// Validates: Requirements 7.1, 7.2, 7.3
func TestTimeoutConfigurationMapping(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		enabled := rapid.Bool().Draw(t, "enabled")
		requestTimeout := rapid.SampledFrom([]string{"1s", "5s", "10s", "30s", "1m"}).Draw(t, "requestTimeout")
		responseTimeout := rapid.SampledFrom([]string{"", "500ms", "2s", "5s"}).Draw(t, "responseTimeout")

		config := &resiliencev1.TimeoutConfig{
			Enabled:         enabled,
			RequestTimeout:  requestTimeout,
			ResponseTimeout: responseTimeout,
		}

		mapper := linkerd.NewAnnotationMapper()
		annotations := mapper.TimeoutAnnotations(config)

		if enabled {
			if annotations[linkerd.AnnotationTimeoutRequest] != requestTimeout {
				t.Fatalf("request timeout mismatch: expected %s, got %s", requestTimeout, annotations[linkerd.AnnotationTimeoutRequest])
			}
			if responseTimeout != "" && annotations[linkerd.AnnotationTimeoutResponse] != responseTimeout {
				t.Fatalf("response timeout mismatch: expected %s, got %s", responseTimeout, annotations[linkerd.AnnotationTimeoutResponse])
			}
		} else {
			if annotations != nil && len(annotations) > 0 {
				t.Fatalf("expected no annotations when disabled")
			}
		}
	})
}


// Helper functions

func generateResiliencePolicy(t *rapid.T) *resiliencev1.ResiliencePolicy {
	return &resiliencev1.ResiliencePolicy{
		Spec: resiliencev1.ResiliencePolicySpec{
			TargetRef: resiliencev1.TargetReference{
				Name:      rapid.StringMatching(`[a-z][a-z0-9-]{0,62}`).Draw(t, "serviceName"),
				Namespace: rapid.SampledFrom([]string{"", "default", "production"}).Draw(t, "namespace"),
			},
			CircuitBreaker: generateCircuitBreakerConfig(t),
			Retry:          generateRetryConfig(t),
			Timeout:        generateTimeoutConfig(t),
		},
	}
}

func generateCircuitBreakerConfig(t *rapid.T) *resiliencev1.CircuitBreakerConfig {
	if !rapid.Bool().Draw(t, "hasCircuitBreaker") {
		return nil
	}
	return &resiliencev1.CircuitBreakerConfig{
		Enabled:          rapid.Bool().Draw(t, "cbEnabled"),
		FailureThreshold: rapid.Int32Range(1, 100).Draw(t, "cbThreshold"),
	}
}

func generateRetryConfig(t *rapid.T) *resiliencev1.RetryConfig {
	if !rapid.Bool().Draw(t, "hasRetry") {
		return nil
	}
	return &resiliencev1.RetryConfig{
		Enabled:              rapid.Bool().Draw(t, "retryEnabled"),
		MaxAttempts:          rapid.Int32Range(1, 10).Draw(t, "retryAttempts"),
		RetryableStatusCodes: rapid.SampledFrom([]string{"", "5xx", "5xx,429"}).Draw(t, "retryCodes"),
		RetryTimeout:         rapid.SampledFrom([]string{"", "1s", "500ms"}).Draw(t, "retryTimeout"),
	}
}

func generateTimeoutConfig(t *rapid.T) *resiliencev1.TimeoutConfig {
	if !rapid.Bool().Draw(t, "hasTimeout") {
		return nil
	}
	return &resiliencev1.TimeoutConfig{
		Enabled:         rapid.Bool().Draw(t, "timeoutEnabled"),
		RequestTimeout:  rapid.SampledFrom([]string{"1s", "5s", "10s"}).Draw(t, "reqTimeout"),
		ResponseTimeout: rapid.SampledFrom([]string{"", "500ms", "2s"}).Draw(t, "respTimeout"),
	}
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
