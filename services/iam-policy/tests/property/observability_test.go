// Package property contains property-based tests for IAM Policy Service.
package property

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/observability"
	"github.com/auth-platform/iam-policy-service/tests/testutil"
	"pgregory.net/rapid"
)

// TestObservabilityTraceContextPropagation validates Property 10.
// Trace context must be propagated correctly.
func TestObservabilityTraceContextPropagation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		traceID := testutil.TraceIDGen().Draw(t, "traceID")
		spanID := testutil.SpanIDGen().Draw(t, "spanID")

		// Create context with trace info
		ctx := testutil.ContextWithTrace(context.Background(), traceID, spanID)

		// Extract trace context
		extractedTraceID, extractedSpanID := testutil.ExtractTraceContext(ctx)

		// Property: trace ID must be preserved
		if extractedTraceID != traceID {
			t.Errorf("trace ID mismatch: expected %s, got %s", traceID, extractedTraceID)
		}

		// Property: span ID must be preserved
		if extractedSpanID != spanID {
			t.Errorf("span ID mismatch: expected %s, got %s", spanID, extractedSpanID)
		}
	})
}

// TestMetricsRecordingAccuracy validates Property 11.
// Metrics must be recorded accurately.
func TestMetricsRecordingAccuracy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numDecisions := rapid.IntRange(1, 100).Draw(t, "numDecisions")
		numAllowed := rapid.IntRange(0, numDecisions).Draw(t, "numAllowed")

		metrics := observability.NewMetrics()

		// Record decisions
		for i := 0; i < numAllowed; i++ {
			metrics.RecordAuthDecision(true, time.Millisecond)
		}
		for i := 0; i < numDecisions-numAllowed; i++ {
			metrics.RecordAuthDecision(false, time.Millisecond)
		}

		snapshot := metrics.GetSnapshot()

		// Property: total decisions must match
		if snapshot.AuthDecisions != int64(numDecisions) {
			t.Errorf("total decisions mismatch: expected %d, got %d", numDecisions, snapshot.AuthDecisions)
		}

		// Property: allowed count must match
		if snapshot.AuthAllowed != int64(numAllowed) {
			t.Errorf("allowed count mismatch: expected %d, got %d", numAllowed, snapshot.AuthAllowed)
		}

		// Property: denied count must match
		expectedDenied := numDecisions - numAllowed
		if snapshot.AuthDenied != int64(expectedDenied) {
			t.Errorf("denied count mismatch: expected %d, got %d", expectedDenied, snapshot.AuthDenied)
		}
	})
}

// TestCacheMetricsAccuracy validates cache metrics accuracy.
func TestCacheMetricsAccuracy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numHits := rapid.IntRange(0, 50).Draw(t, "numHits")
		numMisses := rapid.IntRange(0, 50).Draw(t, "numMisses")

		metrics := observability.NewMetrics()

		for i := 0; i < numHits; i++ {
			metrics.RecordCacheHit()
		}
		for i := 0; i < numMisses; i++ {
			metrics.RecordCacheMiss()
		}

		snapshot := metrics.GetSnapshot()

		// Property: hits must match
		if snapshot.CacheHits != int64(numHits) {
			t.Errorf("cache hits mismatch: expected %d, got %d", numHits, snapshot.CacheHits)
		}

		// Property: misses must match
		if snapshot.CacheMisses != int64(numMisses) {
			t.Errorf("cache misses mismatch: expected %d, got %d", numMisses, snapshot.CacheMisses)
		}
	})
}

// TestGRPCMetricsAccuracy validates gRPC metrics accuracy.
func TestGRPCMetricsAccuracy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		method := rapid.SampledFrom([]string{
			"/iam.IAMPolicyService/Authorize",
			"/iam.IAMPolicyService/BatchAuthorize",
			"/iam.IAMPolicyService/GetUserPermissions",
		}).Draw(t, "method")
		numRequests := rapid.IntRange(1, 50).Draw(t, "numRequests")

		metrics := observability.NewMetrics()

		for i := 0; i < numRequests; i++ {
			metrics.RecordGRPCRequest(method, time.Millisecond, nil)
		}

		snapshot := metrics.GetSnapshot()

		// Property: request count must match
		methodMetrics, ok := snapshot.GRPCMethods[method]
		if !ok {
			t.Fatalf("method %s not found in metrics", method)
		}

		if methodMetrics.Requests != int64(numRequests) {
			t.Errorf("request count mismatch: expected %d, got %d", numRequests, methodMetrics.Requests)
		}
	})
}

// TestMetricsConcurrency validates metrics under concurrent access.
func TestMetricsConcurrency(t *testing.T) {
	metrics := observability.NewMetrics()
	numGoroutines := 10
	numOpsPerGoroutine := 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOpsPerGoroutine; j++ {
				metrics.RecordAuthDecision(j%2 == 0, time.Millisecond)
				metrics.RecordCacheHit()
				metrics.RecordPolicyEvaluation()
			}
			done <- true
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	snapshot := metrics.GetSnapshot()
	expectedTotal := int64(numGoroutines * numOpsPerGoroutine)

	// Property: total must match expected
	if snapshot.AuthDecisions != expectedTotal {
		t.Errorf("auth decisions mismatch: expected %d, got %d", expectedTotal, snapshot.AuthDecisions)
	}
	if snapshot.CacheHits != expectedTotal {
		t.Errorf("cache hits mismatch: expected %d, got %d", expectedTotal, snapshot.CacheHits)
	}
	if snapshot.PolicyEvals != expectedTotal {
		t.Errorf("policy evals mismatch: expected %d, got %d", expectedTotal, snapshot.PolicyEvals)
	}
}
