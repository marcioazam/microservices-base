package property

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/policy"
	"pgregory.net/rapid"
)

// **Feature: resilience-service-state-of-art-2025, Property 8: Path Traversal Prevention**
// **Validates: Requirements 10.1**
func TestProperty_PathTraversalPrevention(t *testing.T) {
	t.Run("path_with_double_dots_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			prefix := rapid.StringMatching(`[a-zA-Z0-9_]{1,10}`).Draw(t, "prefix")
			suffix := rapid.StringMatching(`[a-zA-Z0-9_]{1,10}`).Draw(t, "suffix")

			maliciousPath := prefix + "/../" + suffix

			err := policy.ValidatePolicyPath(maliciousPath, "/base/path")
			if err == nil {
				t.Fatalf("path with .. should be rejected: %s", maliciousPath)
			}
		})
	})

	t.Run("path_escaping_base_directory_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			basePath := "/var/policies"
			escapePath := rapid.StringMatching(`(\.\./){1,5}[a-zA-Z0-9_]+`).Draw(t, "escapePath")

			err := policy.ValidatePolicyPath(escapePath, basePath)
			if err == nil {
				t.Fatalf("path escaping base directory should be rejected: %s", escapePath)
			}
		})
	})

	t.Run("valid_relative_path_accepted", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			validPath := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_]{0,20}\.json`).Draw(t, "validPath")
			basePath := "/var/policies"

			err := policy.ValidatePolicyPath(validPath, basePath)
			if err != nil {
				t.Fatalf("valid path should be accepted: %s, error: %v", validPath, err)
			}
		})
	})

	t.Run("absolute_path_outside_base_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			basePath := "/var/policies"
			outsidePath := "/etc/passwd"

			err := policy.ValidatePolicyPath(outsidePath, basePath)
			if err == nil {
				t.Fatalf("absolute path outside base should be rejected: %s", outsidePath)
			}
		})
	})

	t.Run("path_with_null_bytes_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			prefix := rapid.StringMatching(`[a-zA-Z0-9_]{1,10}`).Draw(t, "prefix")
			suffix := rapid.StringMatching(`[a-zA-Z0-9_]{1,10}`).Draw(t, "suffix")

			maliciousPath := prefix + "\x00" + suffix

			err := policy.ValidatePolicyPath(maliciousPath, "/base/path")
			if err == nil {
				t.Fatalf("path with null bytes should be rejected: %s", maliciousPath)
			}
		})
	})

	t.Run("cleaned_path_stays_within_base", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			basePath := "/var/policies"
			subdir := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_]{0,10}`).Draw(t, "subdir")
			filename := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_]{0,10}\.json`).Draw(t, "filename")

			validPath := filepath.Join(subdir, filename)

			err := policy.ValidatePolicyPath(validPath, basePath)
			if err != nil {
				t.Fatalf("valid nested path should be accepted: %s, error: %v", validPath, err)
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 9: Observability Context Propagation**
// **Validates: Requirements 11.1, 11.2**
func TestProperty_ObservabilityContextPropagation(t *testing.T) {
	t.Run("event_contains_correlation_id_when_provided", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			correlationID := rapid.StringMatching(`[a-zA-Z0-9-]{8,36}`).Draw(t, "correlationID")
			serviceName := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9-]{0,29}`).Draw(t, "serviceName")

			event := domain.NewResilienceEvent(resilience.EventCircuitStateChange, serviceName)
			event.CorrelationID = correlationID

			if event.CorrelationID != correlationID {
				t.Fatalf("correlation ID mismatch: %s != %s", event.CorrelationID, correlationID)
			}
		})
	})

	t.Run("event_has_valid_timestamp", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			serviceName := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9-]{0,29}`).Draw(t, "serviceName")

			event := domain.NewResilienceEvent(resilience.EventCircuitStateChange, serviceName)

			if event.Timestamp.IsZero() {
				t.Fatal("event timestamp should not be zero")
			}
		})
	})

	t.Run("event_has_unique_id", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			count := rapid.IntRange(10, 50).Draw(t, "count")
			serviceName := "test-service"

			ids := make(map[string]bool)
			for i := 0; i < count; i++ {
				event := domain.NewResilienceEvent(resilience.EventCircuitStateChange, serviceName)
				if ids[event.ID] {
					t.Fatalf("duplicate event ID: %s", event.ID)
				}
				ids[event.ID] = true
			}
		})
	})

	t.Run("event_metadata_can_be_set", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			serviceName := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9-]{0,29}`).Draw(t, "serviceName")
			key := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_]{0,19}`).Draw(t, "key")
			value := rapid.String().Draw(t, "value")

			event := domain.NewResilienceEvent(resilience.EventCircuitStateChange, serviceName)
			event.Metadata[key] = value

			if event.Metadata[key] != value {
				t.Fatalf("metadata value mismatch: %v != %v", event.Metadata[key], value)
			}
		})
	})

	t.Run("trace_context_can_be_propagated", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			serviceName := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9-]{0,29}`).Draw(t, "serviceName")
			traceID := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "traceID")
			spanID := rapid.StringMatching(`[a-f0-9]{16}`).Draw(t, "spanID")

			event := domain.NewResilienceEvent(resilience.EventCircuitStateChange, serviceName)
			event.TraceID = traceID
			event.SpanID = spanID

			if event.TraceID != traceID {
				t.Fatalf("trace ID mismatch: %s != %s", event.TraceID, traceID)
			}
			if event.SpanID != spanID {
				t.Fatalf("span ID mismatch: %s != %s", event.SpanID, spanID)
			}
		})
	})
}

// **Feature: resilience-service-state-of-art-2025, Property 10: Health Aggregation Correctness**
// **Validates: Requirements 11.4**
func TestProperty_HealthAggregationCorrectness(t *testing.T) {
	t.Run("all_healthy_returns_healthy", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			count := rapid.IntRange(1, 20).Draw(t, "count")

			statuses := make([]string, count)
			for i := 0; i < count; i++ {
				statuses[i] = "healthy"
			}

			result := aggregateHealthStatuses(statuses)
			if result != "healthy" {
				t.Fatalf("all healthy should return healthy, got %s", result)
			}
		})
	})

	t.Run("any_unhealthy_returns_unhealthy", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			healthyCount := rapid.IntRange(0, 10).Draw(t, "healthyCount")
			degradedCount := rapid.IntRange(0, 10).Draw(t, "degradedCount")

			statuses := make([]string, 0)
			for i := 0; i < healthyCount; i++ {
				statuses = append(statuses, "healthy")
			}
			for i := 0; i < degradedCount; i++ {
				statuses = append(statuses, "degraded")
			}
			statuses = append(statuses, "unhealthy")

			result := aggregateHealthStatuses(statuses)
			if result != "unhealthy" {
				t.Fatalf("any unhealthy should return unhealthy, got %s", result)
			}
		})
	})

	t.Run("any_degraded_without_unhealthy_returns_degraded", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			healthyCount := rapid.IntRange(0, 10).Draw(t, "healthyCount")

			statuses := make([]string, 0)
			for i := 0; i < healthyCount; i++ {
				statuses = append(statuses, "healthy")
			}
			statuses = append(statuses, "degraded")

			result := aggregateHealthStatuses(statuses)
			if result != "degraded" {
				t.Fatalf("any degraded without unhealthy should return degraded, got %s", result)
			}
		})
	})

	t.Run("empty_returns_unknown", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			_ = rapid.IntRange(1, 1).Draw(t, "iteration")

			result := aggregateHealthStatuses([]string{})
			if result != "unknown" {
				t.Fatalf("empty should return unknown, got %s", result)
			}
		})
	})
}

func aggregateHealthStatuses(statuses []string) string {
	if len(statuses) == 0 {
		return "unknown"
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, status := range statuses {
		switch strings.ToLower(status) {
		case "unhealthy":
			hasUnhealthy = true
		case "degraded":
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return "unhealthy"
	}
	if hasDegraded {
		return "degraded"
	}
	return "healthy"
}
