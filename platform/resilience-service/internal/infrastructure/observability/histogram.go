// Package observability provides OpenTelemetry-based observability implementations.
package observability

import (
	"sort"
	"sync"
	"time"
)

// DefaultLatencyBuckets defines standard latency buckets in seconds.
var DefaultLatencyBuckets = []float64{
	0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
}

// Histogram tracks latency distributions for percentile calculations.
type Histogram struct {
	mu      sync.RWMutex
	buckets []float64
	counts  []uint64
	sum     float64
	count   uint64
}

// NewHistogram creates a new histogram with the given buckets.
func NewHistogram(buckets []float64) *Histogram {
	sorted := make([]float64, len(buckets))
	copy(sorted, buckets)
	sort.Float64s(sorted)

	return &Histogram{
		buckets: sorted,
		counts:  make([]uint64, len(sorted)+1),
	}
}

// Observe records a value in the histogram.
func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.sum += value
	h.count++

	for i, bound := range h.buckets {
		if value <= bound {
			h.counts[i]++
			return
		}
	}
	h.counts[len(h.buckets)]++
}

// ObserveDuration records a duration in seconds.
func (h *Histogram) ObserveDuration(d time.Duration) {
	h.Observe(d.Seconds())
}

// HistogramData holds histogram data for export.
type HistogramData struct {
	Buckets     []float64
	BucketCount []uint64
	Sum         float64
	Count       uint64
}

// Data returns a copy of the histogram data.
func (h *Histogram) Data() HistogramData {
	h.mu.RLock()
	defer h.mu.RUnlock()

	buckets := make([]float64, len(h.buckets))
	copy(buckets, h.buckets)

	counts := make([]uint64, len(h.counts))
	copy(counts, h.counts)

	return HistogramData{
		Buckets:     buckets,
		BucketCount: counts,
		Sum:         h.sum,
		Count:       h.count,
	}
}

// Percentile calculates the approximate percentile (0-100).
func (h *Histogram) Percentile(p float64) float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.count == 0 {
		return 0
	}

	threshold := uint64(float64(h.count) * p / 100.0)
	cumulative := uint64(0)

	for i, count := range h.counts {
		cumulative += count
		if cumulative >= threshold {
			if i < len(h.buckets) {
				return h.buckets[i]
			}
			if len(h.buckets) > 0 {
				return h.buckets[len(h.buckets)-1]
			}
			return 0
		}
	}

	return 0
}

// LatencyHistograms holds histograms for different operations.
type LatencyHistograms struct {
	mu         sync.RWMutex
	histograms map[string]*Histogram
	buckets    []float64
}

// NewLatencyHistograms creates a new latency histogram collection.
func NewLatencyHistograms() *LatencyHistograms {
	return &LatencyHistograms{
		histograms: make(map[string]*Histogram),
		buckets:    DefaultLatencyBuckets,
	}
}

// getOrCreate returns existing histogram or creates new one.
func (l *LatencyHistograms) getOrCreate(name string) *Histogram {
	l.mu.Lock()
	defer l.mu.Unlock()

	if h, ok := l.histograms[name]; ok {
		return h
	}

	h := NewHistogram(l.buckets)
	l.histograms[name] = h
	return h
}

// Observe records a latency observation.
func (l *LatencyHistograms) Observe(name string, duration time.Duration) {
	h := l.getOrCreate(name)
	h.ObserveDuration(duration)
}

// GetPercentiles returns p50, p95, p99 for a histogram.
func (l *LatencyHistograms) GetPercentiles(name string) (p50, p95, p99 float64) {
	l.mu.RLock()
	h, ok := l.histograms[name]
	l.mu.RUnlock()

	if !ok {
		return 0, 0, 0
	}

	return h.Percentile(50), h.Percentile(95), h.Percentile(99)
}

// GetAllData returns all histogram data.
func (l *LatencyHistograms) GetAllData() map[string]HistogramData {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make(map[string]HistogramData, len(l.histograms))
	for name, h := range l.histograms {
		result[name] = h.Data()
	}
	return result
}

// Names returns all histogram names sorted alphabetically.
func (l *LatencyHistograms) Names() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.histograms))
	for name := range l.histograms {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
