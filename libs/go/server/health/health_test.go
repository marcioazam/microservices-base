package health

import (
	"context"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 7: Health Status Aggregation Returns Worst**
// **Validates: Requirements 4.2**
func TestHealthStatusAggregationReturnsWorst(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generate status values (0=Healthy, 1=Degraded, 2=Unhealthy)
	statusGen := gen.IntRange(0, 2).Map(func(i int) Status { return Status(i) })

	properties.Property("aggregation returns worst status", prop.ForAll(
		func(statuses []Status) bool {
			if len(statuses) == 0 {
				return AggregateStatusValues(statuses) == Healthy
			}

			result := AggregateStatusValues(statuses)

			// Find expected worst
			expected := Healthy
			for _, s := range statuses {
				if s > expected {
					expected = s
				}
			}

			return result == expected
		},
		gen.SliceOf(statusGen),
	))

	properties.Property("unhealthy > degraded > healthy", prop.ForAll(
		func(_ int) bool {
			// Test ordering
			if AggregateStatusValues([]Status{Healthy, Degraded}) != Degraded {
				return false
			}
			if AggregateStatusValues([]Status{Healthy, Unhealthy}) != Unhealthy {
				return false
			}
			if AggregateStatusValues([]Status{Degraded, Unhealthy}) != Unhealthy {
				return false
			}
			if AggregateStatusValues([]Status{Healthy, Degraded, Unhealthy}) != Unhealthy {
				return false
			}
			return true
		},
		gen.Int(),
	))

	properties.Property("empty list returns Healthy", prop.ForAll(
		func(_ int) bool {
			return AggregateStatusValues([]Status{}) == Healthy
		},
		gen.Int(),
	))

	properties.TestingRun(t)
}

func TestStatusString(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{Healthy, "healthy"},
		{Degraded, "degraded"},
		{Unhealthy, "unhealthy"},
		{Status(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("Status(%d).String() = %s, want %s", tt.status, got, tt.expected)
		}
	}
}

func TestAggregator(t *testing.T) {
	t.Run("Register and Check", func(t *testing.T) {
		agg := NewAggregator()

		agg.RegisterFunc("db", func(ctx context.Context) Check {
			return NewHealthyCheck("db")
		})
		agg.RegisterFunc("cache", func(ctx context.Context) Check {
			return NewDegradedCheck("cache", "high latency")
		})

		status, checks := agg.Check(context.Background())

		if status != Degraded {
			t.Errorf("expected Degraded, got %v", status)
		}
		if len(checks) != 2 {
			t.Errorf("expected 2 checks, got %d", len(checks))
		}
	})

	t.Run("OnChange callback", func(t *testing.T) {
		agg := NewAggregator()

		var changedCheck Check
		agg.OnChange(func(c Check) {
			changedCheck = c
		})

		agg.RegisterFunc("test", func(ctx context.Context) Check {
			return NewUnhealthyCheck("test", "failed")
		})

		agg.Check(context.Background())

		if changedCheck.Name != "test" {
			t.Error("expected onChange to be called")
		}
	})

	t.Run("GetStatus without running checks", func(t *testing.T) {
		agg := NewAggregator()

		agg.RegisterFunc("test", func(ctx context.Context) Check {
			return NewHealthyCheck("test")
		})

		// Before any check
		if agg.GetStatus() != Healthy {
			t.Error("expected Healthy for empty results")
		}

		// After check
		agg.Check(context.Background())
		if agg.GetStatus() != Healthy {
			t.Error("expected Healthy after check")
		}
	})

	t.Run("GetChecks returns results", func(t *testing.T) {
		agg := NewAggregator()

		agg.RegisterFunc("test", func(ctx context.Context) Check {
			return NewHealthyCheck("test")
		})

		agg.Check(context.Background())
		checks := agg.GetChecks()

		if len(checks) != 1 {
			t.Errorf("expected 1 check, got %d", len(checks))
		}
	})
}

func TestCheckHelpers(t *testing.T) {
	t.Run("NewHealthyCheck", func(t *testing.T) {
		check := NewHealthyCheck("test")
		if check.Status != Healthy || check.Name != "test" {
			t.Error("unexpected check values")
		}
	})

	t.Run("NewDegradedCheck", func(t *testing.T) {
		check := NewDegradedCheck("test", "slow")
		if check.Status != Degraded || check.Message != "slow" {
			t.Error("unexpected check values")
		}
	})

	t.Run("NewUnhealthyCheck", func(t *testing.T) {
		check := NewUnhealthyCheck("test", "down")
		if check.Status != Unhealthy || check.Message != "down" {
			t.Error("unexpected check values")
		}
	})

	t.Run("WithDetails", func(t *testing.T) {
		check := NewHealthyCheck("test").WithDetails(map[string]interface{}{
			"latency": 100,
		})
		if check.Details["latency"] != 100 {
			t.Error("expected details to be set")
		}
	})
}

func TestAggregateStatuses(t *testing.T) {
	tests := []struct {
		name     string
		checks   []Check
		expected Status
	}{
		{
			name:     "empty returns healthy",
			checks:   []Check{},
			expected: Healthy,
		},
		{
			name: "all healthy",
			checks: []Check{
				{Status: Healthy},
				{Status: Healthy},
			},
			expected: Healthy,
		},
		{
			name: "one degraded",
			checks: []Check{
				{Status: Healthy},
				{Status: Degraded},
			},
			expected: Degraded,
		},
		{
			name: "one unhealthy",
			checks: []Check{
				{Status: Healthy},
				{Status: Degraded},
				{Status: Unhealthy},
			},
			expected: Unhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AggregateStatuses(tt.checks)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
