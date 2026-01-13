// Package property contains property-based tests.
package property

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/auth-platform/cache-service/internal/localcache"
	"github.com/authcorp/libs/go/src/fault"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// tripCircuit trips the circuit breaker by executing failing operations.
func tripCircuit(cb *fault.CircuitBreaker, failures int) {
	for i := 0; i < failures; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("simulated failure")
		})
	}
}

// TestDegradedModeResponse tests Property 15: Degraded Mode Response.
func TestDegradedModeResponse(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("local cache serves when circuit open", prop.ForAll(
		func(key, value string) bool {
			if key == "" || value == "" {
				return true
			}

			cbConfig := fault.NewCircuitBreakerConfig("test",
				fault.WithFailureThreshold(1),
				fault.WithSuccessThreshold(1),
				fault.WithCircuitTimeout(time.Hour),
			)
			cb, _ := fault.NewCircuitBreaker(cbConfig)

			tripCircuit(cb, 1)

			lc := localcache.New(localcache.Config{
				MaxSize:    1000,
				DefaultTTL: time.Minute,
			})
			lc.Set(key, []byte(value), time.Minute)

			if cb.State() != fault.StateOpen {
				return false
			}

			result, found := lc.Get(key)
			return found && string(result) == value
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 100 }),
	))

	properties.Property("circuit breaker state transitions", prop.ForAll(
		func(failures int) bool {
			threshold := 3
			if failures < 0 {
				failures = 0
			}
			if failures > 10 {
				failures = 10
			}

			cbConfig := fault.NewCircuitBreakerConfig("test",
				fault.WithFailureThreshold(threshold),
				fault.WithSuccessThreshold(1),
				fault.WithCircuitTimeout(time.Hour),
			)
			cb, _ := fault.NewCircuitBreaker(cbConfig)

			tripCircuit(cb, failures)

			if failures >= threshold {
				return cb.State() == fault.StateOpen
			}
			return cb.State() == fault.StateClosed
		},
		gen.IntRange(0, 10),
	))

	properties.Property("local cache data persists in degraded mode", prop.ForAll(
		func(entries map[string]string) bool {
			if len(entries) == 0 {
				return true
			}

			lc := localcache.New(localcache.Config{
				MaxSize:    1000,
				DefaultTTL: time.Minute,
			})

			for k, v := range entries {
				if k != "" && v != "" {
					lc.Set(k, []byte(v), time.Minute)
				}
			}

			cbConfig := fault.NewCircuitBreakerConfig("test",
				fault.WithFailureThreshold(1),
				fault.WithSuccessThreshold(1),
				fault.WithCircuitTimeout(time.Hour),
			)
			cb, _ := fault.NewCircuitBreaker(cbConfig)
			tripCircuit(cb, 1)

			for k, v := range entries {
				if k == "" || v == "" {
					continue
				}
				result, found := lc.Get(k)
				if !found || string(result) != v {
					return false
				}
			}
			return true
		},
		gen.MapOf(
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		).SuchThat(func(m map[string]string) bool { return len(m) <= 20 }),
	))

	properties.TestingRun(t)
}
