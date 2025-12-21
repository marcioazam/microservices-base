package property

import (
	"context"
	"testing"

	"github.com/auth-platform/libs/go/resilience/health"
	"pgregory.net/rapid"
)

// **Feature: resilience-service-state-of-art-2025, Property 18: Health Aggregation Logic**
// **Validates: Requirements 6.1**
func TestProperty_HealthAggregationLogic(t *testing.T) {
	t.Run("all_healthy_returns_healthy", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			serviceCount := rapid.IntRange(1, 10).Draw(t, "serviceCount")

			agg := health.NewAggregator(health.Config{})
			for i := 0; i < serviceCount; i++ {
				name := string(rune('a' + i))
				agg.UpdateHealth(name, health.HealthHealthy, "")
			}
			h, _ := agg.GetAggregatedHealth(context.Background())
			if h.Status != health.HealthHealthy {
				t.Fatalf("expected healthy, got %s", h.Status)
			}
		})
	})

	t.Run("any_degraded_returns_degraded", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			healthyCount := rapid.IntRange(1, 10).Draw(t, "healthyCount")

			agg := health.NewAggregator(health.Config{})
			for i := 0; i < healthyCount; i++ {
				name := string(rune('a' + i))
				agg.UpdateHealth(name, health.HealthHealthy, "")
			}
			agg.UpdateHealth("degraded", health.HealthDegraded, "degraded")
			h, _ := agg.GetAggregatedHealth(context.Background())
			if h.Status != health.HealthDegraded {
				t.Fatalf("expected degraded, got %s", h.Status)
			}
		})
	})

	t.Run("any_unhealthy_returns_unhealthy", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			healthyCount := rapid.IntRange(0, 5).Draw(t, "healthyCount")
			degradedCount := rapid.IntRange(0, 5).Draw(t, "degradedCount")

			agg := health.NewAggregator(health.Config{})
			for i := 0; i < healthyCount; i++ {
				name := string(rune('a' + i))
				agg.UpdateHealth(name, health.HealthHealthy, "")
			}
			for i := 0; i < degradedCount; i++ {
				name := "degraded-" + string(rune('a'+i))
				agg.UpdateHealth(name, health.HealthDegraded, "")
			}
			agg.UpdateHealth("unhealthy", health.HealthUnhealthy, "unhealthy")
			h, _ := agg.GetAggregatedHealth(context.Background())
			if h.Status != health.HealthUnhealthy {
				t.Fatalf("expected unhealthy, got %s", h.Status)
			}
		})
	})

	t.Run("aggregate_statuses_function", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			statusCount := rapid.IntRange(1, 10).Draw(t, "statusCount")
			statuses := make([]int, statusCount)
			for i := 0; i < statusCount; i++ {
				statuses[i] = rapid.IntRange(0, 2).Draw(t, "status")
			}

			healthStatuses := make([]health.HealthStatus, len(statuses))
			hasUnhealthy := false
			hasDegraded := false

			for i, s := range statuses {
				switch s % 3 {
				case 0:
					healthStatuses[i] = health.HealthHealthy
				case 1:
					healthStatuses[i] = health.HealthDegraded
					hasDegraded = true
				case 2:
					healthStatuses[i] = health.HealthUnhealthy
					hasUnhealthy = true
				}
			}

			result := health.AggregateStatuses(healthStatuses)

			if hasUnhealthy {
				if result != health.HealthUnhealthy {
					t.Fatalf("expected unhealthy when hasUnhealthy, got %s", result)
				}
			} else if hasDegraded {
				if result != health.HealthDegraded {
					t.Fatalf("expected degraded when hasDegraded, got %s", result)
				}
			} else {
				if result != health.HealthHealthy {
					t.Fatalf("expected healthy, got %s", result)
				}
			}
		})
	})
}
