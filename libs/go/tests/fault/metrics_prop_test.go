package fault_test

import (
	"context"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/fault"
	"pgregory.net/rapid"
)

// Property: ExecutionMetrics builder methods are immutable
func TestExecutionMetricsBuilderImmutability(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policyName := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "policy_name")
		durationMs := rapid.Int64Range(1, 10000).Draw(t, "duration_ms")
		success := rapid.Bool().Draw(t, "success")

		duration := time.Duration(durationMs) * time.Millisecond
		original := fault.NewExecutionMetrics(policyName, duration, success)

		// Apply builder methods
		withCircuit := original.WithCircuitState("open")
		withRetry := original.WithRetryAttempts(3)

		// Original should be unchanged
		if original.CircuitState != "" {
			t.Errorf("Original CircuitState modified: got %s", original.CircuitState)
		}
		if original.RetryAttempts != 0 {
			t.Errorf("Original RetryAttempts modified: got %d", original.RetryAttempts)
		}

		// New instances should have values
		if withCircuit.CircuitState != "open" {
			t.Errorf("WithCircuitState failed: got %s", withCircuit.CircuitState)
		}
		if withRetry.RetryAttempts != 3 {
			t.Errorf("WithRetryAttempts failed: got %d", withRetry.RetryAttempts)
		}
	})
}

// Property: ExecutionMetrics preserves required fields
func TestExecutionMetricsPreservesFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policyName := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "policy_name")
		durationMs := rapid.Int64Range(1, 10000).Draw(t, "duration_ms")
		success := rapid.Bool().Draw(t, "success")

		duration := time.Duration(durationMs) * time.Millisecond
		metrics := fault.NewExecutionMetrics(policyName, duration, success)

		if metrics.PolicyName != policyName {
			t.Errorf("PolicyName = %s, want %s", metrics.PolicyName, policyName)
		}
		if metrics.ExecutionTime != duration {
			t.Errorf("ExecutionTime = %v, want %v", metrics.ExecutionTime, duration)
		}
		if metrics.Success != success {
			t.Errorf("Success = %v, want %v", metrics.Success, success)
		}
		if metrics.Timestamp.IsZero() {
			t.Error("Timestamp should not be zero")
		}
	})
}

// Property: IsSuccessful is consistent with Success field
func TestExecutionMetricsIsSuccessfulConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policyName := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "policy_name")
		success := rapid.Bool().Draw(t, "success")

		metrics := fault.NewExecutionMetrics(policyName, time.Second, success)

		if metrics.IsSuccessful() != success {
			t.Errorf("IsSuccessful() = %v, want %v", metrics.IsSuccessful(), success)
		}
	})
}

// Property: WasRetried is consistent with RetryAttempts
func TestExecutionMetricsWasRetriedConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policyName := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "policy_name")
		retryAttempts := rapid.IntRange(0, 10).Draw(t, "retry_attempts")

		metrics := fault.NewExecutionMetrics(policyName, time.Second, true).
			WithRetryAttempts(retryAttempts)

		expectedWasRetried := retryAttempts > 0
		if metrics.WasRetried() != expectedWasRetried {
			t.Errorf("WasRetried() = %v, want %v (attempts=%d)",
				metrics.WasRetried(), expectedWasRetried, retryAttempts)
		}
	})
}

// Property: WasRateLimited is consistent with RateLimited field
func TestExecutionMetricsWasRateLimitedConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policyName := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "policy_name")
		rateLimited := rapid.Bool().Draw(t, "rate_limited")

		metrics := fault.NewExecutionMetrics(policyName, time.Second, true).
			WithRateLimit(rateLimited)

		if metrics.WasRateLimited() != rateLimited {
			t.Errorf("WasRateLimited() = %v, want %v", metrics.WasRateLimited(), rateLimited)
		}
	})
}

// Property: Builder chain preserves all values
func TestExecutionMetricsBuilderChain(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		policyName := rapid.StringMatching(`[a-z]{5,15}`).Draw(rt, "policy_name")
		circuitState := rapid.SampledFrom([]string{"closed", "open", "half-open"}).Draw(rt, "circuit_state")
		retryAttempts := rapid.IntRange(0, 10).Draw(rt, "retry_attempts")
		rateLimited := rapid.Bool().Draw(rt, "rate_limited")
		bulkheadQueued := rapid.Bool().Draw(rt, "bulkhead_queued")
		correlationID := rapid.StringMatching(`[a-f0-9]{32}`).Draw(rt, "correlation_id")
		traceID := rapid.StringMatching(`[a-f0-9]{32}`).Draw(rt, "trace_id")

		metrics := fault.NewExecutionMetrics(policyName, time.Second, true).
			WithCircuitState(circuitState).
			WithRetryAttempts(retryAttempts).
			WithRateLimit(rateLimited).
			WithBulkheadQueue(bulkheadQueued).
			WithCorrelationID(correlationID).
			WithTraceID(traceID)

		if metrics.CircuitState != circuitState {
			rt.Errorf("CircuitState = %s, want %s", metrics.CircuitState, circuitState)
		}
		if metrics.RetryAttempts != retryAttempts {
			rt.Errorf("RetryAttempts = %d, want %d", metrics.RetryAttempts, retryAttempts)
		}
		if metrics.RateLimited != rateLimited {
			rt.Errorf("RateLimited = %v, want %v", metrics.RateLimited, rateLimited)
		}
		if metrics.BulkheadQueued != bulkheadQueued {
			rt.Errorf("BulkheadQueued = %v, want %v", metrics.BulkheadQueued, bulkheadQueued)
		}
		if metrics.CorrelationID != correlationID {
			rt.Errorf("CorrelationID = %s, want %s", metrics.CorrelationID, correlationID)
		}
		if metrics.TraceID != traceID {
			rt.Errorf("TraceID = %s, want %s", metrics.TraceID, traceID)
		}
	})
}

// Property: NoOpMetricsRecorder does not panic
func TestNoOpMetricsRecorderNoPanic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		recorder := fault.NoOpMetricsRecorder{}
		ctx := context.Background()

		policyName := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "policy_name")
		metrics := fault.NewExecutionMetrics(policyName, time.Second, true)

		// None of these should panic
		recorder.RecordExecution(ctx, metrics)
		recorder.RecordCircuitState(ctx, policyName, "open")
		recorder.RecordRetryAttempt(ctx, policyName, 1)
		recorder.RecordRateLimit(ctx, policyName, true)
		recorder.RecordBulkheadQueue(ctx, policyName, true)
	})
}

// Property: Timestamp is always in the past or present
func TestExecutionMetricsTimestampNotFuture(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policyName := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "policy_name")

		before := time.Now().UTC()
		metrics := fault.NewExecutionMetrics(policyName, time.Second, true)
		after := time.Now().UTC()

		if metrics.Timestamp.Before(before) {
			t.Errorf("Timestamp %v is before creation time %v", metrics.Timestamp, before)
		}
		if metrics.Timestamp.After(after) {
			t.Errorf("Timestamp %v is after check time %v", metrics.Timestamp, after)
		}
	})
}
