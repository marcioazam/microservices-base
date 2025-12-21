package health

import (
	"context"
	"testing"

	"github.com/authcorp/libs/go/src/server"
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

	properties.Property("unhealthy > degraded > healthy", prop.ForAll(
		func(_ int) bool {
			// Test ordering using HealthChecker
			hc := server.NewHealthChecker()
			hc.Register("healthy", func(ctx context.Context) server.HealthCheck {
				return server.NewHealthyCheck("healthy")
			})
			resp := hc.Check(context.Background())
			if resp.Status != server.StatusHealthy {
				return false
			}

			hc2 := server.NewHealthChecker()
			hc2.Register("degraded", func(ctx context.Context) server.HealthCheck {
				return server.NewDegradedCheck("degraded", "slow")
			})
			resp2 := hc2.Check(context.Background())
			if resp2.Status != server.StatusDegraded {
				return false
			}

			hc3 := server.NewHealthChecker()
			hc3.Register("unhealthy", func(ctx context.Context) server.HealthCheck {
				return server.NewUnhealthyCheck("unhealthy", "down")
			})
			resp3 := hc3.Check(context.Background())
			if resp3.Status != server.StatusUnhealthy {
				return false
			}

			return true
		},
		gen.Int(),
	))

	properties.TestingRun(t)
}

func TestHealthChecker(t *testing.T) {
	t.Run("Register and Check", func(t *testing.T) {
		hc := server.NewHealthChecker()

		hc.Register("db", func(ctx context.Context) server.HealthCheck {
			return server.NewHealthyCheck("db")
		})
		hc.Register("cache", func(ctx context.Context) server.HealthCheck {
			return server.NewDegradedCheck("cache", "high latency")
		})

		resp := hc.Check(context.Background())

		if resp.Status != server.StatusDegraded {
			t.Errorf("expected Degraded, got %v", resp.Status)
		}
		if len(resp.Checks) != 2 {
			t.Errorf("expected 2 checks, got %d", len(resp.Checks))
		}
	})

	t.Run("Unregister removes check", func(t *testing.T) {
		hc := server.NewHealthChecker()
		hc.Register("test", func(ctx context.Context) server.HealthCheck {
			return server.NewHealthyCheck("test")
		})
		hc.Unregister("test")

		resp := hc.Check(context.Background())
		if len(resp.Checks) != 0 {
			t.Errorf("expected 0 checks, got %d", len(resp.Checks))
		}
	})

	t.Run("WithTimeout sets timeout", func(t *testing.T) {
		hc := server.NewHealthChecker().WithTimeout(100)
		if hc == nil {
			t.Error("expected non-nil")
		}
	})
}

func TestCheckHelpers(t *testing.T) {
	t.Run("NewHealthyCheck", func(t *testing.T) {
		check := server.NewHealthyCheck("test")
		if check.Status != server.StatusHealthy || check.Name != "test" {
			t.Error("unexpected check values")
		}
	})

	t.Run("NewDegradedCheck", func(t *testing.T) {
		check := server.NewDegradedCheck("test", "slow")
		if check.Status != server.StatusDegraded || check.Message != "slow" {
			t.Error("unexpected check values")
		}
	})

	t.Run("NewUnhealthyCheck", func(t *testing.T) {
		check := server.NewUnhealthyCheck("test", "down")
		if check.Status != server.StatusUnhealthy || check.Message != "down" {
			t.Error("unexpected check values")
		}
	})
}

func TestHealthAggregator(t *testing.T) {
	t.Run("NewHealthAggregator creates aggregator", func(t *testing.T) {
		agg := server.NewHealthAggregator()
		if agg == nil {
			t.Error("expected non-nil")
		}
	})

	t.Run("GetStatus returns healthy by default", func(t *testing.T) {
		agg := server.NewHealthAggregator()
		if agg.GetStatus() != server.StatusHealthy {
			t.Error("expected healthy")
		}
	})
}

func TestHealthResponse(t *testing.T) {
	t.Run("empty checks returns healthy", func(t *testing.T) {
		hc := server.NewHealthChecker()
		resp := hc.Check(context.Background())
		if resp.Status != server.StatusHealthy {
			t.Errorf("expected healthy, got %v", resp.Status)
		}
	})

	t.Run("all healthy returns healthy", func(t *testing.T) {
		hc := server.NewHealthChecker()
		hc.Register("a", func(ctx context.Context) server.HealthCheck {
			return server.NewHealthyCheck("a")
		})
		hc.Register("b", func(ctx context.Context) server.HealthCheck {
			return server.NewHealthyCheck("b")
		})
		resp := hc.Check(context.Background())
		if resp.Status != server.StatusHealthy {
			t.Errorf("expected healthy, got %v", resp.Status)
		}
	})

	t.Run("one degraded returns degraded", func(t *testing.T) {
		hc := server.NewHealthChecker()
		hc.Register("a", func(ctx context.Context) server.HealthCheck {
			return server.NewHealthyCheck("a")
		})
		hc.Register("b", func(ctx context.Context) server.HealthCheck {
			return server.NewDegradedCheck("b", "slow")
		})
		resp := hc.Check(context.Background())
		if resp.Status != server.StatusDegraded {
			t.Errorf("expected degraded, got %v", resp.Status)
		}
	})

	t.Run("one unhealthy returns unhealthy", func(t *testing.T) {
		hc := server.NewHealthChecker()
		hc.Register("a", func(ctx context.Context) server.HealthCheck {
			return server.NewHealthyCheck("a")
		})
		hc.Register("b", func(ctx context.Context) server.HealthCheck {
			return server.NewDegradedCheck("b", "slow")
		})
		hc.Register("c", func(ctx context.Context) server.HealthCheck {
			return server.NewUnhealthyCheck("c", "down")
		})
		resp := hc.Check(context.Background())
		if resp.Status != server.StatusUnhealthy {
			t.Errorf("expected unhealthy, got %v", resp.Status)
		}
	})
}
