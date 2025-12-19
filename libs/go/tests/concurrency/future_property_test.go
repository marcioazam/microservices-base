package concurrency_test

import (
	"context"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/concurrency"
	"github.com/authcorp/libs/go/src/functional"
	"pgregory.net/rapid"
)

// Property 18: Future Context Cancellation
// Cancelled context causes WaitContext to return error.
func TestProperty_FutureContextCancellation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create a slow future
		future := concurrency.NewFuture(func() (int, error) {
			time.Sleep(time.Second * 10)
			return 42, nil
		})

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// WaitContext should return context error
		result := future.WaitContext(ctx)

		if result.IsOk() {
			t.Fatalf("Should return error on cancelled context")
		}

		if result.UnwrapErr() != context.Canceled {
			t.Fatalf("Expected context.Canceled, got %v", result.UnwrapErr())
		}
	})
}

// Property 19: Future Result Integration
// Future.Result() returns Some when done, None when pending.
func TestProperty_FutureResultIntegration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")

		// Completed future
		future := concurrency.Resolve(value)

		// Should be done immediately
		if !future.IsDone() {
			t.Fatalf("Resolved future should be done")
		}

		// Result should be Some
		opt := future.Result()
		if !opt.IsSome() {
			t.Fatalf("Done future should have Some result")
		}

		// Result should be Ok with correct value
		result := opt.Unwrap()
		if !result.IsOk() {
			t.Fatalf("Resolved future should be Ok")
		}
		if result.Unwrap() != value {
			t.Fatalf("Expected %d, got %d", value, result.Unwrap())
		}
	})
}

// Property: Resolve creates completed Ok future
func TestProperty_ResolveCreatesOk(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")

		future := concurrency.Resolve(value)
		result := future.Wait()

		if !result.IsOk() {
			t.Fatalf("Resolve should create Ok future")
		}
		if result.Unwrap() != value {
			t.Fatalf("Value mismatch")
		}
	})
}

// Property: Reject creates completed Err future
func TestProperty_RejectCreatesErr(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		errMsg := rapid.String().Draw(t, "errMsg")
		err := functional.NewError(errMsg)

		future := concurrency.Reject[int](err)
		result := future.Wait()

		if !result.IsErr() {
			t.Fatalf("Reject should create Err future")
		}
		if result.UnwrapErr().Error() != errMsg {
			t.Fatalf("Error message mismatch")
		}
	})
}

// Property: All waits for all futures
func TestProperty_AllWaitsForAll(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(1, 5).Draw(t, "count")
		values := make([]int, count)
		futures := make([]*concurrency.Future[int], count)

		for i := 0; i < count; i++ {
			values[i] = rapid.Int().Draw(t, "value")
			futures[i] = concurrency.Resolve(values[i])
		}

		results := concurrency.All(futures...)

		if len(results) != count {
			t.Fatalf("Expected %d results, got %d", count, len(results))
		}

		for i, result := range results {
			if !result.IsOk() {
				t.Fatalf("Result %d should be Ok", i)
			}
			if result.Unwrap() != values[i] {
				t.Fatalf("Value mismatch at %d", i)
			}
		}
	})
}

// Property: Map transforms future value
func TestProperty_FutureMap(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		multiplier := rapid.IntRange(1, 10).Draw(t, "multiplier")

		future := concurrency.Resolve(value)
		mapped := concurrency.Map(future, func(v int) int { return v * multiplier })

		result := mapped.Wait()

		if !result.IsOk() {
			t.Fatalf("Mapped future should be Ok")
		}
		if result.Unwrap() != value*multiplier {
			t.Fatalf("Expected %d, got %d", value*multiplier, result.Unwrap())
		}
	})
}
