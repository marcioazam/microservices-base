// Package policy implements resilience policy management.
package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"gopkg.in/yaml.v3"
)

// Engine implements the PolicyEngine interface.
type Engine struct {
	mu             sync.RWMutex
	policies       map[string]*domain.ResiliencePolicy
	configPath     string
	reloadInterval time.Duration
	stopCh         chan struct{}
	eventCh        chan domain.PolicyEvent
}

// Config holds engine creation options.
type Config struct {
	ConfigPath     string
	ReloadInterval time.Duration
}

// NewEngine creates a new policy engine.
func NewEngine(cfg Config) *Engine {
	return &Engine{
		policies:       make(map[string]*domain.ResiliencePolicy),
		configPath:     cfg.ConfigPath,
		reloadInterval: cfg.ReloadInterval,
		stopCh:         make(chan struct{}),
		eventCh:        make(chan domain.PolicyEvent, 100),
	}
}

// GetPolicy retrieves policy by name.
func (e *Engine) GetPolicy(name string) (*domain.ResiliencePolicy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policy, ok := e.policies[name]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", name)
	}

	return policy, nil
}

// UpdatePolicy updates or creates a policy.
func (e *Engine) UpdatePolicy(policy *domain.ResiliencePolicy) error {
	if err := e.Validate(policy); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	existing, exists := e.policies[policy.Name]
	eventType := domain.PolicyCreated
	if exists {
		eventType = domain.PolicyUpdated
		policy.Version = existing.Version + 1
	} else {
		policy.Version = 1
	}

	e.policies[policy.Name] = policy

	// Emit event
	select {
	case e.eventCh <- domain.PolicyEvent{Type: eventType, Policy: policy}:
	default:
	}

	return nil
}

// DeletePolicy removes a policy.
func (e *Engine) DeletePolicy(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	policy, ok := e.policies[name]
	if !ok {
		return fmt.Errorf("policy not found: %s", name)
	}

	delete(e.policies, name)

	// Emit event
	select {
	case e.eventCh <- domain.PolicyEvent{Type: domain.PolicyDeleted, Policy: policy}:
	default:
	}

	return nil
}

// ListPolicies returns all policies.
func (e *Engine) ListPolicies() ([]*domain.ResiliencePolicy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*domain.ResiliencePolicy, 0, len(e.policies))
	for _, p := range e.policies {
		result = append(result, p)
	}

	return result, nil
}

// WatchPolicies streams policy changes.
func (e *Engine) WatchPolicies(ctx context.Context) (<-chan domain.PolicyEvent, error) {
	return e.eventCh, nil
}

// Validate validates a policy configuration.
func (e *Engine) Validate(policy *domain.ResiliencePolicy) error {
	if policy.Name == "" {
		return domain.NewInvalidPolicyError("policy name is required")
	}

	if policy.CircuitBreaker != nil {
		if err := validateCircuitBreaker(policy.CircuitBreaker); err != nil {
			return err
		}
	}

	if policy.Retry != nil {
		if err := validateRetry(policy.Retry); err != nil {
			return err
		}
	}

	if policy.Timeout != nil {
		if err := validateTimeout(policy.Timeout); err != nil {
			return err
		}
	}

	if policy.RateLimit != nil {
		if err := validateRateLimit(policy.RateLimit); err != nil {
			return err
		}
	}

	if policy.Bulkhead != nil {
		if err := validateBulkhead(policy.Bulkhead); err != nil {
			return err
		}
	}

	return nil
}

func validateCircuitBreaker(cfg *domain.CircuitBreakerConfig) error {
	if cfg.FailureThreshold <= 0 {
		return domain.NewInvalidPolicyError("circuit_breaker.failure_threshold must be positive")
	}
	if cfg.SuccessThreshold <= 0 {
		return domain.NewInvalidPolicyError("circuit_breaker.success_threshold must be positive")
	}
	if cfg.Timeout <= 0 {
		return domain.NewInvalidPolicyError("circuit_breaker.timeout must be positive")
	}
	return nil
}

func validateRetry(cfg *domain.RetryConfig) error {
	if cfg.MaxAttempts <= 0 {
		return domain.NewInvalidPolicyError("retry.max_attempts must be positive")
	}
	if cfg.BaseDelay <= 0 {
		return domain.NewInvalidPolicyError("retry.base_delay must be positive")
	}
	if cfg.MaxDelay <= 0 {
		return domain.NewInvalidPolicyError("retry.max_delay must be positive")
	}
	if cfg.Multiplier < 1.0 {
		return domain.NewInvalidPolicyError("retry.multiplier must be at least 1.0")
	}
	if cfg.JitterPercent < 0 || cfg.JitterPercent > 1.0 {
		return domain.NewInvalidPolicyError("retry.jitter_percent must be between 0 and 1")
	}
	return nil
}

func validateTimeout(cfg *domain.TimeoutConfig) error {
	if cfg.Default <= 0 {
		return domain.NewInvalidPolicyError("timeout.default must be positive")
	}
	return nil
}

func validateRateLimit(cfg *domain.RateLimitConfig) error {
	if cfg.Limit <= 0 {
		return domain.NewInvalidPolicyError("rate_limit.limit must be positive")
	}
	if cfg.Window <= 0 {
		return domain.NewInvalidPolicyError("rate_limit.window must be positive")
	}
	return nil
}

func validateBulkhead(cfg *domain.BulkheadConfig) error {
	if cfg.MaxConcurrent <= 0 {
		return domain.NewInvalidPolicyError("bulkhead.max_concurrent must be positive")
	}
	if cfg.MaxQueue < 0 {
		return domain.NewInvalidPolicyError("bulkhead.max_queue must be non-negative")
	}
	return nil
}

// StartHotReload starts watching for configuration changes.
func (e *Engine) StartHotReload(ctx context.Context) error {
	if e.configPath == "" {
		return nil
	}

	// Initial load
	if err := e.loadFromFile(); err != nil {
		return fmt.Errorf("initial policy load: %w", err)
	}

	// Start watching
	go e.watchLoop(ctx)

	return nil
}

// Stop stops the hot-reload watcher.
func (e *Engine) Stop() {
	close(e.stopCh)
}

func (e *Engine) watchLoop(ctx context.Context) {
	ticker := time.NewTicker(e.reloadInterval)
	defer ticker.Stop()

	var lastModTime time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			info, err := os.Stat(e.configPath)
			if err != nil {
				continue
			}

			if info.ModTime().After(lastModTime) {
				if err := e.loadFromFile(); err == nil {
					lastModTime = info.ModTime()
				}
			}
		}
	}
}

func (e *Engine) loadFromFile() error {
	data, err := os.ReadFile(e.configPath)
	if err != nil {
		return err
	}

	var policies []*domain.ResiliencePolicy

	ext := filepath.Ext(e.configPath)
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &policies); err != nil {
			return err
		}
	case ".json":
		if err := json.Unmarshal(data, &policies); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported config format: %s", ext)
	}

	for _, p := range policies {
		if err := e.UpdatePolicy(p); err != nil {
			return err
		}
	}

	return nil
}
