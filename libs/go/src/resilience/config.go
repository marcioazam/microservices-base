package resilience

import (
	"time"
)

// CircuitBreakerConfig configures circuit breaker behavior.
type CircuitBreakerConfig struct {
	Name             string
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
	HalfOpenRequests int
	OnStateChange    func(from, to string)
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          time.Second * 30,
		HalfOpenRequests: 1,
	}
}

// Validate checks configuration validity.
func (c CircuitBreakerConfig) Validate() error {
	if c.FailureThreshold <= 0 {
		return NewInvalidPolicyError("FailureThreshold", c.FailureThreshold, "positive integer")
	}
	if c.SuccessThreshold <= 0 {
		return NewInvalidPolicyError("SuccessThreshold", c.SuccessThreshold, "positive integer")
	}
	if c.Timeout <= 0 {
		return NewInvalidPolicyError("Timeout", c.Timeout, "positive duration")
	}
	if c.HalfOpenRequests <= 0 {
		return NewInvalidPolicyError("HalfOpenRequests", c.HalfOpenRequests, "positive integer")
	}
	return nil
}

// CircuitBreakerOption configures CircuitBreakerConfig.
type CircuitBreakerOption func(*CircuitBreakerConfig)

// WithFailureThreshold sets failure threshold.
func WithFailureThreshold(n int) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) { c.FailureThreshold = n }
}

// WithSuccessThreshold sets success threshold.
func WithSuccessThreshold(n int) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) { c.SuccessThreshold = n }
}

// WithCircuitTimeout sets circuit timeout.
func WithCircuitTimeout(d time.Duration) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) { c.Timeout = d }
}

// WithHalfOpenRequests sets half-open request count.
func WithHalfOpenRequests(n int) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) { c.HalfOpenRequests = n }
}

// WithStateChangeCallback sets state change callback.
func WithStateChangeCallback(fn func(from, to string)) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) { c.OnStateChange = fn }
}

// NewCircuitBreakerConfig creates config with options.
func NewCircuitBreakerConfig(name string, opts ...CircuitBreakerOption) CircuitBreakerConfig {
	cfg := DefaultCircuitBreakerConfig()
	cfg.Name = name
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	JitterStrategy  JitterStrategy
	RetryIf         func(error) bool
}

// JitterStrategy defines jitter calculation method.
type JitterStrategy int

const (
	NoJitter JitterStrategy = iota
	FullJitter
	EqualJitter
	DecorrelatedJitter
)

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:     3,
		InitialInterval: time.Millisecond * 100,
		MaxInterval:     time.Second * 10,
		Multiplier:      2.0,
		JitterStrategy:  FullJitter,
		RetryIf:         func(error) bool { return true },
	}
}

// Validate checks configuration validity.
func (c RetryConfig) Validate() error {
	if c.MaxAttempts <= 0 {
		return NewInvalidPolicyError("MaxAttempts", c.MaxAttempts, "positive integer")
	}
	if c.InitialInterval <= 0 {
		return NewInvalidPolicyError("InitialInterval", c.InitialInterval, "positive duration")
	}
	if c.MaxInterval < c.InitialInterval {
		return NewInvalidPolicyError("MaxInterval", c.MaxInterval, "greater than or equal to InitialInterval")
	}
	if c.Multiplier < 1.0 {
		return NewInvalidPolicyError("Multiplier", c.Multiplier, "at least 1.0")
	}
	return nil
}

// RetryOption configures RetryConfig.
type RetryOption func(*RetryConfig)

// WithMaxAttempts sets max retry attempts.
func WithMaxAttempts(n int) RetryOption {
	return func(c *RetryConfig) { c.MaxAttempts = n }
}

// WithInitialInterval sets initial retry interval.
func WithInitialInterval(d time.Duration) RetryOption {
	return func(c *RetryConfig) { c.InitialInterval = d }
}

// WithMaxInterval sets max retry interval.
func WithMaxInterval(d time.Duration) RetryOption {
	return func(c *RetryConfig) { c.MaxInterval = d }
}

// WithMultiplier sets backoff multiplier.
func WithMultiplier(m float64) RetryOption {
	return func(c *RetryConfig) { c.Multiplier = m }
}

// WithJitterStrategy sets jitter strategy.
func WithJitterStrategy(s JitterStrategy) RetryOption {
	return func(c *RetryConfig) { c.JitterStrategy = s }
}

// WithRetryIf sets retry predicate.
func WithRetryIf(fn func(error) bool) RetryOption {
	return func(c *RetryConfig) { c.RetryIf = fn }
}

// NewRetryConfig creates config with options.
func NewRetryConfig(opts ...RetryOption) RetryConfig {
	cfg := DefaultRetryConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// RateLimitConfig configures rate limiting.
type RateLimitConfig struct {
	Rate       int
	Window     time.Duration
	BurstSize  int
	WaitOnLimit bool
}

// DefaultRateLimitConfig returns sensible defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Rate:       100,
		Window:     time.Second,
		BurstSize:  10,
		WaitOnLimit: false,
	}
}

// Validate checks configuration validity.
func (c RateLimitConfig) Validate() error {
	if c.Rate <= 0 {
		return NewInvalidPolicyError("Rate", c.Rate, "positive integer")
	}
	if c.Window <= 0 {
		return NewInvalidPolicyError("Window", c.Window, "positive duration")
	}
	if c.BurstSize < 0 {
		return NewInvalidPolicyError("BurstSize", c.BurstSize, "non-negative integer")
	}
	return nil
}

// BulkheadConfig configures bulkhead isolation.
type BulkheadConfig struct {
	MaxConcurrent int
	MaxWait       time.Duration
	QueueSize     int
}

// DefaultBulkheadConfig returns sensible defaults.
func DefaultBulkheadConfig() BulkheadConfig {
	return BulkheadConfig{
		MaxConcurrent: 10,
		MaxWait:       time.Second,
		QueueSize:     100,
	}
}

// Validate checks configuration validity.
func (c BulkheadConfig) Validate() error {
	if c.MaxConcurrent <= 0 {
		return NewInvalidPolicyError("MaxConcurrent", c.MaxConcurrent, "positive integer")
	}
	if c.MaxWait < 0 {
		return NewInvalidPolicyError("MaxWait", c.MaxWait, "non-negative duration")
	}
	if c.QueueSize < 0 {
		return NewInvalidPolicyError("QueueSize", c.QueueSize, "non-negative integer")
	}
	return nil
}

// TimeoutConfig configures timeout behavior.
type TimeoutConfig struct {
	Timeout   time.Duration
	OnTimeout func()
}

// DefaultTimeoutConfig returns sensible defaults.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout: time.Second * 30,
	}
}

// Validate checks configuration validity.
func (c TimeoutConfig) Validate() error {
	if c.Timeout <= 0 {
		return NewInvalidPolicyError("Timeout", c.Timeout, "positive duration")
	}
	return nil
}
