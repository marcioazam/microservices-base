// Package entities defines resilience configuration entities.
package entities

import (
	"fmt"
	"time"
)

// CircuitBreakerConfig defines circuit breaker parameters.
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold"`
	SuccessThreshold int           `json:"success_threshold"`
	Timeout          time.Duration `json:"timeout"`
	ProbeCount       int           `json:"probe_count"`
}

// NewCircuitBreakerConfig creates a new circuit breaker configuration.
func NewCircuitBreakerConfig(failureThreshold, successThreshold int, timeout time.Duration, probeCount int) (*CircuitBreakerConfig, error) {
	config := &CircuitBreakerConfig{
		FailureThreshold: failureThreshold,
		SuccessThreshold: successThreshold,
		Timeout:          timeout,
		ProbeCount:       probeCount,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates circuit breaker configuration.
func (c *CircuitBreakerConfig) Validate() error {
	if c.FailureThreshold < 1 || c.FailureThreshold > 100 {
		return fmt.Errorf("failure threshold must be between 1 and 100, got: %d", c.FailureThreshold)
	}

	if c.SuccessThreshold < 1 || c.SuccessThreshold > 10 {
		return fmt.Errorf("success threshold must be between 1 and 10, got: %d", c.SuccessThreshold)
	}

	if c.SuccessThreshold > c.FailureThreshold {
		return fmt.Errorf("success threshold (%d) cannot be greater than failure threshold (%d)",
			c.SuccessThreshold, c.FailureThreshold)
	}

	if c.Timeout < time.Second || c.Timeout > 5*time.Minute {
		return fmt.Errorf("timeout must be between 1s and 5m, got: %v", c.Timeout)
	}

	if c.ProbeCount < 1 || c.ProbeCount > 10 {
		return fmt.Errorf("probe count must be between 1 and 10, got: %d", c.ProbeCount)
	}

	return nil
}

// Clone creates a deep copy of the circuit breaker configuration.
func (c *CircuitBreakerConfig) Clone() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold: c.FailureThreshold,
		SuccessThreshold: c.SuccessThreshold,
		Timeout:          c.Timeout,
		ProbeCount:       c.ProbeCount,
	}
}

// RetryConfig defines retry behavior parameters.
type RetryConfig struct {
	MaxAttempts   int           `json:"max_attempts"`
	BaseDelay     time.Duration `json:"base_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	Multiplier    float64       `json:"multiplier"`
	JitterPercent float64       `json:"jitter_percent"`
}

// NewRetryConfig creates a new retry configuration.
func NewRetryConfig(maxAttempts int, baseDelay, maxDelay time.Duration, multiplier, jitterPercent float64) (*RetryConfig, error) {
	config := &RetryConfig{
		MaxAttempts:   maxAttempts,
		BaseDelay:     baseDelay,
		MaxDelay:      maxDelay,
		Multiplier:    multiplier,
		JitterPercent: jitterPercent,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates retry configuration.
func (r *RetryConfig) Validate() error {
	if r.MaxAttempts < 1 || r.MaxAttempts > 10 {
		return fmt.Errorf("max attempts must be between 1 and 10, got: %d", r.MaxAttempts)
	}

	if r.BaseDelay < time.Millisecond || r.BaseDelay > 10*time.Second {
		return fmt.Errorf("base delay must be between 1ms and 10s, got: %v", r.BaseDelay)
	}

	if r.MaxDelay < time.Second || r.MaxDelay > 5*time.Minute {
		return fmt.Errorf("max delay must be between 1s and 5m, got: %v", r.MaxDelay)
	}

	if r.BaseDelay > r.MaxDelay {
		return fmt.Errorf("base delay (%v) cannot be greater than max delay (%v)", r.BaseDelay, r.MaxDelay)
	}

	if r.Multiplier < 1.0 || r.Multiplier > 10.0 {
		return fmt.Errorf("multiplier must be between 1.0 and 10.0, got: %f", r.Multiplier)
	}

	if r.JitterPercent < 0.0 || r.JitterPercent > 1.0 {
		return fmt.Errorf("jitter percent must be between 0.0 and 1.0, got: %f", r.JitterPercent)
	}

	return nil
}

// Clone creates a deep copy of the retry configuration.
func (r *RetryConfig) Clone() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   r.MaxAttempts,
		BaseDelay:     r.BaseDelay,
		MaxDelay:      r.MaxDelay,
		Multiplier:    r.Multiplier,
		JitterPercent: r.JitterPercent,
	}
}

// TimeoutConfig defines timeout parameters.
type TimeoutConfig struct {
	Default time.Duration `json:"default"`
	Max     time.Duration `json:"max"`
}

// NewTimeoutConfig creates a new timeout configuration.
func NewTimeoutConfig(defaultTimeout, maxTimeout time.Duration) (*TimeoutConfig, error) {
	config := &TimeoutConfig{
		Default: defaultTimeout,
		Max:     maxTimeout,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates timeout configuration.
func (t *TimeoutConfig) Validate() error {
	if t.Default < 100*time.Millisecond || t.Default > 5*time.Minute {
		return fmt.Errorf("default timeout must be between 100ms and 5m, got: %v", t.Default)
	}

	if t.Max < time.Second || t.Max > 10*time.Minute {
		return fmt.Errorf("max timeout must be between 1s and 10m, got: %v", t.Max)
	}

	if t.Default > t.Max {
		return fmt.Errorf("default timeout (%v) cannot be greater than max timeout (%v)", t.Default, t.Max)
	}

	return nil
}

// Clone creates a deep copy of the timeout configuration.
func (t *TimeoutConfig) Clone() *TimeoutConfig {
	return &TimeoutConfig{
		Default: t.Default,
		Max:     t.Max,
	}
}

// RateLimitConfig defines rate limiting parameters.
type RateLimitConfig struct {
	Algorithm string        `json:"algorithm"`
	Limit     int           `json:"limit"`
	Window    time.Duration `json:"window"`
	BurstSize int           `json:"burst_size"`
}

// NewRateLimitConfig creates a new rate limit configuration.
func NewRateLimitConfig(algorithm string, limit int, window time.Duration, burstSize int) (*RateLimitConfig, error) {
	config := &RateLimitConfig{
		Algorithm: algorithm,
		Limit:     limit,
		Window:    window,
		BurstSize: burstSize,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates rate limit configuration.
func (r *RateLimitConfig) Validate() error {
	validAlgorithms := map[string]bool{
		"token_bucket":   true,
		"sliding_window": true,
	}

	if !validAlgorithms[r.Algorithm] {
		return fmt.Errorf("algorithm must be 'token_bucket' or 'sliding_window', got: %s", r.Algorithm)
	}

	if r.Limit < 1 || r.Limit > 100000 {
		return fmt.Errorf("limit must be between 1 and 100000, got: %d", r.Limit)
	}

	if r.Window < time.Second || r.Window > time.Hour {
		return fmt.Errorf("window must be between 1s and 1h, got: %v", r.Window)
	}

	if r.BurstSize < 1 || r.BurstSize > 10000 {
		return fmt.Errorf("burst size must be between 1 and 10000, got: %d", r.BurstSize)
	}

	if r.BurstSize > r.Limit {
		return fmt.Errorf("burst size (%d) cannot be greater than limit (%d)", r.BurstSize, r.Limit)
	}

	return nil
}

// Clone creates a deep copy of the rate limit configuration.
func (r *RateLimitConfig) Clone() *RateLimitConfig {
	return &RateLimitConfig{
		Algorithm: r.Algorithm,
		Limit:     r.Limit,
		Window:    r.Window,
		BurstSize: r.BurstSize,
	}
}

// BulkheadConfig defines bulkhead isolation parameters.
type BulkheadConfig struct {
	MaxConcurrent int           `json:"max_concurrent"`
	MaxQueue      int           `json:"max_queue"`
	QueueTimeout  time.Duration `json:"queue_timeout"`
}

// NewBulkheadConfig creates a new bulkhead configuration.
func NewBulkheadConfig(maxConcurrent, maxQueue int, queueTimeout time.Duration) (*BulkheadConfig, error) {
	config := &BulkheadConfig{
		MaxConcurrent: maxConcurrent,
		MaxQueue:      maxQueue,
		QueueTimeout:  queueTimeout,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates bulkhead configuration.
func (b *BulkheadConfig) Validate() error {
	if b.MaxConcurrent < 1 || b.MaxConcurrent > 10000 {
		return fmt.Errorf("max concurrent must be between 1 and 10000, got: %d", b.MaxConcurrent)
	}

	if b.MaxQueue < 0 || b.MaxQueue > 10000 {
		return fmt.Errorf("max queue must be between 0 and 10000, got: %d", b.MaxQueue)
	}

	if b.QueueTimeout < time.Millisecond || b.QueueTimeout > 30*time.Second {
		return fmt.Errorf("queue timeout must be between 1ms and 30s, got: %v", b.QueueTimeout)
	}

	return nil
}

// Clone creates a deep copy of the bulkhead configuration.
func (b *BulkheadConfig) Clone() *BulkheadConfig {
	return &BulkheadConfig{
		MaxConcurrent: b.MaxConcurrent,
		MaxQueue:      b.MaxQueue,
		QueueTimeout:  b.QueueTimeout,
	}
}