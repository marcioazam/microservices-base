package health

import (
	"context"
	"sync"
	"testing"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/testutil"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-microservice, Property 18: Health Aggregation Logic**
// **Validates: Requirements 6.1**
func TestProperty_HealthAggregationLogic(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("all_healthy_returns_healthy", prop.ForAll(
		func(serviceCount int) bool {
			agg := NewAggregator(Config{})

			// Register all healthy services
			for i := 0; i < serviceCount; i++ {
				name := string(rune('a' + i))
				agg.UpdateHealth(name, domain.HealthHealthy, "")
			}

			health, _ := agg.GetAggregatedHealth(context.Background())
			return health.Status == domain.HealthHealthy
		},
		gen.IntRange(1, 10),
	))

	props.Property("any_degraded_returns_degraded", prop.ForAll(
		func(healthyCount int, degradedIndex int) bool {
			agg := NewAggregator(Config{})

			// Register healthy services
			for i := 0; i < healthyCount; i++ {
				name := string(rune('a' + i))
				agg.UpdateHealth(name, domain.HealthHealthy, "")
			}

			// Add one degraded service
			agg.UpdateHealth("degraded", domain.HealthDegraded, "degraded")

			health, _ := agg.GetAggregatedHealth(context.Background())
			return health.Status == domain.HealthDegraded
		},
		gen.IntRange(1, 10),
		gen.IntRange(0, 9),
	))

	props.Property("any_unhealthy_returns_unhealthy", prop.ForAll(
		func(healthyCount int, degradedCount int) bool {
			agg := NewAggregator(Config{})

			// Register healthy services
			for i := 0; i < healthyCount; i++ {
				name := string(rune('a' + i))
				agg.UpdateHealth(name, domain.HealthHealthy, "")
			}

			// Register degraded services
			for i := 0; i < degradedCount; i++ {
				name := "degraded-" + string(rune('a'+i))
				agg.UpdateHealth(name, domain.HealthDegraded, "")
			}

			// Add one unhealthy service
			agg.UpdateHealth("unhealthy", domain.HealthUnhealthy, "unhealthy")

			health, _ := agg.GetAggregatedHealth(context.Background())
			return health.Status == domain.HealthUnhealthy
		},
		gen.IntRange(0, 5),
		gen.IntRange(0, 5),
	))

	props.Property("aggregate_statuses_function", prop.ForAll(
		func(statuses []int) bool {
			// Convert ints to HealthStatus
			healthStatuses := make([]domain.HealthStatus, len(statuses))
			hasUnhealthy := false
			hasDegraded := false

			for i, s := range statuses {
				switch s % 3 {
				case 0:
					healthStatuses[i] = domain.HealthHealthy
				case 1:
					healthStatuses[i] = domain.HealthDegraded
					hasDegraded = true
				case 2:
					healthStatuses[i] = domain.HealthUnhealthy
					hasUnhealthy = true
				}
			}

			result := AggregateStatuses(healthStatuses)

			if hasUnhealthy {
				return result == domain.HealthUnhealthy
			}
			if hasDegraded {
				return result == domain.HealthDegraded
			}
			return result == domain.HealthHealthy
		},
		gen.SliceOfN(10, gen.IntRange(0, 2)),
	))

	props.TestingRun(t)
}

// **Feature: resilience-microservice, Property 19: Health Change Event Emission**
// **Validates: Requirements 6.2**
func TestProperty_HealthChangeEventEmission(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("status_change_emits_event", prop.ForAll(
		func(initialStatus int, newStatus int) bool {
			// Skip if statuses are the same
			if initialStatus%3 == newStatus%3 {
				return true
			}

			emitter := newMockEmitter()
			builder := domain.NewEventBuilder(emitter, "health-aggregator", nil)
			agg := NewAggregator(Config{EventBuilder: builder})

			// Set initial status
			initial := statusFromInt(initialStatus)
			agg.UpdateHealth("test-service", initial, "initial")

			// Clear events from initial setup
			emitter.Clear()

			// Change status
			newStat := statusFromInt(newStatus)
			agg.UpdateHealth("test-service", newStat, "changed")

			// Should have emitted exactly one event
			events := emitter.GetEvents()
			if len(events) != 1 {
				return false
			}

			event := events[0]
			if event.Type != domain.EventHealthChange {
				return false
			}

			// Check metadata
			prevStatus, ok := event.Metadata["previous_status"].(string)
			if !ok || prevStatus != string(initial) {
				return false
			}

			newStatusMeta, ok := event.Metadata["new_status"].(string)
			if !ok || newStatusMeta != string(newStat) {
				return false
			}

			return true
		},
		gen.IntRange(0, 2),
		gen.IntRange(0, 2),
	))

	props.Property("same_status_no_event", prop.ForAll(
		func(status int) bool {
			emitter := newMockEmitter()
			builder := domain.NewEventBuilder(emitter, "health-aggregator", nil)
			agg := NewAggregator(Config{EventBuilder: builder})

			stat := statusFromInt(status)

			// Set initial status
			agg.UpdateHealth("test-service", stat, "initial")
			emitter.Clear()

			// Set same status again
			agg.UpdateHealth("test-service", stat, "same")

			// Should not have emitted any events
			return len(emitter.GetEvents()) == 0
		},
		gen.IntRange(0, 2),
	))

	props.TestingRun(t)
}

func statusFromInt(i int) domain.HealthStatus {
	switch i % 3 {
	case 0:
		return domain.HealthHealthy
	case 1:
		return domain.HealthDegraded
	default:
		return domain.HealthUnhealthy
	}
}

// mockEmitter is a test implementation of EventEmitter.
type mockEmitter struct {
	mu     sync.Mutex
	events []domain.ResilienceEvent
}

func newMockEmitter() *mockEmitter {
	return &mockEmitter{events: make([]domain.ResilienceEvent, 0)}
}

func (m *mockEmitter) Emit(event domain.ResilienceEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockEmitter) EmitAudit(event domain.AuditEvent) {}

func (m *mockEmitter) GetEvents() []domain.ResilienceEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]domain.ResilienceEvent, len(m.events))
	copy(result, m.events)
	return result
}

func (m *mockEmitter) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = nil
}
