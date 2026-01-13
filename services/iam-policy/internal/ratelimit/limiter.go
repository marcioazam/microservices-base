// Package ratelimit provides rate limiting for IAM Policy Service.
package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/errors"
)

// Limiter provides rate limiting functionality.
type Limiter struct {
	mu       sync.RWMutex
	clients  map[string]*clientState
	config   LimiterConfig
	cleanupT *time.Ticker
	done     chan struct{}
}

// LimiterConfig holds configuration for the rate limiter.
type LimiterConfig struct {
	RequestsPerSecond int
	BurstSize         int
	CleanupInterval   time.Duration
}

// DefaultLimiterConfig returns default configuration.
func DefaultLimiterConfig() LimiterConfig {
	return LimiterConfig{
		RequestsPerSecond: 100,
		BurstSize:         200,
		CleanupInterval:   time.Minute,
	}
}

type clientState struct {
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewLimiter creates a new rate limiter.
func NewLimiter(config LimiterConfig) *Limiter {
	l := &Limiter{
		clients: make(map[string]*clientState),
		config:  config,
		done:    make(chan struct{}),
	}

	l.cleanupT = time.NewTicker(config.CleanupInterval)
	go l.cleanup()

	return l
}

// Allow checks if a request is allowed for the given client.
func (l *Limiter) Allow(ctx context.Context, clientID string) error {
	if clientID == "" {
		return nil // Allow anonymous requests
	}

	state := l.getOrCreateState(clientID)

	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(state.lastUpdate).Seconds()
	state.lastUpdate = now

	// Add tokens based on elapsed time
	state.tokens += elapsed * float64(l.config.RequestsPerSecond)
	if state.tokens > float64(l.config.BurstSize) {
		state.tokens = float64(l.config.BurstSize)
	}

	// Check if we have tokens available
	if state.tokens < 1 {
		return errors.RateLimited("rate limit exceeded")
	}

	state.tokens--
	return nil
}

// AllowN checks if n requests are allowed for the given client.
func (l *Limiter) AllowN(ctx context.Context, clientID string, n int) error {
	if clientID == "" || n <= 0 {
		return nil
	}

	state := l.getOrCreateState(clientID)

	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(state.lastUpdate).Seconds()
	state.lastUpdate = now

	state.tokens += elapsed * float64(l.config.RequestsPerSecond)
	if state.tokens > float64(l.config.BurstSize) {
		state.tokens = float64(l.config.BurstSize)
	}

	if state.tokens < float64(n) {
		return errors.RateLimited("rate limit exceeded")
	}

	state.tokens -= float64(n)
	return nil
}

func (l *Limiter) getOrCreateState(clientID string) *clientState {
	l.mu.RLock()
	state, ok := l.clients[clientID]
	l.mu.RUnlock()

	if ok {
		return state
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring write lock
	if state, ok = l.clients[clientID]; ok {
		return state
	}

	state = &clientState{
		tokens:     float64(l.config.BurstSize),
		lastUpdate: time.Now(),
	}
	l.clients[clientID] = state
	return state
}

func (l *Limiter) cleanup() {
	for {
		select {
		case <-l.cleanupT.C:
			l.removeStaleClients()
		case <-l.done:
			return
		}
	}
}

func (l *Limiter) removeStaleClients() {
	l.mu.Lock()
	defer l.mu.Unlock()

	threshold := time.Now().Add(-5 * time.Minute)
	for id, state := range l.clients {
		state.mu.Lock()
		if state.lastUpdate.Before(threshold) {
			delete(l.clients, id)
		}
		state.mu.Unlock()
	}
}

// Close stops the rate limiter.
func (l *Limiter) Close() {
	l.cleanupT.Stop()
	close(l.done)
}

// Stats returns rate limiter statistics.
func (l *Limiter) Stats() LimiterStats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return LimiterStats{
		ActiveClients: len(l.clients),
	}
}

// LimiterStats holds rate limiter statistics.
type LimiterStats struct {
	ActiveClients int
}
