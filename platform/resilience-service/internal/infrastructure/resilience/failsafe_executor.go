// Package resilience provides failsafe-go based resilience implementations.
package resilience

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/authcorp/libs/go/src/functional"
	libfault "github.com/authcorp/libs/go/src/fault"
	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/interfaces"
	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/bulkhead"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/ratelimiter"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
	"github.com/failsafe-go/failsafe-go/timeout"
)

// FailsafeExecutor implements ResilienceExecutor using failsafe-go library.
type FailsafeExecutor struct {
	policies map[string]*PolicyExecutor
	metrics  interfaces.MetricsRecorder
	logger   *slog.Logger
	mu       sync.RWMutex
}

// PolicyExecutor wraps failsafe-go policies for a specific resilience policy.
type PolicyExecutor struct {
	name     string
	executor failsafe.Executor[any]
	policies []failsafe.Policy[any]
}

// NewFailsafeExecutor creates a new failsafe-go based executor.
func NewFailsafeExecutor(
	metrics interfaces.MetricsRecorder,
	logger *slog.Logger,
) *FailsafeExecutor {
	return &FailsafeExecutor{
		policies: make(map[string]*PolicyExecutor),
		metrics:  metrics,
		logger:   logger,
	}
}

// RegisterPolicy registers a resilience policy with the executor.
func (f *FailsafeExecutor) RegisterPolicy(policy *entities.Policy) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	policyExecutor, err := f.createPolicyExecutor(policy)
	if err != nil {
		return fmt.Errorf("failed to create policy executor: %w", err)
	}

	f.policies[policy.Name()] = policyExecutor

	f.logger.Info("policy registered",
		slog.String("policy_name", policy.Name()),
		slog.Int("version", policy.Version()))

	return nil
}

// UnregisterPolicy removes a policy from the executor.
func (f *FailsafeExecutor) UnregisterPolicy(policyName string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.policies, policyName)

	f.logger.Info("policy unregistered",
		slog.String("policy_name", policyName))
}

// Execute executes an operation with the specified resilience policy.
func (f *FailsafeExecutor) Execute(ctx context.Context, policyName string, operation func() error) error {
	f.mu.RLock()
	policyExecutor, exists := f.policies[policyName]
	f.mu.RUnlock()

	if !exists {
		return fmt.Errorf("policy '%s' not found", policyName)
	}

	start := time.Now()

	_, err := policyExecutor.executor.GetWithExecution(func(exec failsafe.Execution[any]) (any, error) {
		f.recordExecutionMetrics(ctx, policyName, exec)
		return nil, operation()
	})

	duration := time.Since(start)
	success := err == nil

	metrics := libfault.NewExecutionMetrics(policyName, duration, success)
	f.metrics.RecordExecution(ctx, metrics)

	return err
}

// ExecuteWithResult executes an operation with result and the specified resilience policy.
func (f *FailsafeExecutor) ExecuteWithResult(ctx context.Context, policyName string, operation func() (any, error)) functional.Result[any] {
	f.mu.RLock()
	policyExecutor, exists := f.policies[policyName]
	f.mu.RUnlock()

	if !exists {
		return functional.Err[any](fmt.Errorf("policy '%s' not found", policyName))
	}

	start := time.Now()

	result, err := policyExecutor.executor.GetWithExecution(func(exec failsafe.Execution[any]) (any, error) {
		f.recordExecutionMetrics(ctx, policyName, exec)
		return operation()
	})

	duration := time.Since(start)
	success := err == nil

	metrics := libfault.NewExecutionMetrics(policyName, duration, success)
	f.metrics.RecordExecution(ctx, metrics)

	if err != nil {
		return functional.Err[any](err)
	}
	return functional.Ok(result)
}

// createPolicyExecutor creates a failsafe-go executor from a resilience policy.
func (f *FailsafeExecutor) createPolicyExecutor(policy *entities.Policy) (*PolicyExecutor, error) {
	var policies []failsafe.Policy[any]

	if policy.CircuitBreaker().IsSome() {
		cb := policy.CircuitBreaker().Unwrap()
		cbPolicy := circuitbreaker.Builder[any]().
			WithFailureThreshold(uint(cb.FailureThreshold)).
			WithSuccessThreshold(uint(cb.SuccessThreshold)).
			WithDelay(cb.Timeout).
			OnStateChanged(func(event circuitbreaker.StateChangedEvent) {
				f.logger.Info("circuit breaker state changed",
					slog.String("policy_name", policy.Name()),
					slog.String("old_state", event.OldState.String()),
					slog.String("new_state", event.NewState.String()))
			}).
			Build()
		policies = append(policies, cbPolicy)
	}

	if policy.Retry().IsSome() {
		retry := policy.Retry().Unwrap()
		jitterDuration := time.Duration(float64(retry.BaseDelay) * retry.JitterPercent)
		retryPolicy := retrypolicy.Builder[any]().
			WithMaxAttempts(retry.MaxAttempts).
			WithBackoff(retry.BaseDelay, retry.MaxDelay).
			WithJitter(jitterDuration).
			OnRetryScheduled(func(event failsafe.ExecutionScheduledEvent[any]) {
				f.logger.Debug("retry scheduled",
					slog.String("policy_name", policy.Name()),
					slog.Int("attempt", event.Attempts()),
					slog.Duration("delay", event.Delay))
			}).
			Build()
		policies = append(policies, retryPolicy)
	}

	if policy.Timeout().IsSome() {
		to := policy.Timeout().Unwrap()
		timeoutPolicy := timeout.With[any](to.Default)
		policies = append(policies, timeoutPolicy)
	}

	if policy.RateLimit().IsSome() {
		rl := policy.RateLimit().Unwrap()
		var rateLimiter failsafe.Policy[any]

		switch rl.Algorithm {
		case "token_bucket":
			interval := time.Duration(rl.Window.Nanoseconds() / int64(rl.Limit))
			rateLimiter = ratelimiter.BurstyBuilder[any](uint(rl.Limit), interval).
				WithMaxWaitTime(time.Second).
				Build()
		case "sliding_window":
			rateLimiter = ratelimiter.SmoothBuilder[any](uint(rl.Limit), rl.Window).
				WithMaxWaitTime(time.Second).
				Build()
		default:
			return nil, fmt.Errorf("unsupported rate limit algorithm: %s", rl.Algorithm)
		}

		policies = append(policies, rateLimiter)
	}

	if policy.Bulkhead().IsSome() {
		bh := policy.Bulkhead().Unwrap()
		bulkheadPolicy := bulkhead.Builder[any](uint(bh.MaxConcurrent)).
			WithMaxWaitTime(bh.QueueTimeout).
			Build()
		policies = append(policies, bulkheadPolicy)
	}

	if len(policies) == 0 {
		return nil, fmt.Errorf("policy '%s' has no resilience patterns configured", policy.Name())
	}

	executor := failsafe.NewExecutor[any](policies...)

	return &PolicyExecutor{
		name:     policy.Name(),
		executor: executor,
		policies: policies,
	}, nil
}

// recordExecutionMetrics records metrics during execution.
func (f *FailsafeExecutor) recordExecutionMetrics(ctx context.Context, policyName string, exec failsafe.Execution[any]) {
	if exec.Attempts() > 1 {
		f.metrics.RecordRetryAttempt(ctx, policyName, exec.Attempts())
	}
}

// GetPolicyNames returns the names of all registered policies.
func (f *FailsafeExecutor) GetPolicyNames() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.policies))
	for name := range f.policies {
		names = append(names, name)
	}

	return names
}

// GetPolicyExecutor returns the policy executor for a given policy name.
func (f *FailsafeExecutor) GetPolicyExecutor(policyName string) (*PolicyExecutor, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	executor, exists := f.policies[policyName]
	return executor, exists
}

// Ensure FailsafeExecutor implements ResilienceExecutor.
var _ interfaces.ResilienceExecutor = (*FailsafeExecutor)(nil)
