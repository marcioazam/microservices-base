package health

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 8: Health Status Aggregation**
// **Validates: Requirements 2.1**
// *For any* set of service health statuses, the aggregated status SHALL be the worst status (unhealthy > degraded > healthy).
func TestHealthStatusAggregation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	genHealthStatus := gen.OneConstOf(HealthHealthy, HealthDegraded, HealthUnhealthy)

	properties.Property("aggregated status is worst status", prop.ForAll(
		func(statuses []HealthStatus) bool {
			if len(statuses) == 0 {
				return AggregateStatuses(statuses) == HealthHealthy
			}

			result := AggregateStatuses(statuses)

			// Find expected worst status
			hasUnhealthy := false
			hasDegraded := false
			for _, s := range statuses {
				if s == HealthUnhealthy {
					hasUnhealthy = true
				}
				if s == HealthDegraded {
					hasDegraded = true
				}
			}

			if hasUnhealthy {
				return result == HealthUnhealthy
			}
			if hasDegraded {
				return result == HealthDegraded
			}
			return result == HealthHealthy
		},
		gen.SliceOf(genHealthStatus),
	))

	properties.Property("unhealthy always wins", prop.ForAll(
		func(statuses []HealthStatus) bool {
			statusesWithUnhealthy := append(statuses, HealthUnhealthy)
			return AggregateStatuses(statusesWithUnhealthy) == HealthUnhealthy
		},
		gen.SliceOf(genHealthStatus),
	))

	properties.Property("degraded beats healthy", prop.ForAll(
		func(n int) bool {
			statuses := make([]HealthStatus, n)
			for i := 0; i < n; i++ {
				statuses[i] = HealthHealthy
			}
			statuses = append(statuses, HealthDegraded)
			return AggregateStatuses(statuses) == HealthDegraded
		},
		gen.IntRange(0, 10),
	))

	properties.TestingRun(t)
}

func TestAggregateStatusPairwise(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	genHealthStatus := gen.OneConstOf(HealthHealthy, HealthDegraded, HealthUnhealthy)

	properties.Property("aggregateStatus is commutative", prop.ForAll(
		func(a, b HealthStatus) bool {
			return aggregateStatus(a, b) == aggregateStatus(b, a)
		},
		genHealthStatus,
		genHealthStatus,
	))

	properties.Property("aggregateStatus is idempotent", prop.ForAll(
		func(s HealthStatus) bool {
			return aggregateStatus(s, s) == s
		},
		genHealthStatus,
	))

	properties.TestingRun(t)
}
