package resilience

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting.
type RateLimiter struct {
	config        RateLimitConfig
	tokens        int
	lastRefill    time.Time
	mu            sync.Mutex
	correlationID string
	service       string
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(config RateLimitConfig) (*RateLimiter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &RateLimiter{
		config:     config,
		tokens:     config.Rate + config.BurstSize,
		lastRefill: time.Now(),
	}, nil
}

// Allow checks if request is allowed.
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// AllowN checks if n requests are allowed.
func (rl *RateLimiter) AllowN(n int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens >= n {
		rl.tokens -= n
		return true
	}
	return false
}

// Wait blocks until request is allowed or context cancelled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(rl.config.Window / time.Duration(rl.config.Rate)):
		}
	}
}

// Execute runs operation with rate limiting.
func (rl *RateLimiter) Execute(ctx context.Context, op func(context.Context) error) error {
	if !rl.Allow() {
		if rl.config.WaitOnLimit {
			if err := rl.Wait(ctx); err != nil {
				return err
			}
		} else {
			return rl.rateLimitError()
		}
	}
	return op(ctx)
}

func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	tokensToAdd := int(elapsed / rl.config.Window * time.Duration(rl.config.Rate))
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		maxTokens := rl.config.Rate + rl.config.BurstSize
		if rl.tokens > maxTokens {
			rl.tokens = maxTokens
		}
		rl.lastRefill = now
	}
}

func (rl *RateLimiter) rateLimitError() *RateLimitError {
	retryAfter := rl.config.Window / time.Duration(rl.config.Rate)
	return NewRateLimitError(
		rl.service,
		rl.correlationID,
		rl.config.Rate,
		rl.config.Window,
		retryAfter,
	)
}

// SetService sets service name for errors.
func (rl *RateLimiter) SetService(service string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.service = service
}

// SetCorrelationID sets correlation ID for errors.
func (rl *RateLimiter) SetCorrelationID(id string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.correlationID = id
}

// Tokens returns current token count.
func (rl *RateLimiter) Tokens() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.refill()
	return rl.tokens
}
