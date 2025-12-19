package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"gopkg.in/yaml.v3"
)

// **Feature: resilience-lib-extraction, Property 6: Domain Type JSON Round-Trip**
// **Validates: Requirements 3.8, 3.9, 9.1, 9.2, 9.3**
func TestDomainTypeJSONRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// CircuitBreakerConfig round-trip
	properties.Property("CircuitBreakerConfig JSON round-trip", prop.ForAll(
		func(name string, failThresh, succThresh int, timeoutMs int64) bool {
			if failThresh < 0 || succThresh < 0 || timeoutMs < 0 {
				return true // Skip invalid values
			}
			original := CircuitBreakerConfig{
				Name:             name,
				FailureThreshold: failThresh,
				SuccessThreshold: succThresh,
				Timeout:          time.Duration(timeoutMs) * time.Millisecond,
			}
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}
			var restored CircuitBreakerConfig
			if err := json.Unmarshal(data, &restored); err != nil {
				return false
			}
			return original.Name == restored.Name &&
				original.FailureThreshold == restored.FailureThreshold &&
				original.SuccessThreshold == restored.SuccessThreshold &&
				original.Timeout == restored.Timeout
		},
		gen.AnyString(),
		gen.IntRange(0, 100),
		gen.IntRange(0, 100),
		gen.Int64Range(0, 60000),
	))

	// RetryConfig round-trip
	properties.Property("RetryConfig JSON round-trip", prop.ForAll(
		func(maxAttempts int, multiplier float64) bool {
			if maxAttempts < 0 || multiplier < 0 {
				return true
			}
			original := RetryConfig{
				MaxAttempts:  maxAttempts,
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     10 * time.Second,
				Multiplier:   multiplier,
				Jitter:       0.1,
			}
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}
			var restored RetryConfig
			if err := json.Unmarshal(data, &restored); err != nil {
				return false
			}
			return original.MaxAttempts == restored.MaxAttempts &&
				original.Multiplier == restored.Multiplier
		},
		gen.IntRange(0, 10),
		gen.Float64Range(1.0, 5.0),
	))

	// ResiliencePolicy round-trip
	properties.Property("ResiliencePolicy JSON round-trip", prop.ForAll(
		func(name string, version int) bool {
			if version < 0 {
				return true
			}
			original := ResiliencePolicy{
				Name:    name,
				Version: version,
			}
			data, err := json.Marshal(original)
			if err != nil {
				return false
			}
			var restored ResiliencePolicy
			if err := json.Unmarshal(data, &restored); err != nil {
				return false
			}
			return original.Name == restored.Name &&
				original.Version == restored.Version
		},
		gen.AnyString(),
		gen.IntRange(0, 100),
	))

	properties.TestingRun(t)
}

func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("CircuitState(%d).String() = %s, want %s", tt.state, got, tt.expected)
		}
	}
}

func TestHealthStatusString(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{Healthy, "healthy"},
		{Degraded, "degraded"},
		{Unhealthy, "unhealthy"},
		{HealthStatus(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("HealthStatus(%d).String() = %s, want %s", tt.status, got, tt.expected)
		}
	}
}

func TestCircuitStateJSONMarshal(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitClosed, `"closed"`},
		{CircuitOpen, `"open"`},
		{CircuitHalfOpen, `"half-open"`},
	}

	for _, tt := range tests {
		data, err := json.Marshal(tt.state)
		if err != nil {
			t.Errorf("failed to marshal %v: %v", tt.state, err)
			continue
		}
		if string(data) != tt.expected {
			t.Errorf("Marshal(%v) = %s, want %s", tt.state, data, tt.expected)
		}
	}
}

func TestCircuitStateJSONUnmarshal(t *testing.T) {
	tests := []struct {
		input    string
		expected CircuitState
	}{
		{`"closed"`, CircuitClosed},
		{`"open"`, CircuitOpen},
		{`"half-open"`, CircuitHalfOpen},
	}

	for _, tt := range tests {
		var state CircuitState
		if err := json.Unmarshal([]byte(tt.input), &state); err != nil {
			t.Errorf("failed to unmarshal %s: %v", tt.input, err)
			continue
		}
		if state != tt.expected {
			t.Errorf("Unmarshal(%s) = %v, want %v", tt.input, state, tt.expected)
		}
	}
}

func TestResiliencePolicyYAMLRoundTrip(t *testing.T) {
	original := ResiliencePolicy{
		Name:    "test-policy",
		Version: 1,
		CircuitBreaker: &CircuitBreakerConfig{
			Name:             "cb",
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
		},
		Retry: &RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     10 * time.Second,
			Multiplier:   2.0,
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var restored ResiliencePolicy
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if original.Name != restored.Name {
		t.Errorf("Name mismatch: %s != %s", original.Name, restored.Name)
	}
	if original.Version != restored.Version {
		t.Errorf("Version mismatch: %d != %d", original.Version, restored.Version)
	}
}

func TestDefaultConfigs(t *testing.T) {
	t.Run("DefaultCircuitBreakerConfig", func(t *testing.T) {
		cfg := DefaultCircuitBreakerConfig()
		if cfg.FailureThreshold <= 0 {
			t.Error("expected positive failure threshold")
		}
	})

	t.Run("DefaultRateLimitConfig", func(t *testing.T) {
		cfg := DefaultRateLimitConfig()
		if cfg.Limit <= 0 {
			t.Error("expected positive limit")
		}
	})

	t.Run("DefaultRetryConfig", func(t *testing.T) {
		cfg := DefaultRetryConfig()
		if cfg.MaxAttempts <= 0 {
			t.Error("expected positive max attempts")
		}
	})

	t.Run("DefaultTimeoutConfig", func(t *testing.T) {
		cfg := DefaultTimeoutConfig()
		if cfg.Default <= 0 {
			t.Error("expected positive default timeout")
		}
	})

	t.Run("DefaultBulkheadConfig", func(t *testing.T) {
		cfg := DefaultBulkheadConfig()
		if cfg.MaxConcurrent <= 0 {
			t.Error("expected positive max concurrent")
		}
	})
}
