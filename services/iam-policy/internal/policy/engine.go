// Package policy provides OPA-based policy evaluation for IAM Policy Service.
package policy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/auth-platform/iam-policy-service/internal/cache"
	"github.com/auth-platform/iam-policy-service/internal/logging"
	"github.com/fsnotify/fsnotify"
	"github.com/open-policy-agent/opa/rego"
)

// EvaluationResult holds the result of a policy evaluation.
type EvaluationResult struct {
	Allowed     bool
	PolicyName  string
	Policies    []string
	FromCache   bool
	EvalTimeNs  int64
}

// Engine evaluates OPA policies with caching support.
type Engine struct {
	mu          sync.RWMutex
	queries     map[string]*rego.PreparedEvalQuery
	policies    map[string]string
	cache       CacheInterface
	logger      *logging.Logger
	policyPath  string
	evalCount   atomic.Int64
	cacheHits   atomic.Int64
	cacheMisses atomic.Int64
}

// EngineConfig holds configuration for the policy engine.
type EngineConfig struct {
	PolicyPath string
	Cache      CacheInterface
	Logger     *logging.Logger
}

// CacheInterface defines the interface for decision caching.
type CacheInterface interface {
	Get(ctx context.Context, input map[string]interface{}) (*cache.Decision, bool)
	Set(ctx context.Context, input map[string]interface{}, decision *cache.Decision) error
	Invalidate(ctx context.Context) error
}

// NewEngine creates a new policy engine.
func NewEngine(cfg EngineConfig) (*Engine, error) {
	e := &Engine{
		queries:    make(map[string]*rego.PreparedEvalQuery),
		policies:   make(map[string]string),
		cache:      cfg.Cache,
		logger:     cfg.Logger,
		policyPath: cfg.PolicyPath,
	}

	if err := e.loadPolicies(cfg.PolicyPath); err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Engine) loadPolicies(policyPath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(policyPath, "*.rego"))
	if err != nil {
		return fmt.Errorf("failed to glob policies: %w", err)
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read policy %s: %w", file, err)
		}

		name := filepath.Base(file)
		e.policies[name] = string(content)

		query, err := rego.New(
			rego.Query("data.authz.allow"),
			rego.Module(name, string(content)),
		).PrepareForEval(context.Background())

		if err != nil {
			if e.logger != nil {
				e.logger.Warn(context.Background(), "failed to prepare policy",
					logging.String("policy", name), logging.Error(err))
			}
			continue
		}

		e.queries[name] = &query
	}

	if e.logger != nil {
		e.logger.Info(context.Background(), "policies loaded",
			logging.Int("count", len(e.policies)))
	}
	return nil
}

// Evaluate evaluates policies against input, using cache when available.
func (e *Engine) Evaluate(ctx context.Context, input map[string]interface{}) (*EvaluationResult, error) {
	e.evalCount.Add(1)

	// Check cache first
	if e.cache != nil {
		if decision, found := e.cache.Get(ctx, input); found {
			e.cacheHits.Add(1)
			return &EvaluationResult{
				Allowed:   decision.Allowed,
				FromCache: true,
			}, nil
		}
		e.cacheMisses.Add(1)
	}

	// Evaluate policies
	result, err := e.evaluatePolicies(ctx, input)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if e.cache != nil {
		decision := &cache.Decision{Allowed: result.Allowed, Reason: result.PolicyName}
		_ = e.cache.Set(ctx, input, decision)
	}

	return result, nil
}

func (e *Engine) evaluatePolicies(ctx context.Context, input map[string]interface{}) (*EvaluationResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for name, query := range e.queries {
		results, err := query.Eval(ctx, rego.EvalInput(input))
		if err != nil {
			continue
		}

		if len(results) > 0 && len(results[0].Expressions) > 0 {
			if allowed, ok := results[0].Expressions[0].Value.(bool); ok && allowed {
				return &EvaluationResult{
					Allowed:    true,
					PolicyName: name,
					Policies:   []string{name},
				}, nil
			}
		}
	}

	return &EvaluationResult{Allowed: false}, nil
}

// ReloadPolicies reloads policies and invalidates cache.
func (e *Engine) ReloadPolicies(ctx context.Context) error {
	if err := e.loadPolicies(e.policyPath); err != nil {
		return err
	}

	if e.cache != nil {
		return e.cache.Invalidate(ctx)
	}
	return nil
}

// WatchPolicies watches for policy file changes.
func (e *Engine) WatchPolicies(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		if e.logger != nil {
			e.logger.Error(ctx, "failed to create watcher", logging.Error(err))
		}
		return
	}
	defer watcher.Close()

	if err := watcher.Add(e.policyPath); err != nil {
		if e.logger != nil {
			e.logger.Error(ctx, "failed to watch policy path", logging.Error(err))
		}
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if e.logger != nil {
					e.logger.Info(ctx, "policy change detected", logging.String("file", event.Name))
				}
				if err := e.ReloadPolicies(ctx); err != nil {
					if e.logger != nil {
						e.logger.Error(ctx, "failed to reload policies", logging.Error(err))
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			if e.logger != nil {
				e.logger.Error(ctx, "watcher error", logging.Error(err))
			}
		}
	}
}

// GetPolicyCount returns the number of loaded policies.
func (e *Engine) GetPolicyCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.policies)
}

// Stats returns engine statistics.
func (e *Engine) Stats() EngineStats {
	return EngineStats{
		EvalCount:   e.evalCount.Load(),
		CacheHits:   e.cacheHits.Load(),
		CacheMisses: e.cacheMisses.Load(),
		PolicyCount: e.GetPolicyCount(),
	}
}

// EngineStats holds engine statistics.
type EngineStats struct {
	EvalCount   int64
	CacheHits   int64
	CacheMisses int64
	PolicyCount int
}
