// Package entities defines domain entities with pure business logic.
package entities

import (
	"fmt"
	"time"

	"github.com/authcorp/libs/go/src/functional"
)

// Policy represents a resilience policy configuration with type-safe optional configs.
type Policy struct {
	name           string
	version        int
	circuitBreaker functional.Option[*CircuitBreakerConfig]
	retry          functional.Option[*RetryConfig]
	timeout        functional.Option[*TimeoutConfig]
	rateLimit      functional.Option[*RateLimitConfig]
	bulkhead       functional.Option[*BulkheadConfig]
	createdAt      time.Time
	updatedAt      time.Time
}

// NewPolicy creates a new policy with validation.
func NewPolicy(name string) (*Policy, error) {
	if name == "" {
		return nil, fmt.Errorf("policy name cannot be empty")
	}

	now := time.Now().UTC()
	return &Policy{
		name:           name,
		version:        1,
		circuitBreaker: functional.None[*CircuitBreakerConfig](),
		retry:          functional.None[*RetryConfig](),
		timeout:        functional.None[*TimeoutConfig](),
		rateLimit:      functional.None[*RateLimitConfig](),
		bulkhead:       functional.None[*BulkheadConfig](),
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// Name returns the policy name.
func (p *Policy) Name() string {
	return p.name
}

// Version returns the policy version.
func (p *Policy) Version() int {
	return p.version
}

// CreatedAt returns the creation timestamp.
func (p *Policy) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns the last update timestamp.
func (p *Policy) UpdatedAt() time.Time {
	return p.updatedAt
}

// SetCircuitBreaker sets the circuit breaker configuration.
func (p *Policy) SetCircuitBreaker(config *CircuitBreakerConfig) functional.Result[*Policy] {
	if config == nil {
		return functional.Err[*Policy](fmt.Errorf("circuit breaker config cannot be nil"))
	}
	result := config.ValidateResult()
	if result.IsErr() {
		return functional.Err[*Policy](fmt.Errorf("invalid circuit breaker config: %w", result.UnwrapErr()))
	}
	p.circuitBreaker = functional.Some(config)
	p.updatedAt = time.Now().UTC()
	return functional.Ok(p)
}

// CircuitBreaker returns the circuit breaker configuration as Option.
func (p *Policy) CircuitBreaker() functional.Option[*CircuitBreakerConfig] {
	return p.circuitBreaker
}

// SetRetry sets the retry configuration.
func (p *Policy) SetRetry(config *RetryConfig) functional.Result[*Policy] {
	if config == nil {
		return functional.Err[*Policy](fmt.Errorf("retry config cannot be nil"))
	}
	result := config.ValidateResult()
	if result.IsErr() {
		return functional.Err[*Policy](fmt.Errorf("invalid retry config: %w", result.UnwrapErr()))
	}
	p.retry = functional.Some(config)
	p.updatedAt = time.Now().UTC()
	return functional.Ok(p)
}

// Retry returns the retry configuration as Option.
func (p *Policy) Retry() functional.Option[*RetryConfig] {
	return p.retry
}

// SetTimeout sets the timeout configuration.
func (p *Policy) SetTimeout(config *TimeoutConfig) functional.Result[*Policy] {
	if config == nil {
		return functional.Err[*Policy](fmt.Errorf("timeout config cannot be nil"))
	}
	result := config.ValidateResult()
	if result.IsErr() {
		return functional.Err[*Policy](fmt.Errorf("invalid timeout config: %w", result.UnwrapErr()))
	}
	p.timeout = functional.Some(config)
	p.updatedAt = time.Now().UTC()
	return functional.Ok(p)
}

// Timeout returns the timeout configuration as Option.
func (p *Policy) Timeout() functional.Option[*TimeoutConfig] {
	return p.timeout
}

// SetRateLimit sets the rate limit configuration.
func (p *Policy) SetRateLimit(config *RateLimitConfig) functional.Result[*Policy] {
	if config == nil {
		return functional.Err[*Policy](fmt.Errorf("rate limit config cannot be nil"))
	}
	result := config.ValidateResult()
	if result.IsErr() {
		return functional.Err[*Policy](fmt.Errorf("invalid rate limit config: %w", result.UnwrapErr()))
	}
	p.rateLimit = functional.Some(config)
	p.updatedAt = time.Now().UTC()
	return functional.Ok(p)
}

// RateLimit returns the rate limit configuration as Option.
func (p *Policy) RateLimit() functional.Option[*RateLimitConfig] {
	return p.rateLimit
}

// SetBulkhead sets the bulkhead configuration.
func (p *Policy) SetBulkhead(config *BulkheadConfig) functional.Result[*Policy] {
	if config == nil {
		return functional.Err[*Policy](fmt.Errorf("bulkhead config cannot be nil"))
	}
	result := config.ValidateResult()
	if result.IsErr() {
		return functional.Err[*Policy](fmt.Errorf("invalid bulkhead config: %w", result.UnwrapErr()))
	}
	p.bulkhead = functional.Some(config)
	p.updatedAt = time.Now().UTC()
	return functional.Ok(p)
}

// Bulkhead returns the bulkhead configuration as Option.
func (p *Policy) Bulkhead() functional.Option[*BulkheadConfig] {
	return p.bulkhead
}

// IncrementVersion increments the policy version.
func (p *Policy) IncrementVersion() {
	p.version++
	p.updatedAt = time.Now().UTC()
}

// HasAnyPattern returns true if at least one resilience pattern is configured.
func (p *Policy) HasAnyPattern() bool {
	return p.circuitBreaker.IsSome() || p.retry.IsSome() || p.timeout.IsSome() ||
		p.rateLimit.IsSome() || p.bulkhead.IsSome()
}

// Validate validates the complete policy configuration.
func (p *Policy) Validate() error {
	return p.ValidateResult().UnwrapErr()
}

// ValidateResult validates and returns Result for functional composition.
func (p *Policy) ValidateResult() functional.Result[*Policy] {
	if p.name == "" {
		return functional.Err[*Policy](fmt.Errorf("policy name cannot be empty"))
	}

	if p.version < 1 {
		return functional.Err[*Policy](fmt.Errorf("policy version must be positive"))
	}

	if !p.HasAnyPattern() {
		return functional.Err[*Policy](fmt.Errorf("policy must have at least one resilience pattern"))
	}

	// Validate individual configurations if present
	if p.circuitBreaker.IsSome() {
		if result := p.circuitBreaker.Unwrap().ValidateResult(); result.IsErr() {
			return functional.Err[*Policy](fmt.Errorf("circuit breaker: %w", result.UnwrapErr()))
		}
	}

	if p.retry.IsSome() {
		if result := p.retry.Unwrap().ValidateResult(); result.IsErr() {
			return functional.Err[*Policy](fmt.Errorf("retry: %w", result.UnwrapErr()))
		}
	}

	if p.timeout.IsSome() {
		if result := p.timeout.Unwrap().ValidateResult(); result.IsErr() {
			return functional.Err[*Policy](fmt.Errorf("timeout: %w", result.UnwrapErr()))
		}
	}

	if p.rateLimit.IsSome() {
		if result := p.rateLimit.Unwrap().ValidateResult(); result.IsErr() {
			return functional.Err[*Policy](fmt.Errorf("rate limit: %w", result.UnwrapErr()))
		}
	}

	if p.bulkhead.IsSome() {
		if result := p.bulkhead.Unwrap().ValidateResult(); result.IsErr() {
			return functional.Err[*Policy](fmt.Errorf("bulkhead: %w", result.UnwrapErr()))
		}
	}

	return functional.Ok(p)
}

// Clone creates a deep copy of the policy.
func (p *Policy) Clone() *Policy {
	clone := &Policy{
		name:           p.name,
		version:        p.version,
		circuitBreaker: functional.None[*CircuitBreakerConfig](),
		retry:          functional.None[*RetryConfig](),
		timeout:        functional.None[*TimeoutConfig](),
		rateLimit:      functional.None[*RateLimitConfig](),
		bulkhead:       functional.None[*BulkheadConfig](),
		createdAt:      p.createdAt,
		updatedAt:      p.updatedAt,
	}

	if p.circuitBreaker.IsSome() {
		clone.circuitBreaker = functional.Some(p.circuitBreaker.Unwrap().Clone())
	}
	if p.retry.IsSome() {
		clone.retry = functional.Some(p.retry.Unwrap().Clone())
	}
	if p.timeout.IsSome() {
		clone.timeout = functional.Some(p.timeout.Unwrap().Clone())
	}
	if p.rateLimit.IsSome() {
		clone.rateLimit = functional.Some(p.rateLimit.Unwrap().Clone())
	}
	if p.bulkhead.IsSome() {
		clone.bulkhead = functional.Some(p.bulkhead.Unwrap().Clone())
	}

	return clone
}
