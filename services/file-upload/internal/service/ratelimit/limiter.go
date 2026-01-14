// Package ratelimit provides rate limiting using Cache Service.
package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CacheClient defines the interface for cache operations.
type CacheClient interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// Result contains rate limit check result.
type Result struct {
	Allowed    bool
	Remaining  int
	ResetAt    time.Time
	RetryAfter time.Duration
}

// Config holds rate limiter configuration.
type Config struct {
	Limit     int
	Window    time.Duration
	KeyPrefix string
}

// Limiter implements sliding window rate limiting.
type Limiter struct {
	cache     CacheClient
	limit     int
	window    time.Duration
	keyPrefix string

	// Local fallback
	localData map[string]*windowData
	localMu   sync.RWMutex
	useFallback bool
}

type windowData struct {
	Timestamps []int64 `json:"timestamps"`
}

// NewLimiter creates a new rate limiter.
func NewLimiter(cache CacheClient, cfg Config) *Limiter {
	keyPrefix := cfg.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = "file-upload:ratelimit"
	}
	window := cfg.Window
	if window == 0 {
		window = time.Minute
	}

	return &Limiter{
		cache:     cache,
		limit:     cfg.Limit,
		window:    window,
		keyPrefix: keyPrefix,
		localData: make(map[string]*windowData),
	}
}

// Allow checks if a request is allowed for the given tenant.
func (l *Limiter) Allow(ctx context.Context, tenantID string) (*Result, error) {
	key := fmt.Sprintf("%s:%s", l.keyPrefix, tenantID)
	now := time.Now()
	windowStart := now.Add(-l.window).UnixNano()

	// Try cache first
	if !l.useFallback {
		result, err := l.allowWithCache(ctx, key, now, windowStart)
		if err == nil {
			return result, nil
		}
		// Fall back to local on cache error
		l.useFallback = true
	}

	// Use local fallback
	return l.allowWithLocal(key, now, windowStart), nil
}

func (l *Limiter) allowWithCache(ctx context.Context, key string, now time.Time, windowStart int64) (*Result, error) {
	// Get current window data
	data, err := l.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var wd windowData
	if data != nil {
		if err := json.Unmarshal(data, &wd); err != nil {
			wd = windowData{}
		}
	}

	// Filter timestamps within window
	var validTimestamps []int64
	for _, ts := range wd.Timestamps {
		if ts > windowStart {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check limit
	if len(validTimestamps) >= l.limit {
		retryAfter := l.calculateRetryAfter(validTimestamps, now, windowStart)
		return &Result{
			Allowed:    false,
			Remaining:  0,
			ResetAt:    now.Add(l.window),
			RetryAfter: retryAfter,
		}, nil
	}

	// Add current request
	validTimestamps = append(validTimestamps, now.UnixNano())
	wd.Timestamps = validTimestamps

	// Save to cache
	encoded, _ := json.Marshal(wd)
	if err := l.cache.Set(ctx, key, encoded, l.window+time.Second); err != nil {
		return nil, err
	}

	return &Result{
		Allowed:   true,
		Remaining: l.limit - len(validTimestamps),
		ResetAt:   now.Add(l.window),
	}, nil
}

func (l *Limiter) allowWithLocal(key string, now time.Time, windowStart int64) *Result {
	l.localMu.Lock()
	defer l.localMu.Unlock()

	wd, exists := l.localData[key]
	if !exists {
		wd = &windowData{}
		l.localData[key] = wd
	}

	// Filter timestamps within window
	var validTimestamps []int64
	for _, ts := range wd.Timestamps {
		if ts > windowStart {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check limit
	if len(validTimestamps) >= l.limit {
		retryAfter := l.calculateRetryAfter(validTimestamps, now, windowStart)
		return &Result{
			Allowed:    false,
			Remaining:  0,
			ResetAt:    now.Add(l.window),
			RetryAfter: retryAfter,
		}
	}

	// Add current request
	validTimestamps = append(validTimestamps, now.UnixNano())
	wd.Timestamps = validTimestamps

	return &Result{
		Allowed:   true,
		Remaining: l.limit - len(validTimestamps),
		ResetAt:   now.Add(l.window),
	}
}

func (l *Limiter) calculateRetryAfter(timestamps []int64, now time.Time, windowStart int64) time.Duration {
	if len(timestamps) == 0 {
		return l.window
	}

	// Find oldest timestamp in window
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

// Reset resets the rate limit for a tenant.
func (l *Limiter) Reset(ctx context.Context, tenantID string) error {
	key := fmt.Sprintf("%s:%s", l.keyPrefix, tenantID)

	l.localMu.Lock()
	delete(l.localData, key)
	l.localMu.Unlock()

	if !l.useFallback {
		return l.cache.Set(ctx, key, []byte("{}"), time.Second)
	}
	return nil
}

// GetLimit returns the configured limit.
func (l *Limiter) GetLimit() int {
	return l.limit
}

// GetWindow returns the configured window duration.
func (l *Limiter) GetWindow() time.Duration {
	return l.window
}

// SetFallbackMode enables or disables fallback mode.
func (l *Limiter) SetFallbackMode(enabled bool) {
	l.useFallback = enabled
}

// IsFallbackMode returns true if using local fallback.
func (l *Limiter) IsFallbackMode() bool {
	return l.useFallback
}
