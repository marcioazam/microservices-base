package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
)

// SlidingWindow implements the sliding window rate limiting algorithm.
type SlidingWindow struct {
	mu           sync.Mutex
	limit        int
	window       time.Duration
	requests     []time.Time
	eventEmitter domain.EventEmitter
}

// SlidingWindowConfig holds sliding window configuration.
type SlidingWindowConfig struct {
	Limit        int           // Maximum requests per window
	Window       time.Duration // Time window
	EventEmitter domain.EventEmitter
}

// NewSlidingWindow creates a new sliding window rate limiter.
func NewSlidingWindow(cfg SlidingWindowConfig) *SlidingWindow {
	return &SlidingWindow{
		limit:        cfg.Limit,
		window:       cfg.Window,
		requests:     make([]time.Time, 0),
		eventEmitter: cfg.EventEmitter,
	}
}

// Allow checks if a request should be allowed.
func (sw *SlidingWindow) Allow(ctx context.Context, key string) (domain.RateLimitDecision, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	sw.pruneOldRequests(now)

	decision := domain.RateLimitDecision{
		Limit:     sw.limit,
		Remaining: sw.limit - len(sw.requests),
		ResetAt:   sw.calculateResetTime(now),
	}

	if len(sw.requests) < sw.limit {
		sw.requests = append(sw.requests, now)
		decision.Allowed = true
		decision.Remaining = sw.limit - len(sw.requests)
	} else {
		decision.Allowed = false
		decision.RetryAfter = sw.calculateRetryAfter(now)
		sw.emitRateLimitEvent(key, decision)
	}

	return decision, nil
}

// GetHeaders returns rate limit headers for response.
func (sw *SlidingWindow) GetHeaders(ctx context.Context, key string) (domain.RateLimitHeaders, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	sw.pruneOldRequests(now)

	return domain.RateLimitHeaders{
		Limit:     sw.limit,
		Remaining: sw.limit - len(sw.requests),
		Reset:     sw.calculateResetTime(now).Unix(),
	}, nil
}

// pruneOldRequests removes requests outside the window. Must be called with lock held.
func (sw *SlidingWindow) pruneOldRequests(now time.Time) {
	windowStart := now.Add(-sw.window)

	// Find first request within window
	firstValid := 0
	for i, t := range sw.requests {
		if t.After(windowStart) {
			firstValid = i
			break
		}
		firstValid = i + 1
	}

	// Remove old requests
	if firstValid > 0 {
		sw.requests = sw.requests[firstValid:]
	}
}

// calculateResetTime returns when the oldest request will expire.
func (sw *SlidingWindow) calculateResetTime(now time.Time) time.Time {
	if len(sw.requests) == 0 {
		return now
	}

	// Oldest request will expire at its timestamp + window
	return sw.requests[0].Add(sw.window)
}

// calculateRetryAfter returns how long to wait for a slot.
func (sw *SlidingWindow) calculateRetryAfter(now time.Time) time.Duration {
	if len(sw.requests) == 0 {
		return 0
	}

	// Wait until oldest request expires
	oldestExpiry := sw.requests[0].Add(sw.window)
	if oldestExpiry.After(now) {
		return oldestExpiry.Sub(now)
	}

	return 0
}

// emitRateLimitEvent emits a rate limit event.
func (sw *SlidingWindow) emitRateLimitEvent(key string, decision domain.RateLimitDecision) {
	if sw.eventEmitter == nil {
		return
	}

	event := domain.ResilienceEvent{
		ID:        generateEventID(),
		Type:      domain.EventRateLimitHit,
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"key":         key,
			"allowed":     decision.Allowed,
			"remaining":   decision.Remaining,
			"retry_after": decision.RetryAfter.String(),
		},
	}

	sw.eventEmitter.Emit(event)
}

// GetRequestCount returns current request count within window (for testing).
func (sw *SlidingWindow) GetRequestCount() int {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.pruneOldRequests(time.Now())
	return len(sw.requests)
}

// GetRequests returns timestamps of requests within window (for testing).
func (sw *SlidingWindow) GetRequests() []time.Time {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.pruneOldRequests(time.Now())
	result := make([]time.Time, len(sw.requests))
	copy(result, sw.requests)
	return result
}
