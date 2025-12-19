package resilience_test

import (
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/resilience"
	"pgregory.net/rapid"
)

// Property 5: Invalid Configuration Detection
// Validate() returns InvalidPolicyError for invalid configs.
func TestProperty_InvalidConfigDetection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Test CircuitBreakerConfig with invalid values
		invalidThreshold := rapid.IntRange(-100, 0).Draw(t, "invalidThreshold")
		cfg := resilience.CircuitBreakerConfig{
			FailureThreshold: invalidThreshold,
			SuccessThreshold: 2,
			Timeout:          time.Second,
			HalfOpenRequests: 1,
		}
		
		err := cfg.Validate()
		if err == nil {
			t.Fatalf("Expected error for invalid FailureThreshold: %d", invalidThreshold)
		}
		if !resilience.IsInvalidPolicy(err) {
			t.Fatalf("Expected InvalidPolicyError, got: %T", err)
		}
	})
}

// Property: Valid configs pass validation
func TestProperty_ValidConfigPasses(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		failureThreshold := rapid.IntRange(1, 100).Draw(t, "failureThreshold")
		successThreshold := rapid.IntRange(1, 100).Draw(t, "successThreshold")
		timeoutSec := rapid.IntRange(1, 300).Draw(t, "timeoutSec")
		halfOpen := rapid.IntRange(1, 10).Draw(t, "halfOpen")
		
		cfg := resilience.CircuitBreakerConfig{
			FailureThreshold: failureThreshold,
			SuccessThreshold: successThreshold,
			Timeout:          time.Duration(timeoutSec) * time.Second,
			HalfOpenRequests: halfOpen,
		}
		
		err := cfg.Validate()
		if err != nil {
			t.Fatalf("Valid config should pass validation: %v", err)
		}
	})
}

// Property: RetryConfig validation
func TestProperty_RetryConfigValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Invalid: MaxAttempts <= 0
		invalidAttempts := rapid.IntRange(-100, 0).Draw(t, "invalidAttempts")
		cfg := resilience.RetryConfig{
			MaxAttempts:     invalidAttempts,
			InitialInterval: time.Millisecond * 100,
			MaxInterval:     time.Second,
			Multiplier:      2.0,
		}
		
		err := cfg.Validate()
		if err == nil {
			t.Fatalf("Expected error for invalid MaxAttempts: %d", invalidAttempts)
		}
		if !resilience.IsInvalidPolicy(err) {
			t.Fatalf("Expected InvalidPolicyError, got: %T", err)
		}
	})
}

// Property: RetryConfig MaxInterval >= InitialInterval
func TestProperty_RetryConfigIntervalOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		initialMs := rapid.IntRange(100, 1000).Draw(t, "initialMs")
		maxMs := rapid.IntRange(1, initialMs-1).Draw(t, "maxMs")
		
		cfg := resilience.RetryConfig{
			MaxAttempts:     3,
			InitialInterval: time.Duration(initialMs) * time.Millisecond,
			MaxInterval:     time.Duration(maxMs) * time.Millisecond,
			Multiplier:      2.0,
		}
		
		err := cfg.Validate()
		if err == nil {
			t.Fatalf("Expected error when MaxInterval < InitialInterval")
		}
	})
}

// Property: RateLimitConfig validation
func TestProperty_RateLimitConfigValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invalidRate := rapid.IntRange(-100, 0).Draw(t, "invalidRate")
		cfg := resilience.RateLimitConfig{
			Rate:      invalidRate,
			Window:    time.Second,
			BurstSize: 10,
		}
		
		err := cfg.Validate()
		if err == nil {
			t.Fatalf("Expected error for invalid Rate: %d", invalidRate)
		}
	})
}

// Property: BulkheadConfig validation
func TestProperty_BulkheadConfigValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invalidConcurrent := rapid.IntRange(-100, 0).Draw(t, "invalidConcurrent")
		cfg := resilience.BulkheadConfig{
			MaxConcurrent: invalidConcurrent,
			MaxWait:       time.Second,
			QueueSize:     100,
		}
		
		err := cfg.Validate()
		if err == nil {
			t.Fatalf("Expected error for invalid MaxConcurrent: %d", invalidConcurrent)
		}
	})
}

// Property: Functional options apply correctly
func TestProperty_FunctionalOptions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.String().Draw(t, "name")
		threshold := rapid.IntRange(1, 100).Draw(t, "threshold")
		timeoutSec := rapid.IntRange(1, 300).Draw(t, "timeoutSec")
		
		cfg := resilience.NewCircuitBreakerConfig(
			name,
			resilience.WithFailureThreshold(threshold),
			resilience.WithCircuitTimeout(time.Duration(timeoutSec)*time.Second),
		)
		
		if cfg.Name != name {
			t.Fatalf("Name not set: expected %s, got %s", name, cfg.Name)
		}
		if cfg.FailureThreshold != threshold {
			t.Fatalf("FailureThreshold not set: expected %d, got %d", threshold, cfg.FailureThreshold)
		}
		if cfg.Timeout != time.Duration(timeoutSec)*time.Second {
			t.Fatalf("Timeout not set correctly")
		}
	})
}

// Property: Default configs are valid
func TestProperty_DefaultConfigsValid(t *testing.T) {
	// CircuitBreaker
	cbCfg := resilience.DefaultCircuitBreakerConfig()
	if err := cbCfg.Validate(); err != nil {
		t.Fatalf("Default CircuitBreakerConfig should be valid: %v", err)
	}
	
	// Retry
	retryCfg := resilience.DefaultRetryConfig()
	if err := retryCfg.Validate(); err != nil {
		t.Fatalf("Default RetryConfig should be valid: %v", err)
	}
	
	// RateLimit
	rateCfg := resilience.DefaultRateLimitConfig()
	if err := rateCfg.Validate(); err != nil {
		t.Fatalf("Default RateLimitConfig should be valid: %v", err)
	}
	
	// Bulkhead
	bulkCfg := resilience.DefaultBulkheadConfig()
	if err := bulkCfg.Validate(); err != nil {
		t.Fatalf("Default BulkheadConfig should be valid: %v", err)
	}
	
	// Timeout
	timeoutCfg := resilience.DefaultTimeoutConfig()
	if err := timeoutCfg.Validate(); err != nil {
		t.Fatalf("Default TimeoutConfig should be valid: %v", err)
	}
}
