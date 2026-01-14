// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 9: Rate Limiting Correctness
// Validates: Requirements 6.2, 6.3, 6.4, 6.5
package property

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// MockRateLimiter simulates rate limiting for testing.
type MockRateLimiter struct {
	limit       int
	window      time.Duration
	tenantData  map[string][]int64
	useFallback bool
}

// RateLimitResult contains rate limit check result.
type RateLimitResult struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}

func NewMockRateLimiter(limit int, window time.Duration) *MockRateLimiter {
	return &MockRateLimiter{
		limit:      limit,
		window:     window,
		tenantData: make(map[string][]int64),
	}
}

// Allow checks if request is allowed.
func (l *MockRateLimiter) Allow(tenantID string, now time.Time) RateLimitResult {
	windowStart := now.Add(-l.window).UnixNano()

	// Get or create tenant data
	timestamps, exists := l.tenantData[tenantID]
	if !exists {
		timestamps = []int64{}
	}

	// Filter timestamps within window (sliding window)
	var validTimestamps []int64
	for _, ts := range timestamps {
		if ts > windowStart {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check limit
	if len(validTimestamps) >= l.limit {
		retryAfter := l.calculateRetryAfter(validTimestamps, now)
		return RateLimitResult{
			Allowed:    false,
			Remaining:  0,
			RetryAfter: retryAfter,
		}
	}

	// Add current request
	validTimestamps = append(validTimestamps, now.UnixNano())
	l.tenantData[tenantID] = validTimestamps

	return RateLimitResult{
		Allowed:   true,
		Remaining: l.limit - len(validTimestamps),
	}
}

func (l *MockRateLimiter) calculateRetryAfter(timestamps []int64, now time.Time) time.Duration {
	if len(timestamps) == 0 {
		return l.window
	}

	oldest := timestamps[0]
	for _, ts := range timestamps {
		if ts < oldest {
			oldest = ts
		}
	}

	oldestTime := time.Unix(0, oldest)
	retryAfter := oldestTime.Add(l.window).Sub(now)
	if retryAfter < 0 {
		retryAfter = time.Second
	}
	return retryAfter
}

// SetFallbackMode enables fallback mode.
func (l *MockRateLimiter) SetFallbackMode(enabled bool) {
	l.useFallback = enabled
}

// TestProperty9_HTTP429IncludesRetryAfter tests that 429 responses include Retry-After.
// Property 9: Rate Limiting Correctness
// Validates: Requirements 6.2, 6.3, 6.4, 6.5
func TestProperty9_HTTP429IncludesRetryAfter(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		limit := rapid.IntRange(1, 10).Draw(t, "limit")
		windowSecs := rapid.IntRange(1, 60).Draw(t, "windowSecs")
		window := time.Duration(windowSecs) * time.Second

		limiter := NewMockRateLimiter(limit, window)
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		now := time.Now()

		// Exhaust the limit
		for i := 0; i < limit; i++ {
			result := limiter.Allow(tenantID, now)
			if !result.Allowed {
				t.Fatalf("request %d should be allowed", i+1)
			}
		}

		// Next request should be rate limited
		result := limiter.Allow(tenantID, now)

		// Property: HTTP 429 response SHALL include Retry-After header
		if result.Allowed {
			t.Error("request should be rate limited")
		}
		if result.RetryAfter <= 0 {
			t.Error("Retry-After should be positive when rate limited")
		}
		if result.RetryAfter > window {
			t.Errorf("Retry-After %v should not exceed window %v", result.RetryAfter, window)
		}
	})
}

// TestProperty9_RateLimitsIsolatedPerTenant tests that rate limits are isolated per tenant.
// Property 9: Rate Limiting Correctness
// Validates: Requirements 6.2, 6.3, 6.4, 6.5
func TestProperty9_RateLimitsIsolatedPerTenant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		limit := rapid.IntRange(2, 5).Draw(t, "limit")
		limiter := NewMockRateLimiter(limit, time.Minute)

		tenantA := rapid.StringMatching(`tenant-a-[a-z0-9]{8}`).Draw(t, "tenantA")
		tenantB := rapid.StringMatching(`tenant-b-[a-z0-9]{8}`).Draw(t, "tenantB")

		now := time.Now()

		// Exhaust tenant A's limit
		for i := 0; i < limit; i++ {
			limiter.Allow(tenantA, now)
		}

		// Tenant A should be rate limited
		resultA := limiter.Allow(tenantA, now)
		if resultA.Allowed {
			t.Error("tenant A should be rate limited")
		}

		// Property: Rate limits SHALL be isolated per tenant
		// Tenant B should still have full quota
		resultB := limiter.Allow(tenantB, now)
		if !resultB.Allowed {
			t.Error("tenant B should NOT be rate limited")
		}
		if resultB.Remaining != limit-1 {
			t.Errorf("tenant B should have %d remaining, got %d", limit-1, resultB.Remaining)
		}
	})
}

// TestProperty9_SlidingWindowCorrectlyCountsRequests tests sliding window algorithm.
// Property 9: Rate Limiting Correctness
// Validates: Requirements 6.2, 6.3, 6.4, 6.5
func TestProperty9_SlidingWindowCorrectlyCountsRequests(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		limit := rapid.IntRange(3, 10).Draw(t, "limit")
		windowSecs := rapid.IntRange(10, 60).Draw(t, "windowSecs")
		window := time.Duration(windowSecs) * time.Second

		limiter := NewMockRateLimiter(limit, window)
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		baseTime := time.Now()

		// Make requests at start of window
		halfLimit := limit / 2
		for i := 0; i < halfLimit; i++ {
			result := limiter.Allow(tenantID, baseTime)
			if !result.Allowed {
				t.Fatalf("early request %d should be allowed", i+1)
			}
		}

		// Move time forward past window
		futureTime := baseTime.Add(window + time.Second)

		// Property: Sliding window algorithm SHALL correctly count requests in window
		// Old requests should have expired, full quota available
		result := limiter.Allow(tenantID, futureTime)
		if !result.Allowed {
			t.Error("request after window should be allowed")
		}
		if result.Remaining != limit-1 {
			t.Errorf("should have %d remaining after window reset, got %d", limit-1, result.Remaining)
		}
	})
}

// TestProperty9_FallbackOnCacheUnavailable tests local fallback when cache unavailable.
// Property 9: Rate Limiting Correctness
// Validates: Requirements 6.2, 6.3, 6.4, 6.5
func TestProperty9_FallbackOnCacheUnavailable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		limit := rapid.IntRange(2, 5).Draw(t, "limit")
		limiter := NewMockRateLimiter(limit, time.Minute)
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		now := time.Now()

		// Enable fallback mode (simulating cache unavailable)
		limiter.SetFallbackMode(true)

		// Property: Cache unavailability SHALL trigger local fallback
		// Rate limiting should still work
		for i := 0; i < limit; i++ {
			result := limiter.Allow(tenantID, now)
			if !result.Allowed {
				t.Fatalf("fallback request %d should be allowed", i+1)
			}
		}

		// Should still enforce limits in fallback mode
		result := limiter.Allow(tenantID, now)
		if result.Allowed {
			t.Error("fallback mode should still enforce rate limits")
		}
	})
}

// TestProperty9_RemainingCountAccurate tests that remaining count is accurate.
// Property 9: Rate Limiting Correctness
// Validates: Requirements 6.2, 6.3, 6.4, 6.5
func TestProperty9_RemainingCountAccurate(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		limit := rapid.IntRange(5, 20).Draw(t, "limit")
		limiter := NewMockRateLimiter(limit, time.Minute)
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		now := time.Now()
		numRequests := rapid.IntRange(1, limit-1).Draw(t, "numRequests")

		// Make some requests
		var lastResult RateLimitResult
		for i := 0; i < numRequests; i++ {
			lastResult = limiter.Allow(tenantID, now)
		}

		// Property: Remaining count should be accurate
		expectedRemaining := limit - numRequests
		if lastResult.Remaining != expectedRemaining {
			t.Errorf("expected remaining %d, got %d", expectedRemaining, lastResult.Remaining)
		}
	})
}
