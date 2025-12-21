package resilience

import (
	"fmt"
	"time"
)

// CircuitState represents the circuit breaker state.
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig defines circuit breaker behavior.
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold" yaml:"failureThreshold"`
	SuccessThreshold int           `json:"success_threshold" yaml:"successThreshold"`
	Timeout          time.Duration `json:"timeout" yaml:"timeout"`
	ProbeCount       int           `json:"probe_count" yaml:"probeCount"`
}

// Validate validates the circuit breaker configuration.
func (c *CircuitBreakerConfig) Validate() error {
	if c.FailureThreshold <= 0 {
		return fmt.Errorf("circuit_breaker.failure_threshold must be positive")
	}
	if c.SuccessThreshold <= 0 {
		return fmt.Errorf("circuit_breaker.success_threshold must be positive")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("circuit_breaker.timeout must be positive")
	}
	return nil
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		ProbeCount:       1,
	}
}

// RetryConfig defines retry behavior.
type RetryConfig struct {
	MaxAttempts     int           `json:"max_attempts" yaml:"maxAttempts"`
	BaseDelay       time.Duration `json:"base_delay" yaml:"baseDelay"`
	MaxDelay        time.Duration `json:"max_delay" yaml:"maxDelay"`
	Multiplier      float64       `json:"multiplier" yaml:"multiplier"`
	JitterPercent   float64       `json:"jitter_percent" yaml:"jitterPercent"`
	RetryableErrors []string      `json:"retryable_errors" yaml:"retryableErrors"`
}

// Validate validates the retry configuration.
func (c *RetryConfig) Validate() error {
	if c.MaxAttempts <= 0 {
		return fmt.Errorf("retry.max_attempts must be positive")
	}
	if c.BaseDelay <= 0 {
		return fmt.Errorf("retry.base_delay must be positive")
	}
	if c.MaxDelay <= 0 {
		return fmt.Errorf("retry.max_delay must be positive")
	}
	if c.Multiplier < 1.0 {
		return fmt.Errorf("retry.multiplier must be >= 1.0")
	}
	if c.JitterPercent < 0 || c.JitterPercent > 1 {
		return fmt.Errorf("retry.jitter_percent must be in [0, 1]")
	}
	return nil
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		Multiplier:    2.0,
		JitterPercent: 0.1,
	}
}

// TimeoutConfig defines timeout behavior.
type TimeoutConfig struct {
	Default time.Duration            `json:"default" yaml:"default"`
	Max     time.Duration            `json:"max" yaml:"max"`
	PerOp   map[string]time.Duration `json:"per_operation" yaml:"perOperation"`
}

// Validate validates the timeout configuration.
func (c *TimeoutConfig) Validate() error {
	if c.Default <= 0 {
		return fmt.Errorf("timeout.default must be positive")
	}
	return nil
}

// DefaultTimeoutConfig returns a default timeout configuration.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Default: 30 * time.Second,
		Max:     60 * time.Second,
	}
}

// RateLimitAlgorithm defines the rate limiting algorithm.
type RateLimitAlgorithm string

const (
	TokenBucket   RateLimitAlgorithm = "token_bucket"
	SlidingWindow RateLimitAlgorithm = "sliding_window"
)

// RateLimitConfig defines rate limiting behavior.
type RateLimitConfig struct {
	Algorithm RateLimitAlgorithm `json:"algorithm" yaml:"algorithm"`
	Limit     int                `json:"limit" yaml:"limit"`
	Window    time.Duration      `json:"window" yaml:"window"`
	BurstSize int                `json:"burst_size" yaml:"burstSize"`
}

// Validate validates the rate limit configuration.
func (c *RateLimitConfig) Validate() error {
	if c.Limit <= 0 {
		return fmt.Errorf("rate_limit.limit must be positive")
	}
	if c.Window <= 0 {
		return fmt.Errorf("rate_limit.window must be positive")
	}
	return nil
}

// DefaultRateLimitConfig returns a default rate limit configuration.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Algorithm: TokenBucket,
		Limit:     100,
		Window:    time.Second,
		BurstSize: 10,
	}
}

// BulkheadConfig defines bulkhead behavior.
type BulkheadConfig struct {
	MaxConcurrent int           `json:"max_concurrent" yaml:"maxConcurrent"`
	MaxQueue      int           `json:"max_queue" yaml:"maxQueue"`
	QueueTimeout  time.Duration `json:"queue_timeout" yaml:"queueTimeout"`
}

// Validate validates the bulkhead configuration.
func (c *BulkheadConfig) Validate() error {
	if c.MaxConcurrent <= 0 {
		return fmt.Errorf("bulkhead.max_concurrent must be positive")
	}
	if c.MaxQueue < 0 {
		return fmt.Errorf("bulkhead.max_queue must be >= 0")
	}
	return nil
}

// DefaultBulkheadConfig returns a default bulkhead configuration.
func DefaultBulkheadConfig() BulkheadConfig {
	return BulkheadConfig{
		MaxConcurrent: 10,
		MaxQueue:      100,
		QueueTimeout:  5 * time.Second,
	}
}

// ResiliencePolicy represents a complete resilience policy.
type ResiliencePolicy struct {
	Name           string                `json:"name" yaml:"name"`
	Version        int64                 `json:"version" yaml:"version"`
	ServicePattern string                `json:"service_pattern" yaml:"servicePattern"`
	CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker,omitempty" yaml:"circuitBreaker,omitempty"`
	Retry          *RetryConfig          `json:"retry,omitempty" yaml:"retry,omitempty"`
	Timeout        *TimeoutConfig        `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	RateLimit      *RateLimitConfig      `json:"rate_limit,omitempty" yaml:"rateLimit,omitempty"`
	Bulkhead       *BulkheadConfig       `json:"bulkhead,omitempty" yaml:"bulkhead,omitempty"`
}

// Validate validates the resilience policy.
func (p *ResiliencePolicy) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("policy.name must not be empty")
	}
	if p.CircuitBreaker != nil {
		if err := p.CircuitBreaker.Validate(); err != nil {
			return err
		}
	}
	if p.Retry != nil {
		if err := p.Retry.Validate(); err != nil {
			return err
		}
	}
	if p.Timeout != nil {
		if err := p.Timeout.Validate(); err != nil {
			return err
		}
	}
	if p.RateLimit != nil {
		if err := p.RateLimit.Validate(); err != nil {
			return err
		}
	}
	if p.Bulkhead != nil {
		if err := p.Bulkhead.Validate(); err != nil {
			return err
		}
	}
	return nil
}
