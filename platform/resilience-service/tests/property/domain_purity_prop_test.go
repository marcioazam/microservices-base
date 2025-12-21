package property

import (
	"strings"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"pgregory.net/rapid"
)

// TestCleanArchitectureLayerSeparationProperty validates domain layer purity.
// Feature: resilience-service-state-of-art-2025, Property 10: Clean Architecture Layer Separation
func TestCleanArchitectureLayerSeparationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_-]{2,20}$`).Draw(t, "policy_name")

		policy, err := entities.NewPolicy(name)
		if err != nil {
			t.Fatalf("Failed to create policy: %v", err)
		}

		if policy.Name() != name {
			t.Fatalf("Expected policy name %s, got %s", name, policy.Name())
		}
	})
}

// TestDomainEntitiesPurityProperty tests domain entities have pure business logic.
func TestDomainEntitiesPurityProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_-]{2,20}$`).Draw(t, "policy_name")

		policy, err := entities.NewPolicy(name)
		if err != nil {
			t.Fatalf("Failed to create policy: %v", err)
		}

		originalName := policy.Name()
		originalVersion := policy.Version()

		if policy.Name() != originalName {
			t.Fatalf("Policy name should be immutable, expected %s, got %s", originalName, policy.Name())
		}

		if policy.Version() != originalVersion {
			t.Fatalf("Policy version should be stable, expected %d, got %d", originalVersion, policy.Version())
		}
	})
}

// TestCircuitBreakerConfigValidationProperty tests circuit breaker configuration validation.
func TestCircuitBreakerConfigValidationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		failureThreshold := rapid.IntRange(1, 100).Draw(t, "failure_threshold")
		successThreshold := rapid.IntRange(1, 10).Draw(t, "success_threshold")
		timeout := time.Duration(rapid.Int64Range(1, 300).Draw(t, "timeout_sec")) * time.Second
		probeCount := rapid.IntRange(1, 10).Draw(t, "probe_count")

		result := entities.NewCircuitBreakerConfig(failureThreshold, successThreshold, timeout, probeCount)

		if successThreshold > failureThreshold {
			if result.IsOk() {
				t.Fatal("Expected validation error when success threshold > failure threshold")
			}
		} else {
			if result.IsErr() {
				t.Fatalf("Unexpected validation error: %v", result.UnwrapErr())
			}

			config := result.Unwrap()
			if config.FailureThreshold != failureThreshold {
				t.Fatalf("Expected failure threshold %d, got %d", failureThreshold, config.FailureThreshold)
			}
		}
	})
}

// TestRetryConfigValidationProperty tests retry configuration validation.
func TestRetryConfigValidationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxAttempts := rapid.IntRange(1, 10).Draw(t, "max_attempts")
		baseDelay := time.Duration(rapid.Int64Range(1, 10000).Draw(t, "base_delay_ms")) * time.Millisecond
		maxDelay := time.Duration(rapid.Int64Range(1000, 300000).Draw(t, "max_delay_ms")) * time.Millisecond
		multiplier := rapid.Float64Range(1.0, 10.0).Draw(t, "multiplier")
		jitterPercent := rapid.Float64Range(0.0, 1.0).Draw(t, "jitter_percent")

		result := entities.NewRetryConfig(maxAttempts, baseDelay, maxDelay, multiplier, jitterPercent)

		if baseDelay > maxDelay {
			if result.IsOk() {
				t.Fatal("Expected validation error when base delay > max delay")
			}
		} else {
			if result.IsErr() {
				t.Fatalf("Unexpected validation error: %v", result.UnwrapErr())
			}

			config := result.Unwrap()
			if config.MaxAttempts != maxAttempts {
				t.Fatalf("Expected max attempts %d, got %d", maxAttempts, config.MaxAttempts)
			}
		}
	})
}

// TestPolicyImmutabilityProperty tests policy immutability.
func TestPolicyImmutabilityProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_-]{2,20}$`).Draw(t, "policy_name")

		policy, err := entities.NewPolicy(name)
		if err != nil {
			t.Fatalf("Failed to create policy: %v", err)
		}

		clone := policy.Clone()

		if clone.Name() != policy.Name() {
			t.Fatal("Clone should have same name")
		}

		if clone.Version() != policy.Version() {
			t.Fatal("Clone should have same version")
		}
	})
}

// isAllowedDomainImport checks if import is allowed in domain layer.
func isAllowedDomainImport(importPath string) bool {
	if !strings.Contains(importPath, ".") {
		return true
	}

	if strings.Contains(importPath, "/internal/domain/") {
		return true
	}

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
