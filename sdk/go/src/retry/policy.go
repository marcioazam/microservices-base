// Package retry provides retry logic with exponential backoff.
package retry

import (
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// Policy configures retry behavior for operations.
type Policy struct {
	MaxRetries           int
	BaseDelay            time.Duration
	MaxDelay             time.Duration
	Jitter               float64
	RetryableStatusCodes []int
}

// DefaultPolicy returns a sensible default retry policy.
func DefaultPolicy() *Policy {
	return &Policy{
		MaxRetries:           3,
		BaseDelay:            time.Second,
		MaxDelay:             30 * time.Second,
		Jitter:               0.2,
		RetryableStatusCodes: []int{429, 502, 503, 504},
	}
}

// Option configures a Policy.
type Option func(*Policy)

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(n int) Option {
	return func(p *Policy) { p.MaxRetries = n }
}

// WithBaseDelay sets the base delay.
func WithBaseDelay(d time.Duration) Option {
	return func(p *Policy) { p.BaseDelay = d }
}

// WithMaxDelay sets the maximum delay.
func WithMaxDelay(d time.Duration) Option {
	return func(p *Policy) { p.MaxDelay = d }
}

// WithJitter sets the jitter factor (0.0 to 1.0).
func WithJitter(j float64) Option {
	return func(p *Policy) { p.Jitter = j }
}

// WithRetryableStatusCodes sets the retryable HTTP status codes.
func WithRetryableStatusCodes(codes ...int) Option {
	return func(p *Policy) { p.RetryableStatusCodes = codes }
}

// NewPolicy creates a new retry policy with options.
func NewPolicy(opts ...Option) *Policy {
	p := DefaultPolicy()
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// CalculateDelay computes the delay for a given attempt using exponential backoff.
func (p *Policy) CalculateDelay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Exponential backoff: baseDelay * 2^attempt
	delay := float64(p.BaseDelay) * math.Pow(2, float64(attempt))

	// Apply jitter
	if p.Jitter > 0 {
		jitterRange := delay * p.Jitter
		delay = delay - jitterRange + (rand.Float64() * 2 * jitterRange)
	}

	// Clamp to bounds
	if delay < float64(p.BaseDelay) {
		delay = float64(p.BaseDelay)
	}
	if delay > float64(p.MaxDelay) {
		delay = float64(p.MaxDelay)
	}

	return time.Duration(delay)
}

// IsRetryable checks if an HTTP status code should trigger a retry.
func (p *Policy) IsRetryable(statusCode int) bool {
	for _, code := range p.RetryableStatusCodes {
		if code == statusCode {
			return true
		}
	}
	return false
}

// ParseRetryAfter parses the Retry-After header value.
func ParseRetryAfter(header string) (time.Duration, bool) {
	if header == "" {
		return 0, false
	}

	// Try parsing as seconds (integer)
	if seconds, err := strconv.ParseInt(header, 10, 64); err == nil {
		return time.Duration(seconds) * time.Second, true
	}

	// Try parsing as HTTP-date
	if t, err := http.ParseTime(header); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return delay, true
		}
	}

	return 0, false
}
