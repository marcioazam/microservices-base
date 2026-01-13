// Package unit contains unit tests for the resilience operator.
package unit

import (
	"fmt"
	"testing"
)

// MockCircuitBreaker for testing.
type MockCircuitBreaker struct {
	Enabled          bool
	FailureThreshold int32
}

// MockRetry for testing.
type MockRetry struct {
	Enabled     bool
	MaxAttempts int32
	StatusCodes string
	Timeout     string
}

// MockTimeout for testing.
type MockTimeout struct {
	Enabled         bool
	RequestTimeout  string
	ResponseTimeout string
}

func TestAnnotationMapper_CircuitBreakerAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		config   *MockCircuitBreaker
		expected map[string]string
	}{
		{name: "nil config returns nil", config: nil, expected: nil},
		{name: "disabled config returns nil", config: &MockCircuitBreaker{Enabled: false, FailureThreshold: 5}, expected: nil},
		{
			name:   "enabled config returns annotations",
			config: &MockCircuitBreaker{Enabled: true, FailureThreshold: 5},
			expected: map[string]string{
				"config.linkerd.io/failure-accrual":                       "consecutive",
				"config.linkerd.io/failure-accrual-consecutive-failures": "5",
			},
		},
		{
			name:   "custom threshold",
			config: &MockCircuitBreaker{Enabled: true, FailureThreshold: 10},
			expected: map[string]string{
				"config.linkerd.io/failure-accrual":                       "consecutive",
				"config.linkerd.io/failure-accrual-consecutive-failures": "10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := circuitBreakerAnnotations(tt.config)
			assertAnnotations(t, tt.expected, result)
		})
	}
}

func TestAnnotationMapper_RetryAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		config   *MockRetry
		expected map[string]string
	}{
		{name: "nil config returns nil", config: nil, expected: nil},
		{name: "disabled config returns nil", config: &MockRetry{Enabled: false, MaxAttempts: 3}, expected: nil},
		{
			name:     "enabled config returns annotations",
			config:   &MockRetry{Enabled: true, MaxAttempts: 3},
			expected: map[string]string{"retry.linkerd.io/http": "3"},
		},
		{
			name:   "with status codes",
			config: &MockRetry{Enabled: true, MaxAttempts: 3, StatusCodes: "5xx,429"},
			expected: map[string]string{
				"retry.linkerd.io/http":              "3",
				"retry.linkerd.io/http-status-codes": "5xx,429",
			},
		},
		{
			name:   "with timeout",
			config: &MockRetry{Enabled: true, MaxAttempts: 3, Timeout: "5s"},
			expected: map[string]string{
				"retry.linkerd.io/http":    "3",
				"retry.linkerd.io/timeout": "5s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := retryAnnotations(tt.config)
			assertAnnotations(t, tt.expected, result)
		})
	}
}

func TestAnnotationMapper_TimeoutAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		config   *MockTimeout
		expected map[string]string
	}{
		{name: "nil config returns nil", config: nil, expected: nil},
		{name: "disabled config returns nil", config: &MockTimeout{Enabled: false, RequestTimeout: "30s"}, expected: nil},
		{
			name:     "enabled config returns annotations",
			config:   &MockTimeout{Enabled: true, RequestTimeout: "30s"},
			expected: map[string]string{"timeout.linkerd.io/request": "30s"},
		},
		{
			name:   "with response timeout",
			config: &MockTimeout{Enabled: true, RequestTimeout: "30s", ResponseTimeout: "10s"},
			expected: map[string]string{
				"timeout.linkerd.io/request":  "30s",
				"timeout.linkerd.io/response": "10s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timeoutAnnotations(tt.config)
			assertAnnotations(t, tt.expected, result)
		})
	}
}

func TestMergeAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		maps     []map[string]string
		expected map[string]string
	}{
		{name: "empty maps", maps: []map[string]string{}, expected: map[string]string{}},
		{name: "single map", maps: []map[string]string{{"a": "1", "b": "2"}}, expected: map[string]string{"a": "1", "b": "2"}},
		{name: "multiple maps", maps: []map[string]string{{"a": "1"}, {"b": "2"}, {"c": "3"}}, expected: map[string]string{"a": "1", "b": "2", "c": "3"}},
		{name: "overlapping keys - last wins", maps: []map[string]string{{"a": "1"}, {"a": "2"}}, expected: map[string]string{"a": "2"}},
		{name: "nil maps ignored", maps: []map[string]string{{"a": "1"}, nil, {"b": "2"}}, expected: map[string]string{"a": "1", "b": "2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeAnnotations(tt.maps...)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d entries, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("expected %s=%s, got %s=%s", k, v, k, result[k])
				}
			}
		})
	}
}

func TestRemoveAnnotations(t *testing.T) {
	t.Run("remove circuit breaker annotations", func(t *testing.T) {
		annotations := map[string]string{
			"config.linkerd.io/failure-accrual":                       "consecutive",
			"config.linkerd.io/failure-accrual-consecutive-failures": "5",
			"other": "value",
		}
		removeCircuitBreakerAnnotations(annotations)
		if _, ok := annotations["config.linkerd.io/failure-accrual"]; ok {
			t.Error("expected failure-accrual to be removed")
		}
		if annotations["other"] != "value" {
			t.Error("expected other annotation to remain")
		}
	})

	t.Run("remove retry annotations", func(t *testing.T) {
		annotations := map[string]string{
			"retry.linkerd.io/http": "3",
			"other":                 "value",
		}
		removeRetryAnnotations(annotations)
		if _, ok := annotations["retry.linkerd.io/http"]; ok {
			t.Error("expected retry http to be removed")
		}
		if annotations["other"] != "value" {
			t.Error("expected other annotation to remain")
		}
	})

	t.Run("remove timeout annotations", func(t *testing.T) {
		annotations := map[string]string{
			"timeout.linkerd.io/request": "30s",
			"other":                       "value",
		}
		removeTimeoutAnnotations(annotations)
		if _, ok := annotations["timeout.linkerd.io/request"]; ok {
			t.Error("expected timeout request to be removed")
		}
		if annotations["other"] != "value" {
			t.Error("expected other annotation to remain")
		}
	})
}

// Helper functions
func assertAnnotations(t *testing.T, expected, result map[string]string) {
	t.Helper()
	if expected == nil {
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
		return
	}
	for k, v := range expected {
		if result[k] != v {
			t.Errorf("expected %s=%s, got %s=%s", k, v, k, result[k])
		}
	}
}

func circuitBreakerAnnotations(config *MockCircuitBreaker) map[string]string {
	if config == nil || !config.Enabled {
		return nil
	}
	return map[string]string{
		"config.linkerd.io/failure-accrual":                       "consecutive",
		"config.linkerd.io/failure-accrual-consecutive-failures": fmt.Sprintf("%d", config.FailureThreshold),
	}
}

func retryAnnotations(config *MockRetry) map[string]string {
	if config == nil || !config.Enabled {
		return nil
	}
	annotations := map[string]string{"retry.linkerd.io/http": fmt.Sprintf("%d", config.MaxAttempts)}
	if config.StatusCodes != "" {
		annotations["retry.linkerd.io/http-status-codes"] = config.StatusCodes
	}
	if config.Timeout != "" {
		annotations["retry.linkerd.io/timeout"] = config.Timeout
	}
	return annotations
}

func timeoutAnnotations(config *MockTimeout) map[string]string {
	if config == nil || !config.Enabled {
		return nil
	}
	annotations := map[string]string{"timeout.linkerd.io/request": config.RequestTimeout}
	if config.ResponseTimeout != "" {
		annotations["timeout.linkerd.io/response"] = config.ResponseTimeout
	}
	return annotations
}

func mergeAnnotations(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

func removeCircuitBreakerAnnotations(annotations map[string]string) {
	delete(annotations, "config.linkerd.io/failure-accrual")
	delete(annotations, "config.linkerd.io/failure-accrual-consecutive-failures")
}

func removeRetryAnnotations(annotations map[string]string) {
	delete(annotations, "retry.linkerd.io/http")
	delete(annotations, "retry.linkerd.io/http-status-codes")
	delete(annotations, "retry.linkerd.io/timeout")
}

func removeTimeoutAnnotations(annotations map[string]string) {
	delete(annotations, "timeout.linkerd.io/request")
	delete(annotations, "timeout.linkerd.io/response")
}
