package property

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/libs/go/resilience/circuitbreaker"
	"github.com/auth-platform/platform/resilience-service/tests/testutil"
	"gopkg.in/yaml.v3"
	"pgregory.net/rapid"
)

// **Feature: resilience-microservice, Property 2: Circuit Breaker State Serialization Round-Trip**
// **Validates: Requirements 1.6**
func TestProperty_CircuitBreakerStateSerializationRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		serviceNameLen := rapid.IntRange(1, 49).Draw(t, "serviceNameLen")
		stateInt := rapid.IntRange(0, 2).Draw(t, "stateInt")
		failureCount := rapid.IntRange(0, 100).Draw(t, "failureCount")
		successCount := rapid.IntRange(0, 100).Draw(t, "successCount")
		hasLastFailure := rapid.Bool().Draw(t, "hasLastFailure")
		version := rapid.Int64Range(1, 100).Draw(t, "version")

		serviceName := testutil.GenerateAlphaString(serviceNameLen)

		original := resilience.CircuitBreakerState{
			ServiceName:     serviceName,
			State:           resilience.CircuitState(stateInt),
			FailureCount:    failureCount,
			SuccessCount:    successCount,
			LastStateChange: time.Now().Truncate(time.Nanosecond),
			Version:         version,
		}

		if hasLastFailure {
			t := time.Now().Add(-time.Hour).Truncate(time.Nanosecond)
			original.LastFailureTime = &t
		}

		data, err := circuitbreaker.MarshalState(original)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		restored, err := circuitbreaker.UnmarshalState(data)
		if err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if original.ServiceName != restored.ServiceName {
			t.Fatalf("service name mismatch: %s != %s", original.ServiceName, restored.ServiceName)
		}
		if original.State != restored.State {
			t.Fatalf("state mismatch: %v != %v", original.State, restored.State)
		}
		if original.FailureCount != restored.FailureCount {
			t.Fatalf("failure count mismatch: %d != %d", original.FailureCount, restored.FailureCount)
		}
		if original.SuccessCount != restored.SuccessCount {
			t.Fatalf("success count mismatch: %d != %d", original.SuccessCount, restored.SuccessCount)
		}
		if original.Version != restored.Version {
			t.Fatalf("version mismatch: %d != %d", original.Version, restored.Version)
		}
		if !original.LastStateChange.Equal(restored.LastStateChange) {
			t.Fatalf("last state change mismatch")
		}

		if original.LastFailureTime == nil && restored.LastFailureTime != nil {
			t.Fatal("last failure time should be nil")
		}
		if original.LastFailureTime != nil && restored.LastFailureTime == nil {
			t.Fatal("last failure time should not be nil")
		}
		if original.LastFailureTime != nil && restored.LastFailureTime != nil {
			if !original.LastFailureTime.Equal(*restored.LastFailureTime) {
				t.Fatal("last failure time mismatch")
			}
		}
	})
}

// **Feature: platform-resilience-modernization, Property 6: Time Format Consistency**
// **Validates: Requirements 7.5**
func TestProperty_TimeFormatRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		unixNano := rapid.Int64Range(0, time.Now().UnixNano()).Draw(t, "unixNano")

		original := time.Unix(0, unixNano).UTC()
		marshaled := resilience.MarshalTime(original)
		restored, err := resilience.UnmarshalTime(marshaled)
		if err != nil {
			t.Fatalf("unmarshal time failed: %v", err)
		}
		if !original.Equal(restored) {
			t.Fatalf("time mismatch: %v != %v", original, restored)
		}
	})
}

// **Feature: platform-resilience-modernization, Property 5: Serialization Round-Trip Consistency (Policy)**
// **Validates: Requirements 7.2**
func TestProperty_ResiliencePolicySerializationRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		nameLen := rapid.IntRange(1, 30).Draw(t, "nameLen")
		version := rapid.Int64Range(1, 100).Draw(t, "version")

		policy := resilience.ResiliencePolicy{
			Name:    testutil.GenerateAlphaString(nameLen),
			Version: version,
		}

		data, err := json.Marshal(policy)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var restored resilience.ResiliencePolicy
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if policy.Name != restored.Name {
			t.Fatalf("name mismatch: %s != %s", policy.Name, restored.Name)
		}
		if policy.Version != restored.Version {
			t.Fatalf("version mismatch: %d != %d", policy.Version, restored.Version)
		}
	})
}

// **Feature: platform-resilience-modernization, Property 5: Serialization Round-Trip Consistency (RetryConfig)**
// **Validates: Requirements 7.3**
func TestProperty_RetryConfigSerializationRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxAttempts := rapid.IntRange(1, 10).Draw(t, "maxAttempts")
		baseDelayMs := rapid.IntRange(100, 5000).Draw(t, "baseDelayMs")
		maxDelayMs := rapid.IntRange(5000, 60000).Draw(t, "maxDelayMs")

		cfg := resilience.RetryConfig{
			MaxAttempts: maxAttempts,
			BaseDelay:   time.Duration(baseDelayMs) * time.Millisecond,
			MaxDelay:    time.Duration(maxDelayMs) * time.Millisecond,
		}

		data, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var restored resilience.RetryConfig
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if cfg.MaxAttempts != restored.MaxAttempts {
			t.Fatalf("max attempts mismatch: %d != %d", cfg.MaxAttempts, restored.MaxAttempts)
		}
		if cfg.BaseDelay != restored.BaseDelay {
			t.Fatalf("base delay mismatch: %v != %v", cfg.BaseDelay, restored.BaseDelay)
		}
		if cfg.MaxDelay != restored.MaxDelay {
			t.Fatalf("max delay mismatch: %v != %v", cfg.MaxDelay, restored.MaxDelay)
		}
	})
}

// **Feature: platform-resilience-modernization, Property 13: Policy Format Parsing**
// **Validates: Requirements 15.5**
func TestProperty_PolicyFormatParsing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		nameLen := rapid.IntRange(1, 30).Draw(t, "nameLen")
		version := rapid.Int64Range(1, 100).Draw(t, "version")

		policy := resilience.ResiliencePolicy{
			Name:    testutil.GenerateAlphaString(nameLen),
			Version: version,
		}

		jsonData, err := json.Marshal(policy)
		if err != nil {
			t.Fatalf("json marshal failed: %v", err)
		}

		yamlData, err := yaml.Marshal(policy)
		if err != nil {
			t.Fatalf("yaml marshal failed: %v", err)
		}

		var fromJSON, fromYAML resilience.ResiliencePolicy
		if err := json.Unmarshal(jsonData, &fromJSON); err != nil {
			t.Fatalf("json unmarshal failed: %v", err)
		}
		if err := yaml.Unmarshal(yamlData, &fromYAML); err != nil {
			t.Fatalf("yaml unmarshal failed: %v", err)
		}

		if fromJSON.Name != fromYAML.Name {
			t.Fatalf("name mismatch: %s != %s", fromJSON.Name, fromYAML.Name)
		}
		if fromJSON.Version != fromYAML.Version {
			t.Fatalf("version mismatch: %d != %d", fromJSON.Version, fromYAML.Version)
		}
	})
}
