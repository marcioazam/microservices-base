// Package entities defines resilience configuration entities with composable validation.
package entities

import (
	"fmt"
	"time"

	"github.com/authcorp/libs/go/src/functional"
	"github.com/authcorp/libs/go/src/validation"
)

// CircuitBreakerConfig defines circuit breaker parameters.
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold"`
	SuccessThreshold int           `json:"success_threshold"`
	Timeout          time.Duration `json:"timeout"`
	ProbeCount       int           `json:"probe_count"`
}

// NewCircuitBreakerConfig creates a new circuit breaker configuration.
func NewCircuitBreakerConfig(failureThreshold, successThreshold int, timeout time.Duration, probeCount int) functional.Result[*CircuitBreakerConfig] {
	config := &CircuitBreakerConfig{
		FailureThreshold: failureThreshold,
		SuccessThreshold: successThreshold,
		Timeout:          timeout,
		ProbeCount:       probeCount,
	}
	return config.ValidateResult()
}

// Validate validates circuit breaker configuration (legacy compatibility).
func (c *CircuitBreakerConfig) Validate() error {
	result := c.ValidateResult()
	if result.IsErr() {
		return result.UnwrapErr()
	}
	return nil
}

// ValidateResult validates using composable validators from libs/go.
func (c *CircuitBreakerConfig) ValidateResult() functional.Result[*CircuitBreakerConfig] {
	result := validation.NewResult()

	result.Merge(validation.Field("failure_threshold", c.FailureThreshold,
		validation.InRange(1, 100)))

	result.Merge(validation.Field("success_threshold", c.SuccessThreshold,
		validation.InRange(1, 10)))

	result.Merge(validation.Field("timeout", c.Timeout,
		validation.DurationRange(time.Second, 5*time.Minute)))

	result.Merge(validation.Field("probe_count", c.ProbeCount,
		validation.InRange(1, 10)))

	// Cross-field validation
	if c.SuccessThreshold > c.FailureThreshold {
		result.AddFieldError("success_threshold",
			fmt.Sprintf("cannot be greater than failure_threshold (%d)", c.FailureThreshold),
			"cross_field")
	}

	if !result.IsValid() {
		return functional.Err[*CircuitBreakerConfig](
			fmt.Errorf("validation failed: %v", result.ErrorMessages()))
	}
	return functional.Ok(c)
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
func NewRetryConfig(maxAttempts int, baseDelay, maxDelay time.Duration, multiplier, jitterPercent float64) functional.Result[*RetryConfig] {
	config := &RetryConfig{
		MaxAttempts:   maxAttempts,
		BaseDelay:     baseDelay,
		MaxDelay:      maxDelay,
		Multiplier:    multiplier,
		JitterPercent: jitterPercent,
	}
	return config.ValidateResult()
}

// Validate validates retry configuration (legacy compatibility).
func (r *RetryConfig) Validate() error {
	result := r.ValidateResult()
	if result.IsErr() {
		return result.UnwrapErr()
	}
	return nil
}

// ValidateResult validates using composable validators.
func (r *RetryConfig) ValidateResult() functional.Result[*RetryConfig] {
	result := validation.NewResult()

	result.Merge(validation.Field("max_attempts", r.MaxAttempts,
		validation.InRange(1, 10)))

	result.Merge(validation.Field("base_delay", r.BaseDelay,
		validation.DurationRange(time.Millisecond, 10*time.Second)))

	result.Merge(validation.Field("max_delay", r.MaxDelay,
		validation.DurationRange(time.Second, 5*time.Minute)))

	result.Merge(validation.Field("multiplier", r.Multiplier,
		validation.FloatRange(1.0, 10.0)))

	result.Merge(validation.Field("jitter_percent", r.JitterPercent,
		validation.FloatRange(0.0, 1.0)))

	// Cross-field validation
	if r.BaseDelay > r.MaxDelay {
		result.AddFieldError("base_delay",
			fmt.Sprintf("cannot be greater than max_delay (%v)", r.MaxDelay),
			"cross_field")
	}

	if !result.IsValid() {
		return functional.Err[*RetryConfig](
			fmt.Errorf("validation failed: %v", result.ErrorMessages()))
	}
	return functional.Ok(r)
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
func NewTimeoutConfig(defaultTimeout, maxTimeout time.Duration) functional.Result[*TimeoutConfig] {
	config := &TimeoutConfig{
		Default: defaultTimeout,
		Max:     maxTimeout,
	}
	return config.ValidateResult()
}

// Validate validates timeout configuration (legacy compatibility).
func (t *TimeoutConfig) Validate() error {
	result := t.ValidateResult()
	if result.IsErr() {
		return result.UnwrapErr()
	}
	return nil
}

// ValidateResult validates using composable validators.
func (t *TimeoutConfig) ValidateResult() functional.Result[*TimeoutConfig] {
	result := validation.NewResult()

	result.Merge(validation.Field("default", t.Default,
		validation.DurationRange(100*time.Millisecond, 5*time.Minute)))

	result.Merge(validation.Field("max", t.Max,
		validation.DurationRange(time.Second, 10*time.Minute)))

	// Cross-field validation
	if t.Default > t.Max {
		result.AddFieldError("default",
			fmt.Sprintf("cannot be greater than max (%v)", t.Max),
			"cross_field")
	}

	if !result.IsValid() {
		return functional.Err[*TimeoutConfig](
			fmt.Errorf("validation failed: %v", result.ErrorMessages()))
	}
	return functional.Ok(t)
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
func NewRateLimitConfig(algorithm string, limit int, window time.Duration, burstSize int) functional.Result[*RateLimitConfig] {
	config := &RateLimitConfig{
		Algorithm: algorithm,
		Limit:     limit,
		Window:    window,
		BurstSize: burstSize,
	}
	return config.ValidateResult()
}

// Validate validates rate limit configuration (legacy compatibility).
func (r *RateLimitConfig) Validate() error {
	result := r.ValidateResult()
	if result.IsErr() {
		return result.UnwrapErr()
	}
	return nil
}

// ValidateResult validates using composable validators.
func (r *RateLimitConfig) ValidateResult() functional.Result[*RateLimitConfig] {
	result := validation.NewResult()

	result.Merge(validation.Field("algorithm", r.Algorithm,
		validation.OneOf("token_bucket", "sliding_window")))

	result.Merge(validation.Field("limit", r.Limit,
		validation.InRange(1, 100000)))

	result.Merge(validation.Field("window", r.Window,
		validation.DurationRange(time.Second, time.Hour)))

	result.Merge(validation.Field("burst_size", r.BurstSize,
		validation.InRange(1, 10000)))

	// Cross-field validation
	if r.BurstSize > r.Limit {
		result.AddFieldError("burst_size",
			fmt.Sprintf("cannot be greater than limit (%d)", r.Limit),
			"cross_field")
	}

	if !result.IsValid() {
		return functional.Err[*RateLimitConfig](
			fmt.Errorf("validation failed: %v", result.ErrorMessages()))
	}
	return functional.Ok(r)
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
func NewBulkheadConfig(maxConcurrent, maxQueue int, queueTimeout time.Duration) functional.Result[*BulkheadConfig] {
	config := &BulkheadConfig{
		MaxConcurrent: maxConcurrent,
		MaxQueue:      maxQueue,
		QueueTimeout:  queueTimeout,
	}
	return config.ValidateResult()
}

// Validate validates bulkhead configuration (legacy compatibility).
func (b *BulkheadConfig) Validate() error {
	result := b.ValidateResult()
	if result.IsErr() {
		return result.UnwrapErr()
	}
	return nil
}

// ValidateResult validates using composable validators.
func (b *BulkheadConfig) ValidateResult() functional.Result[*BulkheadConfig] {
	result := validation.NewResult()

	result.Merge(validation.Field("max_concurrent", b.MaxConcurrent,
		validation.InRange(1, 10000)))

	result.Merge(validation.Field("max_queue", b.MaxQueue,
		validation.InRange(0, 10000)))

	result.Merge(validation.Field("queue_timeout", b.QueueTimeout,
		validation.DurationRange(time.Millisecond, 30*time.Second)))

	if !result.IsValid() {
		return functional.Err[*BulkheadConfig](
			fmt.Errorf("validation failed: %v", result.ErrorMessages()))
	}
	return functional.Ok(b)
}

// Clone creates a deep copy of the bulkhead configuration.
func (b *BulkheadConfig) Clone() *BulkheadConfig {
	return &BulkheadConfig{
		MaxConcurrent: b.MaxConcurrent,
		MaxQueue:      b.MaxQueue,
		QueueTimeout:  b.QueueTimeout,
	}
}
