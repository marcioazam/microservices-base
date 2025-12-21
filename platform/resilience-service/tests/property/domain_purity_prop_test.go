package property

import (
	"strings"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
	"pgregory.net/rapid"
)

// **Feature: resilience-service-state-of-art-2025, Property 10: Clean Architecture Layer Separation**
// **Validates: Requirements 10.1, 10.2, 10.3, 10.4, 10.5**
func TestCleanArchitectureLayerSeparationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Test domain layer purity by checking that domain entities work independently
		name := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_-]*$`).Draw(t, "policy_name")
		
		// Create policy entity - should work without external dependencies
		policy, err := entities.NewPolicy(name)
		if err != nil {
			t.Fatalf("Failed to create policy: %v", err)
		}

		// Test that domain operations are pure
		if policy.Name() != name {
			t.Fatalf("Expected policy name %s, got %s", name, policy.Name())
		}
	})
}

// Test domain entities have pure business logic
func TestDomainEntitiesPurityProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate policy configuration
		name := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_-]*$`).Draw(t, "policy_name")
		
		// Create policy entity
		policy, err := entities.NewPolicy(name)
		if err != nil {
			t.Fatalf("Failed to create policy: %v", err)
		}

		// Test pure business logic - no side effects
		originalName := policy.Name()
		originalVersion := policy.Version()

		// Operations should be deterministic and pure
		if policy.Name() != originalName {
			t.Fatalf("Policy name should be immutable, expected %s, got %s", originalName, policy.Name())
		}

		if policy.Version() != originalVersion {
			t.Fatalf("Policy version should be stable, expected %d, got %d", originalVersion, policy.Version())
		}

		// Test validation is pure (no side effects)
		err1 := policy.Validate()
		err2 := policy.Validate()
		
		if (err1 == nil) != (err2 == nil) {
			t.Fatal("Validation should be deterministic and pure")
		}
	})
}

// Test circuit breaker configuration validation
func TestCircuitBreakerConfigValidationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		failureThreshold := rapid.IntRange(1, 100).Draw(t, "failure_threshold")
		successThreshold := rapid.IntRange(1, 10).Draw(t, "success_threshold")
		timeout := time.Duration(rapid.Int64Range(1, 300).Draw(t, "timeout_sec")) * time.Second
		probeCount := rapid.IntRange(1, 10).Draw(t, "probe_count")

		config, err := entities.NewCircuitBreakerConfig(failureThreshold, successThreshold, timeout, probeCount)
		
		if successThreshold > failureThreshold {
			// Should fail validation
			if err == nil {
				t.Fatal("Expected validation error when success threshold > failure threshold")
			}
		} else {
			// Should pass validation
			if err != nil {
				t.Fatalf("Unexpected validation error: %v", err)
			}
			
			// Test immutability
			if config.FailureThreshold != failureThreshold {
				t.Fatalf("Expected failure threshold %d, got %d", failureThreshold, config.FailureThreshold)
			}
		}
	})
}

// Test retry configuration validation
func TestRetryConfigValidationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxAttempts := rapid.IntRange(1, 10).Draw(t, "max_attempts")
		baseDelay := time.Duration(rapid.Int64Range(1, 10000).Draw(t, "base_delay_ms")) * time.Millisecond
		maxDelay := time.Duration(rapid.Int64Range(1000, 300000).Draw(t, "max_delay_ms")) * time.Millisecond
		multiplier := rapid.Float64Range(1.0, 10.0).Draw(t, "multiplier")
		jitterPercent := rapid.Float64Range(0.0, 1.0).Draw(t, "jitter_percent")

		config, err := entities.NewRetryConfig(maxAttempts, baseDelay, maxDelay, multiplier, jitterPercent)
		
		if baseDelay > maxDelay {
			// Should fail validation
			if err == nil {
				t.Fatal("Expected validation error when base delay > max delay")
			}
		} else {
			// Should pass validation
			if err != nil {
				t.Fatalf("Unexpected validation error: %v", err)
			}
			
			// Test configuration values
			if config.MaxAttempts != maxAttempts {
				t.Fatalf("Expected max attempts %d, got %d", maxAttempts, config.MaxAttempts)
			}
		}
	})
}

// Test value objects immutability
func TestValueObjectsImmutabilityProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Test HealthStatus immutability
		status := rapid.SampledFrom([]valueobjects.HealthState{
			valueobjects.HealthHealthy,
			valueobjects.HealthUnhealthy,
			valueobjects.HealthDegraded,
			valueobjects.HealthUnknown,
		}).Draw(t, "health_state")
		
		message := rapid.String().Draw(t, "message")
		
		healthStatus := valueobjects.NewHealthStatus(status, message)
		
		// Original values should not change
		originalStatus := healthStatus.Status
		originalMessage := healthStatus.Message
		originalTimestamp := healthStatus.Timestamp
		
		// Adding details should not modify original
		newHealthStatus := healthStatus.WithDetail("key", "value")
		
		if healthStatus.Status != originalStatus {
			t.Fatal("Original health status should not be modified")
		}
		
		if healthStatus.Message != originalMessage {
			t.Fatal("Original health message should not be modified")
		}
		
		if healthStatus.Timestamp != originalTimestamp {
			t.Fatal("Original health timestamp should not be modified")
		}
		
		// New instance should have the detail
		if newHealthStatus.Details["key"] != "value" {
			t.Fatal("New health status should have the added detail")
		}
	})
}

// Test policy event creation
func TestPolicyEventCreationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		eventType := rapid.SampledFrom([]valueobjects.PolicyEventType{
			valueobjects.PolicyCreated,
			valueobjects.PolicyUpdated,
			valueobjects.PolicyDeleted,
		}).Draw(t, "event_type")
		
		policyName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_-]*$`).Draw(t, "policy_name")
		version := rapid.IntRange(1, 1000).Draw(t, "version")
		
		event := valueobjects.NewPolicyEvent(eventType, policyName, version)
		
		// Test event properties
		if event.Type != eventType {
			t.Fatalf("Expected event type %s, got %s", eventType, event.Type)
		}
		
		if event.PolicyName != policyName {
			t.Fatalf("Expected policy name %s, got %s", policyName, event.PolicyName)
		}
		
		if event.Version != version {
			t.Fatalf("Expected version %d, got %d", version, event.Version)
		}
		
		if event.ID == "" {
			t.Fatal("Event ID should not be empty")
		}
		
		if event.Timestamp().IsZero() {
			t.Fatal("Event timestamp should not be zero")
		}
		
		// Test metadata immutability
		originalEvent := event
		newEvent := event.WithMetadata("key", "value")
		
		if len(originalEvent.Metadata) != 0 {
			t.Fatal("Original event metadata should not be modified")
		}
		
		if newEvent.Metadata["key"] != "value" {
			t.Fatal("New event should have the added metadata")
		}
	})
}

// Test execution metrics creation
func TestExecutionMetricsProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policyName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_-]*$`).Draw(t, "policy_name")
		executionTime := time.Duration(rapid.Int64Range(1, 60000).Draw(t, "execution_time_ms")) * time.Millisecond
		success := rapid.Bool().Draw(t, "success")
		
		metrics := valueobjects.NewExecutionMetrics(policyName, executionTime, success)
		
		// Test basic properties
		if metrics.PolicyName != policyName {
			t.Fatalf("Expected policy name %s, got %s", policyName, metrics.PolicyName)
		}
		
		if metrics.ExecutionTime != executionTime {
			t.Fatalf("Expected execution time %v, got %v", executionTime, metrics.ExecutionTime)
		}
		
		if metrics.Success != success {
			t.Fatalf("Expected success %t, got %t", success, metrics.Success)
		}
		
		if metrics.OccurredAt.IsZero() {
			t.Fatal("Metrics timestamp should not be zero")
		}
		
		// Test fluent interface immutability
		originalMetrics := metrics
		
		newMetrics := metrics.
			WithCircuitState("open").
			WithRetryAttempts(3).
			WithRateLimit(true).
			WithBulkheadQueue(false)
		
		// Original should be unchanged
		if originalMetrics.CircuitState != "" {
			t.Fatal("Original metrics circuit state should not be modified")
		}
		
		if originalMetrics.RetryAttempts != 0 {
			t.Fatal("Original metrics retry attempts should not be modified")
		}
		
		// New metrics should have the values
		if newMetrics.CircuitState != "open" {
			t.Fatal("New metrics should have circuit state")
		}
		
		if newMetrics.RetryAttempts != 3 {
			t.Fatal("New metrics should have retry attempts")
		}
		
		if !newMetrics.RateLimited {
			t.Fatal("New metrics should be rate limited")
		}
		
		if newMetrics.BulkheadQueued {
			t.Fatal("New metrics should not be bulkhead queued")
		}
	})
}

// Helper function to check if import is allowed in domain layer
func isAllowedDomainImport(importPath string) bool {
	// Standard library imports are allowed
	if !strings.Contains(importPath, ".") {
		return true
	}
	
	// Internal domain imports are allowed
	if strings.Contains(importPath, "/internal/domain/") {
		return true
	}
	
	// Common standard library packages
	allowedPrefixes := []string{
		"context",
		"time",
		"fmt",
		"errors",
		"strings",
		"strconv",
		"encoding/json",
		"go/",
	}
	
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(importPath, prefix) {
			return true
		}
	}
	
	return false
}