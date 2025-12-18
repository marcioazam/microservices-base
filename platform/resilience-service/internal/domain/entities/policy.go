// Package entities defines domain entities with pure business logic.
package entities

import (
	"fmt"
	"time"
)

// Policy represents a resilience policy configuration.
type Policy struct {
	name           string
	version        int
	circuitBreaker *CircuitBreakerConfig
	retry          *RetryConfig
	timeout        *TimeoutConfig
	rateLimit      *RateLimitConfig
	bulkhead       *BulkheadConfig
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
		name:      name,
		version:   1,
		createdAt: now,
		updatedAt: now,
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
func (p *Policy) SetCircuitBreaker(config *CircuitBreakerConfig) error {
	if config == nil {
		return fmt.Errorf("circuit breaker config cannot be nil")
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid circuit breaker config: %w", err)
	}
	p.circuitBreaker = config
	p.updatedAt = time.Now().UTC()
	return nil
}

// CircuitBreaker returns the circuit breaker configuration.
func (p *Policy) CircuitBreaker() *CircuitBreakerConfig {
	return p.circuitBreaker
}

// SetRetry sets the retry configuration.
func (p *Policy) SetRetry(config *RetryConfig) error {
	if config == nil {
		return fmt.Errorf("retry config cannot be nil")
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid retry config: %w", err)
	}
	p.retry = config
	p.updatedAt = time.Now().UTC()
	return nil
}

// Retry returns the retry configuration.
func (p *Policy) Retry() *RetryConfig {
	return p.retry
}

// SetTimeout sets the timeout configuration.
func (p *Policy) SetTimeout(config *TimeoutConfig) error {
	if config == nil {
		return fmt.Errorf("timeout config cannot be nil")
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid timeout config: %w", err)
	}
	p.timeout = config
	p.updatedAt = time.Now().UTC()
	return nil
}

// Timeout returns the timeout configuration.
func (p *Policy) Timeout() *TimeoutConfig {
	return p.timeout
}

// SetRateLimit sets the rate limit configuration.
func (p *Policy) SetRateLimit(config *RateLimitConfig) error {
	if config == nil {
		return fmt.Errorf("rate limit config cannot be nil")
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid rate limit config: %w", err)
	}
	p.rateLimit = config
	p.updatedAt = time.Now().UTC()
	return nil
}

// RateLimit returns the rate limit configuration.
func (p *Policy) RateLimit() *RateLimitConfig {
	return p.rateLimit
}

// SetBulkhead sets the bulkhead configuration.
func (p *Policy) SetBulkhead(config *BulkheadConfig) error {
	if config == nil {
		return fmt.Errorf("bulkhead config cannot be nil")
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid bulkhead config: %w", err)
	}
	p.bulkhead = config
	p.updatedAt = time.Now().UTC()
	return nil
}

// Bulkhead returns the bulkhead configuration.
func (p *Policy) Bulkhead() *BulkheadConfig {
	return p.bulkhead
}

// IncrementVersion increments the policy version.
func (p *Policy) IncrementVersion() {
	p.version++
	p.updatedAt = time.Now().UTC()
}

// Validate validates the complete policy configuration.
func (p *Policy) Validate() error {
	if p.name == "" {
		return fmt.Errorf("policy name cannot be empty")
	}

	if p.version < 1 {
		return fmt.Errorf("policy version must be positive")
	}

	// At least one resilience pattern must be configured
	hasPattern := p.circuitBreaker != nil || p.retry != nil || p.timeout != nil || 
		p.rateLimit != nil || p.bulkhead != nil

	if !hasPattern {
		return fmt.Errorf("policy must have at least one resilience pattern configured")
	}

	// Validate individual configurations if present
	if p.circuitBreaker != nil {
		if err := p.circuitBreaker.Validate(); err != nil {
			return fmt.Errorf("circuit breaker validation failed: %w", err)
		}
	}

	if p.retry != nil {
		if err := p.retry.Validate(); err != nil {
			return fmt.Errorf("retry validation failed: %w", err)
		}
	}

	if p.timeout != nil {
		if err := p.timeout.Validate(); err != nil {
			return fmt.Errorf("timeout validation failed: %w", err)
		}
	}

	if p.rateLimit != nil {
		if err := p.rateLimit.Validate(); err != nil {
			return fmt.Errorf("rate limit validation failed: %w", err)
		}
	}

	if p.bulkhead != nil {
		if err := p.bulkhead.Validate(); err != nil {
			return fmt.Errorf("bulkhead validation failed: %w", err)
		}
	}

	return nil
}

// Clone creates a deep copy of the policy.
func (p *Policy) Clone() *Policy {
	clone := &Policy{
		name:      p.name,
		version:   p.version,
		createdAt: p.createdAt,
		updatedAt: p.updatedAt,
	}

	if p.circuitBreaker != nil {
		clone.circuitBreaker = p.circuitBreaker.Clone()
	}

	if p.retry != nil {
		clone.retry = p.retry.Clone()
	}

	if p.timeout != nil {
		clone.timeout = p.timeout.Clone()
	}

	if p.rateLimit != nil {
		clone.rateLimit = p.rateLimit.Clone()
	}

	if p.bulkhead != nil {
		clone.bulkhead = p.bulkhead.Clone()
	}

	return clone
}