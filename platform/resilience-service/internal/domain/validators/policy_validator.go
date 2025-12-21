// Package validators provides domain validation implementations with Result types.
package validators

import (
	"fmt"

	"github.com/authcorp/libs/go/src/functional"
	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/interfaces"
)

// PolicyValidator validates policy configurations using Result types.
type PolicyValidator struct{}

// NewPolicyValidator creates a new policy validator.
func NewPolicyValidator() *PolicyValidator {
	return &PolicyValidator{}
}

// Validate validates a complete policy and returns Result.
func (v *PolicyValidator) Validate(policy *entities.Policy) functional.Result[*entities.Policy] {
	if policy == nil {
		return functional.Err[*entities.Policy](fmt.Errorf("policy cannot be nil"))
	}

	if policy.Name() == "" {
		return functional.Err[*entities.Policy](fmt.Errorf("policy name cannot be empty"))
	}

	// Validate individual configurations using Option
	if policy.CircuitBreaker().IsSome() {
		if result := v.ValidateCircuitBreaker(policy.CircuitBreaker().Unwrap()); result.IsErr() {
			return functional.Err[*entities.Policy](fmt.Errorf("circuit breaker: %w", result.UnwrapErr()))
		}
	}

	if policy.Retry().IsSome() {
		if result := v.ValidateRetry(policy.Retry().Unwrap()); result.IsErr() {
			return functional.Err[*entities.Policy](fmt.Errorf("retry: %w", result.UnwrapErr()))
		}
	}

	if policy.Timeout().IsSome() {
		if result := v.ValidateTimeout(policy.Timeout().Unwrap()); result.IsErr() {
			return functional.Err[*entities.Policy](fmt.Errorf("timeout: %w", result.UnwrapErr()))
		}
	}

	if policy.RateLimit().IsSome() {
		if result := v.ValidateRateLimit(policy.RateLimit().Unwrap()); result.IsErr() {
			return functional.Err[*entities.Policy](fmt.Errorf("rate limit: %w", result.UnwrapErr()))
		}
	}

	if policy.Bulkhead().IsSome() {
		if result := v.ValidateBulkhead(policy.Bulkhead().Unwrap()); result.IsErr() {
			return functional.Err[*entities.Policy](fmt.Errorf("bulkhead: %w", result.UnwrapErr()))
		}
	}

	return functional.Ok(policy)
}

// ValidateCircuitBreaker validates circuit breaker configuration.
func (v *PolicyValidator) ValidateCircuitBreaker(config *entities.CircuitBreakerConfig) functional.Result[*entities.CircuitBreakerConfig] {
	if config == nil {
		return functional.Err[*entities.CircuitBreakerConfig](fmt.Errorf("config cannot be nil"))
	}
	return config.ValidateResult()
}

// ValidateRetry validates retry configuration.
func (v *PolicyValidator) ValidateRetry(config *entities.RetryConfig) functional.Result[*entities.RetryConfig] {
	if config == nil {
		return functional.Err[*entities.RetryConfig](fmt.Errorf("config cannot be nil"))
	}
	return config.ValidateResult()
}

// ValidateTimeout validates timeout configuration.
func (v *PolicyValidator) ValidateTimeout(config *entities.TimeoutConfig) functional.Result[*entities.TimeoutConfig] {
	if config == nil {
		return functional.Err[*entities.TimeoutConfig](fmt.Errorf("config cannot be nil"))
	}
	return config.ValidateResult()
}

// ValidateRateLimit validates rate limit configuration.
func (v *PolicyValidator) ValidateRateLimit(config *entities.RateLimitConfig) functional.Result[*entities.RateLimitConfig] {
	if config == nil {
		return functional.Err[*entities.RateLimitConfig](fmt.Errorf("config cannot be nil"))
	}
	return config.ValidateResult()
}

// ValidateBulkhead validates bulkhead configuration.
func (v *PolicyValidator) ValidateBulkhead(config *entities.BulkheadConfig) functional.Result[*entities.BulkheadConfig] {
	if config == nil {
		return functional.Err[*entities.BulkheadConfig](fmt.Errorf("config cannot be nil"))
	}
	return config.ValidateResult()
}

// Ensure PolicyValidator implements interfaces.PolicyValidator.
var _ interfaces.PolicyValidator = (*PolicyValidator)(nil)
