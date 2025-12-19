// Package testutil provides tests for generators.
package testutil

import (
	"testing"
	"time"
)

func TestIntGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := IntGen(1, 10)

	for i := 0; i < 100; i++ {
		v := gen(rng)
		if v < 1 || v > 10 {
			t.Errorf("value %d out of range [1, 10]", v)
		}
	}
}

func TestInt64Gen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := Int64Gen(100, 200)

	for i := 0; i < 100; i++ {
		v := gen(rng)
		if v < 100 || v > 200 {
			t.Errorf("value %d out of range [100, 200]", v)
		}
	}
}

func TestFloat64Gen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := Float64Gen(0.0, 1.0)

	for i := 0; i < 100; i++ {
		v := gen(rng)
		if v < 0.0 || v >= 1.0 {
			t.Errorf("value %f out of range [0.0, 1.0)", v)
		}
	}
}

func TestBoolGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := BoolGen()

	trueCount := 0
	for i := 0; i < 100; i++ {
		if gen(rng) {
			trueCount++
		}
	}

	// Should have some of each
	if trueCount == 0 || trueCount == 100 {
		t.Error("expected mix of true and false")
	}
}

func TestStringGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := StringGen(10)

	for i := 0; i < 100; i++ {
		s := gen(rng)
		if len(s) != 10 {
			t.Errorf("expected length 10, got %d", len(s))
		}
	}
}

func TestStringRangeGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := StringRangeGen(5, 15)

	for i := 0; i < 100; i++ {
		s := gen(rng)
		if len(s) < 5 || len(s) > 15 {
			t.Errorf("length %d out of range [5, 15]", len(s))
		}
	}
}

func TestSliceGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := SliceGen(3, 7, IntGen(1, 100))

	for i := 0; i < 100; i++ {
		slice := gen(rng)
		if len(slice) < 3 || len(slice) > 7 {
			t.Errorf("length %d out of range [3, 7]", len(slice))
		}
	}
}

func TestMapGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := MapGen(2, 5, StringGen(5), IntGen(1, 100))

	for i := 0; i < 100; i++ {
		m := gen(rng)
		if len(m) < 1 || len(m) > 5 { // May have fewer due to key collisions
			t.Errorf("size %d unexpected", len(m))
		}
	}
}

func TestOneOf(t *testing.T) {
	rng := NewSeededRand(42)
	gen := OneOf("a", "b", "c")

	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		seen[gen(rng)] = true
	}

	if len(seen) != 3 {
		t.Errorf("expected to see all 3 choices, saw %d", len(seen))
	}
}

func TestDurationGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := DurationGen(time.Second, time.Minute)

	for i := 0; i < 100; i++ {
		d := gen(rng)
		if d < time.Second || d >= time.Minute {
			t.Errorf("duration %v out of range", d)
		}
	}
}

func TestHealthStatusGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := HealthStatusGen()

	seen := make(map[HealthStatus]bool)
	for i := 0; i < 100; i++ {
		seen[gen(rng)] = true
	}

	if len(seen) != 3 {
		t.Errorf("expected to see all 3 statuses, saw %d", len(seen))
	}
}

func TestCircuitBreakerConfigGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := CircuitBreakerConfigGen()

	for i := 0; i < 100; i++ {
		cfg := gen(rng)
		if cfg.FailureThreshold < 1 || cfg.FailureThreshold > 10 {
			t.Errorf("FailureThreshold %d out of range", cfg.FailureThreshold)
		}
		if cfg.SuccessThreshold < 1 || cfg.SuccessThreshold > 5 {
			t.Errorf("SuccessThreshold %d out of range", cfg.SuccessThreshold)
		}
		if cfg.Timeout < time.Second || cfg.Timeout > time.Minute {
			t.Errorf("Timeout %v out of range", cfg.Timeout)
		}
	}
}

func TestRetryConfigGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := RetryConfigGen()

	for i := 0; i < 100; i++ {
		cfg := gen(rng)
		if cfg.MaxAttempts < 1 || cfg.MaxAttempts > 10 {
			t.Errorf("MaxAttempts %d out of range", cfg.MaxAttempts)
		}
		if cfg.BaseDelay > cfg.MaxDelay {
			t.Errorf("BaseDelay %v > MaxDelay %v", cfg.BaseDelay, cfg.MaxDelay)
		}
		if cfg.Multiplier < 1.5 || cfg.Multiplier > 3.0 {
			t.Errorf("Multiplier %f out of range", cfg.Multiplier)
		}
		if cfg.Jitter < 0.0 || cfg.Jitter > 0.5 {
			t.Errorf("Jitter %f out of range", cfg.Jitter)
		}
	}
}

func TestRateLimitConfigGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := RateLimitConfigGen()

	for i := 0; i < 100; i++ {
		cfg := gen(rng)
		if cfg.Limit < 1 || cfg.Limit > 1000 {
			t.Errorf("Limit %d out of range", cfg.Limit)
		}
		if cfg.BurstSize < 1 {
			t.Errorf("BurstSize %d should be positive", cfg.BurstSize)
		}
	}
}

func TestBulkheadConfigGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := BulkheadConfigGen()

	for i := 0; i < 100; i++ {
		cfg := gen(rng)
		if cfg.MaxConcurrent < 1 || cfg.MaxConcurrent > 100 {
			t.Errorf("MaxConcurrent %d out of range", cfg.MaxConcurrent)
		}
		if cfg.MaxQueue < 0 || cfg.MaxQueue > 1000 {
			t.Errorf("MaxQueue %d out of range", cfg.MaxQueue)
		}
	}
}

func TestResiliencePolicyGen(t *testing.T) {
	rng := NewSeededRand(42)
	gen := ResiliencePolicyGen()

	for i := 0; i < 100; i++ {
		policy := gen(rng)
		if policy.Name == "" {
			t.Error("Name should not be empty")
		}
		if policy.CircuitBreaker == nil {
			t.Error("CircuitBreaker should not be nil")
		}
		if policy.Retry == nil {
			t.Error("Retry should not be nil")
		}
		if policy.RateLimit == nil {
			t.Error("RateLimit should not be nil")
		}
		if policy.Bulkhead == nil {
			t.Error("Bulkhead should not be nil")
		}
		if policy.Timeout == nil {
			t.Error("Timeout should not be nil")
		}
	}
}

func TestSample(t *testing.T) {
	samples := Sample(IntGen(1, 100), 10)
	if len(samples) != 10 {
		t.Errorf("expected 10 samples, got %d", len(samples))
	}
}

func TestSampleSeeded(t *testing.T) {
	samples1 := SampleSeeded(IntGen(1, 100), 10, 42)
	samples2 := SampleSeeded(IntGen(1, 100), 10, 42)

	for i := range samples1 {
		if samples1[i] != samples2[i] {
			t.Error("seeded samples should be identical")
		}
	}
}

func TestFilter(t *testing.T) {
	rng := NewSeededRand(42)
	gen := Filter(IntGen(1, 100), func(n int) bool { return n%2 == 0 }, 100)

	for i := 0; i < 100; i++ {
		v := gen(rng)
		if v%2 != 0 {
			t.Errorf("expected even number, got %d", v)
		}
	}
}

func TestMap(t *testing.T) {
	rng := NewSeededRand(42)
	gen := Map(IntGen(1, 10), func(n int) string {
		return string(rune('a' + n - 1))
	})

	for i := 0; i < 100; i++ {
		s := gen(rng)
		if len(s) != 1 || s[0] < 'a' || s[0] > 'j' {
			t.Errorf("unexpected value: %s", s)
		}
	}
}

func TestConstant(t *testing.T) {
	rng := NewSeededRand(42)
	gen := Constant(42)

	for i := 0; i < 100; i++ {
		if gen(rng) != 42 {
			t.Error("expected constant 42")
		}
	}
}

func TestNullable(t *testing.T) {
	rng := NewSeededRand(42)
	gen := Nullable(IntGen(1, 100), 0.5)

	zeroCount := 0
	for i := 0; i < 100; i++ {
		if gen(rng) == 0 {
			zeroCount++
		}
	}

	// Should have some zeros and some non-zeros
	if zeroCount == 0 || zeroCount == 100 {
		t.Error("expected mix of zero and non-zero values")
	}
}
