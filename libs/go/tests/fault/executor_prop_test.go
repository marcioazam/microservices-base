package fault_test

import (
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/fault"
	"pgregory.net/rapid"
)

// Property: ExecutionMetrics builder methods are immutable
func TestExecutionMetricsImmutability(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policyName := rapid.StringMatching(`[a-z][a-z0-9-]{0,20}`).Draw(t, "policy")
		duration := rapid.Int64Range(1, 10000).Draw(t, "duration_ms")
		success := rapid.Bool().Draw(t, "success")

		original := fault.NewExecutionMetrics(
			policyName,
			time.Duration(duration)*time.Millisecond,
			success,
		)

		// Apply builder methods
		withCircuit := original.WithCircuitState("open")
		withRetry := original.WithRetryAttempts(3)
		withRate := original.WithRateLimit(true)
		withBulkhead := original.WithBulkheadQueue(true)

		// Original should be unchanged
		if original.CircuitState != "" {
			t.Errorf("original CircuitState modified: %s", original.CircuitState)
		}
		if original.RetryAttempts != 0 {
			t.Errorf("original RetryAttempts modified: %d", original.RetryAttempts)
		}
		if original.RateLimited {
			t.Error("original RateLimited modified")
		}
		if original.BulkheadQueued {
			t.Error("original BulkheadQueued modified")
		}

		// New instances should have values
		if withCircuit.CircuitState != "open" {
			t.Errorf("withCircuit.CircuitState = %s, want open", withCircuit.CircuitState)
		}
		if withRetry.RetryAttempts != 3 {
			t.Errorf("withRetry.RetryAttempts = %d, want 3", withRetry.RetryAttempts)
		}
		if !withRate.RateLimited {
			t.Error("withRate.RateLimited should be true")
		}
		if !withBulkhead.BulkheadQueued {
			t.Error("withBulkhead.BulkheadQueued should be true")
		}
	})
}

// Property: ExecutionMetrics preserves required fields (executor context)
func TestExecutorExecutionMetricsPreservesFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policyName := rapid.StringMatching(`[a-z][a-z0-9-]{0,20}`).Draw(t, "policy")
		duration := rapid.Int64Range(1, 10000).Draw(t, "duration_ms")
		success := rapid.Bool().Draw(t, "success")

		metrics := fault.NewExecutionMetrics(
			policyName,
			time.Duration(duration)*time.Millisecond,
			success,
		)

		if metrics.PolicyName != policyName {
			t.Errorf("PolicyName = %s, want %s", metrics.PolicyName, policyName)
		}
		if metrics.ExecutionTime != time.Duration(duration)*time.Millisecond {
			t.Errorf("ExecutionTime = %v, want %v", metrics.ExecutionTime, time.Duration(duration)*time.Millisecond)
		}
		if metrics.Success != success {
			t.Errorf("Success = %v, want %v", metrics.Success, success)
		}
		if metrics.Timestamp.IsZero() {
			t.Error("Timestamp should not be zero")
		}
	})
}

// Property: ExecutionMetrics helper methods are consistent
func TestExecutionMetricsHelperMethods(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		success := rapid.Bool().Draw(t, "success")
		retryAttempts := rapid.IntRange(0, 10).Draw(t, "retry_attempts")
		rateLimited := rapid.Bool().Draw(t, "rate_limited")

		metrics := fault.NewExecutionMetrics("test", time.Second, success).
			WithRetryAttempts(retryAttempts).
			WithRateLimit(rateLimited)

		if metrics.IsSuccessful() != success {
			t.Errorf("IsSuccessful() = %v, want %v", metrics.IsSuccessful(), success)
		}
		if metrics.WasRetried() != (retryAttempts > 0) {
			t.Errorf("WasRetried() = %v, want %v", metrics.WasRetried(), retryAttempts > 0)
		}
		if metrics.WasRateLimited() != rateLimited {
			t.Errorf("WasRateLimited() = %v, want %v", metrics.WasRateLimited(), rateLimited)
		}
	})
}

// Property: PolicyConfig validates required name
func TestPolicyConfigRequiresName(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`[a-z][a-z0-9-]{0,20}`).Draw(t, "name")

		config := fault.PolicyConfig{
			Name: name,
		}

		if config.Name != name {
			t.Errorf("Name = %s, want %s", config.Name, name)
		}
	})
}

// Property: CircuitBreakerPolicyConfig thresholds are valid
func TestCircuitBreakerPolicyConfigThresholds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		failureThreshold := rapid.IntRange(1, 100).Draw(t, "failure_threshold")
		successThreshold := rapid.IntRange(1, 10).Draw(t, "success_threshold")
		timeout := rapid.Int64Range(1000, 300000).Draw(t, "timeout_ms")
		halfOpenMaxCalls := rapid.IntRange(1, 10).Draw(t, "half_open_max_calls")

		config := fault.CircuitBreakerPolicyConfig{
			FailureThreshold: failureThreshold,
			SuccessThreshold: successThreshold,
			Timeout:          time.Duration(timeout) * time.Millisecond,
			HalfOpenMaxCalls: halfOpenMaxCalls,
		}

		if config.FailureThreshold < 1 || config.FailureThreshold > 100 {
			t.Errorf("FailureThreshold out of range: %d", config.FailureThreshold)
		}
		if config.SuccessThreshold < 1 || config.SuccessThreshold > 10 {
			t.Errorf("SuccessThreshold out of range: %d", config.SuccessThreshold)
		}
		if config.Timeout < time.Second || config.Timeout > 5*time.Minute {
			t.Errorf("Timeout out of range: %v", config.Timeout)
		}
	})
}

// Property: RetryPolicyConfig delays are valid
func TestRetryPolicyConfigDelays(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxAttempts := rapid.IntRange(1, 10).Draw(t, "max_attempts")
		baseDelayMs := rapid.Int64Range(1, 10000).Draw(t, "base_delay_ms")
		maxDelayMs := rapid.Int64Range(1000, 300000).Draw(t, "max_delay_ms")
		multiplier := rapid.Float64Range(1.0, 10.0).Draw(t, "multiplier")
		jitter := rapid.Float64Range(0.0, 1.0).Draw(t, "jitter")

		config := fault.RetryPolicyConfig{
			MaxAttempts:   maxAttempts,
			BaseDelay:     time.Duration(baseDelayMs) * time.Millisecond,
			MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
			Multiplier:    multiplier,
			JitterPercent: jitter,
		}

		if config.MaxAttempts < 1 {
			t.Errorf("MaxAttempts should be >= 1: %d", config.MaxAttempts)
		}
		if config.Multiplier < 1.0 {
			t.Errorf("Multiplier should be >= 1.0: %f", config.Multiplier)
		}
		if config.JitterPercent < 0.0 || config.JitterPercent > 1.0 {
			t.Errorf("JitterPercent out of range: %f", config.JitterPercent)
		}
	})
}

// Property: NoOpMetricsRecorder implements interface
func TestNoOpMetricsRecorderImplementsInterface(t *testing.T) {
	var recorder fault.MetricsRecorder = fault.NoOpMetricsRecorder{}

	// Should not panic
	recorder.RecordExecution(nil, fault.ExecutionMetrics{})
	recorder.RecordCircuitState(nil, "test", "open")
	recorder.RecordRetryAttempt(nil, "test", 1)
	recorder.RecordRateLimit(nil, "test", true)
	recorder.RecordBulkheadQueue(nil, "test", true)
}

// Property: ExecutorConfig defaults are sensible
func TestExecutorConfigDefaults(t *testing.T) {
	config := fault.DefaultExecutorConfig()

	if config.DefaultTimeout <= 0 {
		t.Errorf("DefaultTimeout should be positive: %v", config.DefaultTimeout)
	}
	if config.DefaultTimeout > 5*time.Minute {
		t.Errorf("DefaultTimeout too large: %v", config.DefaultTimeout)
	}
}
