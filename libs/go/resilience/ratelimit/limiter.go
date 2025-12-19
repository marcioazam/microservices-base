// Package ratelimit provides generic rate limiting implementations.
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Decision represents a rate limit decision.
type Decision struct {
	Allowed    bool
	Remaining  int
	Limit      int
	ResetAt    time.Time
	RetryAfter time.Duration
}

// Headers contains rate limit response headers.
type Headers struct {
	Limit     int
	Remaining int
	Reset     int64
}

// RateLimiter is the interface for rate limiters.
type RateLimiter[K comparable] interface {
	Allow(ctx context.Context, key K) (Decision, error)
	Headers(ctx context.Context, key K) (Headers, error)
}

// TokenBucket implements a token bucket rate limiter.
type TokenBucket[K comparable] struct {
	mu         sync.Mutex
	capacity   int
	refillRate float64
	buckets    map[K]*bucket
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

// NewTokenBucket creates a new token bucket rate limiter.
func NewTokenBucket[K comparable](capacity int, refillRate float64) *TokenBucket[K] {
	return &TokenBucket[K]{
		capacity:   capacity,
		refillRate: refillRate,
		buckets:    make(map[K]*bucket),
	}
}

// Allow checks if a request is allowed.
func (tb *TokenBucket[K]) Allow(ctx context.Context, key K) (Decision, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	b := tb.getBucket(key)
	tb.refill(b)

	if b.tokens >= 1 {
		b.tokens--
		return Decision{
			Allowed:   true,
			Remaining: int(b.tokens),
			Limit:     tb.capacity,
			ResetAt:   time.Now().Add(time.Duration(float64(time.Second) / tb.refillRate)),
		}, nil
	}

	retryAfter := time.Duration(float64(time.Second) / tb.refillRate)
	return Decision{
		Allowed:    false,
		Remaining:  0,
		Limit:      tb.capacity,
		ResetAt:    time.Now().Add(retryAfter),
		RetryAfter: retryAfter,
	}, nil
}

// Headers returns rate limit headers.
func (tb *TokenBucket[K]) Headers(ctx context.Context, key K) (Headers, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	b := tb.getBucket(key)
	tb.refill(b)

	return Headers{
		Limit:     tb.capacity,
		Remaining: int(b.tokens),
		Reset:     time.Now().Add(time.Duration(float64(time.Second) / tb.refillRate)).Unix(),
	}, nil
}

func (tb *TokenBucket[K]) getBucket(key K) *bucket {
	b, ok := tb.buckets[key]
	if !ok {
		b = &bucket{
			tokens:     float64(tb.capacity),
			lastRefill: time.Now(),
		}
		tb.buckets[key] = b
	}
	return b
}

func (tb *TokenBucket[K]) refill(b *bucket) {
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * tb.refillRate
	if b.tokens > float64(tb.capacity) {
		b.tokens = float64(tb.capacity)
	}
	b.lastRefill = now
}

// SlidingWindow implements a sliding window rate limiter.
type SlidingWindow[K comparable] struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	windows map[K]*windowData
}

type windowData struct {
	requests []time.Time
}

// NewSlidingWindow creates a new sliding window rate limiter.
func NewSlidingWindow[K comparable](limit int, window time.Duration) *SlidingWindow[K] {
	return &SlidingWindow[K]{
		limit:   limit,
		window:  window,
		windows: make(map[K]*windowData),
	}
}

// Allow checks if a request is allowed.
func (sw *SlidingWindow[K]) Allow(ctx context.Context, key K) (Decision, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	w := sw.getWindow(key)
	sw.cleanup(w)

	if len(w.requests) < sw.limit {
		w.requests = append(w.requests, time.Now())
		return Decision{
			Allowed:   true,
			Remaining: sw.limit - len(w.requests),
			Limit:     sw.limit,
			ResetAt:   time.Now().Add(sw.window),
		}, nil
	}

	oldestRequest := w.requests[0]
	retryAfter := sw.window - time.Since(oldestRequest)
	if retryAfter < 0 {
		retryAfter = 0
	}

	return Decision{
		Allowed:    false,
		Remaining:  0,
		Limit:      sw.limit,
		ResetAt:    oldestRequest.Add(sw.window),
		RetryAfter: retryAfter,
	}, nil
}

// Headers returns rate limit headers.
func (sw *SlidingWindow[K]) Headers(ctx context.Context, key K) (Headers, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	w := sw.getWindow(key)
	sw.cleanup(w)

	var resetTime int64
	if len(w.requests) > 0 {
		resetTime = w.requests[0].Add(sw.window).Unix()
	} else {
		resetTime = time.Now().Add(sw.window).Unix()
	}

	return Headers{
		Limit:     sw.limit,
		Remaining: sw.limit - len(w.requests),
		Reset:     resetTime,
	}, nil
}

func (sw *SlidingWindow[K]) getWindow(key K) *windowData {
	w, ok := sw.windows[key]
	if !ok {
		w = &windowData{requests: make([]time.Time, 0)}
		sw.windows[key] = w
	}
	return w
}

func (sw *SlidingWindow[K]) cleanup(w *windowData) {
	cutoff := time.Now().Add(-sw.window)
	i := 0
	for i < len(w.requests) && w.requests[i].Before(cutoff) {
		i++
	}
	w.requests = w.requests[i:]
}

// FixedWindow implements a fixed window rate limiter.
type FixedWindow[K comparable] struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	windows map[K]*fixedWindowData
}

type fixedWindowData struct {
	count       int
	windowStart time.Time
}

// NewFixedWindow creates a new fixed window rate limiter.
func NewFixedWindow[K comparable](limit int, window time.Duration) *FixedWindow[K] {
	return &FixedWindow[K]{
		limit:   limit,
		window:  window,
		windows: make(map[K]*fixedWindowData),
	}
}

// Allow checks if a request is allowed.
func (fw *FixedWindow[K]) Allow(ctx context.Context, key K) (Decision, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	w := fw.getWindow(key)
	fw.maybeReset(w)

	if w.count < fw.limit {
		w.count++
		return Decision{
			Allowed:   true,
			Remaining: fw.limit - w.count,
			Limit:     fw.limit,
			ResetAt:   w.windowStart.Add(fw.window),
		}, nil
	}

	retryAfter := fw.window - time.Since(w.windowStart)
	if retryAfter < 0 {
		retryAfter = 0
	}

	return Decision{
		Allowed:    false,
		Remaining:  0,
		Limit:      fw.limit,
		ResetAt:    w.windowStart.Add(fw.window),
		RetryAfter: retryAfter,
	}, nil
}

// Headers returns rate limit headers.
func (fw *FixedWindow[K]) Headers(ctx context.Context, key K) (Headers, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	w := fw.getWindow(key)
	fw.maybeReset(w)

	return Headers{
		Limit:     fw.limit,
		Remaining: fw.limit - w.count,
		Reset:     w.windowStart.Add(fw.window).Unix(),
	}, nil
}

func (fw *FixedWindow[K]) getWindow(key K) *fixedWindowData {
	w, ok := fw.windows[key]
	if !ok {
		w = &fixedWindowData{
			count:       0,
			windowStart: time.Now(),
		}
		fw.windows[key] = w
	}
	return w
}

func (fw *FixedWindow[K]) maybeReset(w *fixedWindowData) {
	if time.Since(w.windowStart) >= fw.window {
		w.count = 0
		w.windowStart = time.Now()
	}
}
