package unit

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/infra/metrics"
)

func TestHistogram_Observe(t *testing.T) {
	h := metrics.NewHistogram(metrics.DefaultLatencyBuckets)

	// Observe some values
	h.Observe(0.001) // 1ms
	h.Observe(0.005) // 5ms
	h.Observe(0.050) // 50ms
	h.Observe(0.100) // 100ms
	h.Observe(1.000) // 1s

	data := h.Data()

	if data.Count != 5 {
		t.Errorf("Count: got %d, want 5", data.Count)
	}

	expectedSum := 0.001 + 0.005 + 0.050 + 0.100 + 1.000
	if data.Sum != expectedSum {
		t.Errorf("Sum: got %g, want %g", data.Sum, expectedSum)
	}
}

func TestHistogram_ObserveDuration(t *testing.T) {
	h := metrics.NewHistogram(metrics.DefaultLatencyBuckets)

	h.ObserveDuration(1 * time.Millisecond)
	h.ObserveDuration(10 * time.Millisecond)
	h.ObserveDuration(100 * time.Millisecond)

	data := h.Data()

	if data.Count != 3 {
		t.Errorf("Count: got %d, want 3", data.Count)
	}
}

func TestHistogram_Percentile(t *testing.T) {
	h := metrics.NewHistogram([]float64{0.1, 0.5, 1.0, 5.0, 10.0})

	// Add 100 values: 10 in each bucket
	for i := 0; i < 10; i++ {
		h.Observe(0.05) // <= 0.1
		h.Observe(0.3)  // <= 0.5
		h.Observe(0.7)  // <= 1.0
		h.Observe(3.0)  // <= 5.0
		h.Observe(7.0)  // <= 10.0
		h.Observe(15.0) // > 10.0 (+Inf)
	}

	// p50 should be around the 0.5 bucket
	p50 := h.Percentile(50)
	if p50 != 1.0 {
		t.Errorf("p50: got %g, want 1.0", p50)
	}

	// p95 should be in higher buckets
	p95 := h.Percentile(95)
	if p95 < 5.0 {
		t.Errorf("p95: got %g, want >= 5.0", p95)
	}
}

func TestHistogram_EmptyPercentile(t *testing.T) {
	h := metrics.NewHistogram(metrics.DefaultLatencyBuckets)

	p50 := h.Percentile(50)
	if p50 != 0 {
		t.Errorf("Empty histogram p50: got %g, want 0", p50)
	}
}

func TestLatencyHistograms_Observe(t *testing.T) {
	lh := metrics.NewLatencyHistograms()

	lh.Observe("service-a", 10*time.Millisecond)
	lh.Observe("service-a", 20*time.Millisecond)
	lh.Observe("service-b", 50*time.Millisecond)

	data := lh.GetAllData()

	if len(data) != 2 {
		t.Errorf("Expected 2 histograms, got %d", len(data))
	}

	if data["service-a"].Count != 2 {
		t.Errorf("service-a count: got %d, want 2", data["service-a"].Count)
	}

	if data["service-b"].Count != 1 {
		t.Errorf("service-b count: got %d, want 1", data["service-b"].Count)
	}
}

func TestLatencyHistograms_GetPercentiles(t *testing.T) {
	lh := metrics.NewLatencyHistograms()

	// Add many observations
	for i := 0; i < 100; i++ {
		lh.Observe("test", time.Duration(i)*time.Millisecond)
	}

	p50, p95, p99 := lh.GetPercentiles("test")

	// p50 should be around 50ms
	if p50 < 0.025 || p50 > 0.1 {
		t.Errorf("p50: got %g, expected around 0.05", p50)
	}

	// p95 should be higher than p50
	if p95 <= p50 {
		t.Errorf("p95 (%g) should be > p50 (%g)", p95, p50)
	}

	// p99 should be highest
	if p99 < p95 {
		t.Errorf("p99 (%g) should be >= p95 (%g)", p99, p95)
	}
}

func TestLatencyHistograms_NonExistent(t *testing.T) {
	lh := metrics.NewLatencyHistograms()

	p50, p95, p99 := lh.GetPercentiles("non-existent")

	if p50 != 0 || p95 != 0 || p99 != 0 {
		t.Errorf("Non-existent histogram should return 0s, got %g, %g, %g", p50, p95, p99)
	}
}

func TestLatencyHistograms_Names(t *testing.T) {
	lh := metrics.NewLatencyHistograms()

	lh.Observe("zebra", time.Millisecond)
	lh.Observe("alpha", time.Millisecond)
	lh.Observe("beta", time.Millisecond)

	names := lh.Names()

	if len(names) != 3 {
		t.Errorf("Expected 3 names, got %d", len(names))
	}

	// Should be sorted
	if names[0] != "alpha" || names[1] != "beta" || names[2] != "zebra" {
		t.Errorf("Names not sorted: %v", names)
	}
}

func TestMetrics_LatencyHistograms(t *testing.T) {
	m := metrics.NewMetrics()

	// Record some latencies
	m.ObserveCircuitBreakerLatency("service-a", 10*time.Millisecond)
	m.ObserveRetryLatency("service-b", 20*time.Millisecond)
	m.ObserveRateLimitLatency("key-1", 1*time.Millisecond)
	m.ObserveBulkheadLatency("partition-1", 5*time.Millisecond)
	m.ObserveOperationLatency("op-1", 100*time.Millisecond)

	// Get snapshot
	snapshot := m.Snapshot()

	if len(snapshot.CircuitBreakerLatency) != 1 {
		t.Errorf("Expected 1 circuit breaker histogram, got %d", len(snapshot.CircuitBreakerLatency))
	}

	if len(snapshot.RetryLatency) != 1 {
		t.Errorf("Expected 1 retry histogram, got %d", len(snapshot.RetryLatency))
	}

	// Test percentile getters
	p50, p95, p99 := m.GetCircuitBreakerPercentiles("service-a")
	if p50 == 0 && p95 == 0 && p99 == 0 {
		t.Error("Expected non-zero percentiles for circuit breaker")
	}
}

func BenchmarkHistogram_Observe(b *testing.B) {
	h := metrics.NewHistogram(metrics.DefaultLatencyBuckets)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Observe(float64(i%100) / 1000.0)
	}
}

func BenchmarkHistogram_Percentile(b *testing.B) {
	h := metrics.NewHistogram(metrics.DefaultLatencyBuckets)

	// Pre-populate
	for i := 0; i < 10000; i++ {
		h.Observe(float64(i%100) / 1000.0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Percentile(95)
	}
}

func BenchmarkLatencyHistograms_Observe(b *testing.B) {
	lh := metrics.NewLatencyHistograms()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lh.Observe("test-service", time.Duration(i%100)*time.Millisecond)
	}
}
