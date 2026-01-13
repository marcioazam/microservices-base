// Package property contains property-based tests for the cache service.
// Feature: cache-microservice
package property

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/fault"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Property 16: Circuit Breaker Behavior
// For any sequence of Redis failures exceeding the threshold, the circuit SHALL open
// and subsequent requests SHALL fail fast without attempting Redis calls until the timeout period.
// Validates: Requirements 7.3, 7.4
func TestProperty16_CircuitBreakerBehavior(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = PropertyTestIterations
	parameters.Rng.Seed(PropertyTestSeed)

	properties := gopter.NewProperties(parameters)

	properties.Property("Circuit opens after max failures", prop.ForAll(
		func(maxFailures int) bool {
			if maxFailures < 1 || maxFailures > 20 {
				return true
			}

			cbConfig := fault.NewCircuitBreakerConfig("test",
				fault.WithFailureThreshold(maxFailures),
				fault.WithSuccessThreshold(2),
				fault.WithCircuitTimeout(time.Second),
			)
			cb, _ := fault.NewCircuitBreaker(cbConfig)
			ctx := context.Background()

			failingOp := func(ctx context.Context) error {
				return errors.New("simulated failure")
			}

			// Execute failing operations up to threshold
			for i := 0; i < maxFailures; i++ {
				_ = cb.Execute(ctx, failingOp)
			}

			// Circuit should be open now
			return cb.State() == fault.StateOpen
		},
		gen.IntRange(1, 20),
	))

	properties.Property("Open circuit rejects requests immediately", prop.ForAll(
		func(numRequests int) bool {
			if numRequests < 1 || numRequests > 50 {
				return true
			}

			cbConfig := fault.NewCircuitBreakerConfig("test",
				fault.WithFailureThreshold(2),
				fault.WithSuccessThreshold(2),
				fault.WithCircuitTimeout(time.Hour), // Long timeout to keep circuit open
			)
			cb, _ := fault.NewCircuitBreaker(cbConfig)
			ctx := context.Background()

			failingOp := func(ctx context.Context) error {
				return errors.New("simulated failure")
			}

			// Open the circuit
			for i := 0; i < 2; i++ {
				_ = cb.Execute(ctx, failingOp)
			}

			if cb.State() != fault.StateOpen {
				return false
			}

			// All subsequent requests should be rejected
			operationCalled := false
			trackedOp := func(ctx context.Context) error {
				operationCalled = true
				return nil
			}

			for i := 0; i < numRequests; i++ {
				err := cb.Execute(ctx, trackedOp)
				if err == nil {
					return false // Should have returned error
				}
			}

			// Operation should never have been called
			return !operationCalled
		},
		gen.IntRange(1, 50),
	))

	properties.Property("Circuit transitions to half-open after timeout", prop.ForAll(
		func(dummy int) bool {
			cbConfig := fault.NewCircuitBreakerConfig("test",
				fault.WithFailureThreshold(2),
				fault.WithSuccessThreshold(2),
				fault.WithCircuitTimeout(10*time.Millisecond),
			)
			cb, _ := fault.NewCircuitBreaker(cbConfig)
			ctx := context.Background()

			failingOp := func(ctx context.Context) error {
				return errors.New("simulated failure")
			}

			// Open the circuit
			for i := 0; i < 2; i++ {
				_ = cb.Execute(ctx, failingOp)
			}

			if cb.State() != fault.StateOpen {
				return false
			}

			// Wait for timeout
			time.Sleep(20 * time.Millisecond)

			// State should be half-open
			return cb.State() == fault.StateHalfOpen
		},
		gen.Const(0),
	))

	properties.Property("Successful requests in half-open close circuit", prop.ForAll(
		func(successThreshold int) bool {
			if successThreshold < 1 || successThreshold > 10 {
				return true
			}

			cbConfig := fault.NewCircuitBreakerConfig("test",
				fault.WithFailureThreshold(2),
				fault.WithSuccessThreshold(successThreshold),
				fault.WithCircuitTimeout(10*time.Millisecond),
			)
			cb, _ := fault.NewCircuitBreaker(cbConfig)
			ctx := context.Background()

			failingOp := func(ctx context.Context) error {
				return errors.New("simulated failure")
			}
			successOp := func(ctx context.Context) error {
				return nil
			}

			// Open the circuit
			for i := 0; i < 2; i++ {
				_ = cb.Execute(ctx, failingOp)
			}

			// Wait for timeout to transition to half-open
			time.Sleep(20 * time.Millisecond)

			// Execute successful operations
			for i := 0; i < successThreshold; i++ {
				_ = cb.Execute(ctx, successOp)
			}

			// Circuit should be closed
			return cb.State() == fault.StateClosed
		},
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t)
}
