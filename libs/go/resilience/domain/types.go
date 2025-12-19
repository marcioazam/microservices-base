// Package domain provides resilience domain types and interfaces.
package domain

import (
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// CircuitClosed allows requests to pass through.
	CircuitClosed CircuitState = iota
	// CircuitOpen blocks all requests.
	CircuitOpen
	// CircuitHalfOpen allows limited requests to test recovery.
	CircuitHalfOpen
)

// String returns the string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds circuit breaker configuration.
type CircuitBreakerConfig struct {
	// Name is the circuit breaker name.
	Name string `json:"name" yaml:"name"`
	// FailureThreshold is the number of failures before opening.
	FailureThreshold int `json:"failure_threshold" yaml:"failure_threshold"`
	// SuccessThreshold is the number of successes in half-open before closing.
	SuccessThreshold int `json:"success_threshold" yaml:"success_threshold"`
	// Timeout is the duration to wait before transitioning from open to half-open.
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
	// HalfOpenMaxCalls is the max calls allowed in half-open state.
	HalfOpenMaxCalls int `json:"half_open_max_calls" yaml:"half_open_max_calls"`
}

// RateLimitAlgorithm represents the rate limiting algorithm.
type RateLimitAlgorithm string

const (
	TokenBucket   RateLimitAlgorithm = "token_bucket"
	SlidingWindow RateLimitAlgorithm = "sliding_window"
	FixedWindow   RateLimitAlgorithm = "fixed_window"
	LeakyBucket   RateLimitAlgorithm = "leaky_bucket"
)

// RateLimitConfig holds rate limiter configuration.
type RateLimitConfig struct {
	// Algorithm is the rate limiting algorithm.
	Algorithm RateLimitAlgorithm `json:"algorithm" yaml:"algorithm"`
	// Limit is the maximum number of requests.
	Limit int `json:"limit" yaml:"limit"`
	// Window is the time window for the limit.
	Window time.Duration `json:"window" yaml:"window"`
	// BurstSize is the maximum burst size (for token bucket).
	BurstSize int `json:"burst_size" yaml:"burst_size"`
	// RefillRate is the token refill rate per second (for token bucket).
	RefillRate float64 `json:"refill_rate" yaml:"refill_rate"`
}

// RetryConfig holds retry configuration.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts.
	MaxAttempts int `json:"max_attempts" yaml:"max_attempts"`
	// InitialDelay is the initial delay between retries.
	InitialDelay time.Duration `json:"initial_delay" yaml:"initial_delay"`
	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration `json:"max_delay" yaml:"max_delay"`
	// Multiplier is the exponential backoff multiplier.
	Multiplier float64 `json:"multiplier" yaml:"multiplier"`
	// Jitter is the jitter percentage (0-1).
	Jitter float64 `json:"jitter" yaml:"jitter"`
	// RetryableErrors is a list of error codes that should be retried.
	RetryableErrors []string `json:"retryable_errors" yaml:"retryable_errors"`
}

// TimeoutConfig holds timeout configuration.
type TimeoutConfig struct {
	// Default is the default timeout for operations.
	Default time.Duration `json:"default" yaml:"default"`
	// Max is the maximum allowed timeout.
	Max time.Duration `json:"max" yaml:"max"`
	// PerOperation is a map of operation-specific timeouts.
	PerOperation map[string]time.Duration `json:"per_operation" yaml:"per_operation"`
}

// BulkheadConfig holds bulkhead configuration.
type BulkheadConfig struct {
	// Name is the bulkhead name.
	Name string `json:"name" yaml:"name"`
	// MaxConcurrent is the maximum concurrent executions.
	MaxConcurrent int `json:"max_concurrent" yaml:"max_concurrent"`
	// MaxQueue is the maximum queue size.
	MaxQueue int `json:"max_queue" yaml:"max_queue"`
	// QueueTimeout is the maximum time to wait in queue.
	QueueTimeout time.Duration `json:"queue_timeout" yaml:"queue_timeout"`
}

// ResiliencePolicy combines all resilience configurations.
type ResiliencePolicy struct {
	// Name is the policy name.
	Name string `json:"name" yaml:"name"`
	// Version is the policy version.
	Version int `json:"version" yaml:"version"`
	// CircuitBreaker is the circuit breaker configuration.
	CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker,omitempty" yaml:"circuit_breaker,omitempty"`
	// RateLimit is the rate limiter configuration.
	RateLimit *RateLimitConfig `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`
	// Retry is the retry configuration.
	Retry *RetryConfig `json:"retry,omitempty" yaml:"retry,omitempty"`
	// Timeout is the timeout configuration.
	Timeout *TimeoutConfig `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	// Bulkhead is the bulkhead configuration.
	Bulkhead *BulkheadConfig `json:"bulkhead,omitempty" yaml:"bulkhead,omitempty"`
}

// HealthStatus represents the health status of a component.
type HealthStatus int

const (
	// Healthy indicates the component is functioning normally.
	Healthy HealthStatus = iota
	// Degraded indicates the component is functioning with reduced capacity.
	Degraded
	// Unhealthy indicates the component is not functioning.
	Unhealthy
)

// String returns the string representation of the health status.
func (s HealthStatus) String() string {
	switch s {
	case Healthy:
		return "healthy"
	case Degraded:
		return "degraded"
	case Unhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// HealthCheck represents a health check result.
type HealthCheck struct {
	// Name is the component name.
	Name string `json:"name" yaml:"name"`
	// Status is the health status.
	Status HealthStatus `json:"status" yaml:"status"`
	// Message is an optional message.
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
	// Timestamp is when the check was performed.
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
	// Details contains additional details.
	Details map[string]interface{} `json:"details,omitempty" yaml:"details,omitempty"`
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		HalfOpenMaxCalls: 3,
	}
}

// DefaultRateLimitConfig returns a default rate limit configuration.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Algorithm: TokenBucket,
		Limit:     100,
		Window:    time.Minute,
		BurstSize: 10,
	}
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}
}

// DefaultTimeoutConfig returns a default timeout configuration.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Default:      30 * time.Second,
		Max:          5 * time.Minute,
		PerOperation: make(map[string]time.Duration),
	}
}

// DefaultBulkheadConfig returns a default bulkhead configuration.
func DefaultBulkheadConfig() BulkheadConfig {
	return BulkheadConfig{
		MaxConcurrent: 10,
		MaxQueue:      100,
		QueueTimeout:  5 * time.Second,
	}
}
