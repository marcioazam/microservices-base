// Package policy implements resilience policy management.
package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"gopkg.in/yaml.v3"
)

// Engine implements the PolicyEngine interface.
type Engine struct {
	mu             sync.RWMutex
	policies       map[string]*resilience.ResiliencePolicy
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
	// Clean and normalize config path
	configPath := filepath.Clean(cfg.ConfigPath)

	return &Engine{
		policies:       make(map[string]*resilience.ResiliencePolicy),
		configPath:     configPath,
		reloadInterval: cfg.ReloadInterval,
		stopCh:         make(chan struct{}),
		eventCh:        make(chan domain.PolicyEvent, 100),
	}
}

// GetPolicy retrieves policy by name.
func (e *Engine) GetPolicy(name string) (*resilience.ResiliencePolicy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policy, ok := e.policies[name]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", name)
	}

	return policy, nil
}

// UpdatePolicy updates or creates a policy.
func (e *Engine) UpdatePolicy(policy *resilience.ResiliencePolicy) error {
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
	case e.eventCh <- domain.PolicyEvent{Type: eventType, PolicyName: policy.Name, Version: policy.Version, Timestamp: resilience.NowUTC()}:
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
	case e.eventCh <- domain.PolicyEvent{Type: domain.PolicyDeleted, PolicyName: policy.Name, Version: policy.Version, Timestamp: resilience.NowUTC()}:
	default:
	}

	return nil
}

// ListPolicies returns all policies.
func (e *Engine) ListPolicies() ([]*resilience.ResiliencePolicy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*resilience.ResiliencePolicy, 0, len(e.policies))
	for _, p := range e.policies {
		result = append(result, p)
	}

	return result, nil
}

// Policies returns an iterator over all policies.
func (e *Engine) Policies() iter.Seq[*resilience.ResiliencePolicy] {
	return func(yield func(*resilience.ResiliencePolicy) bool) {
		e.mu.RLock()
		defer e.mu.RUnlock()
		for _, p := range e.policies {
			if !yield(p) {
				return
			}
		}
	}
}

// WatchPolicies streams policy changes.
func (e *Engine) WatchPolicies(ctx context.Context) (<-chan domain.PolicyEvent, error) {
	return e.eventCh, nil
}

// Validate validates a policy configuration.
// Delegates to resilience.ResiliencePolicy.Validate() for validation logic.
func (e *Engine) Validate(policy *resilience.ResiliencePolicy) error {
	return policy.Validate()
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
	// Validate path to prevent path traversal attacks
	if err := e.validatePolicyPath(e.configPath); err != nil {
		return fmt.Errorf("invalid policy path: %w", err)
	}

	data, err := os.ReadFile(e.configPath)
	if err != nil {
		return err
	}

	var policies []*resilience.ResiliencePolicy

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

// validatePolicyPath validates that a file path is safe to access.
// It prevents path traversal attacks by ensuring the path stays within
// the expected policy configuration directory.
func (e *Engine) validatePolicyPath(path string) error {
	return ValidatePolicyPath(path, filepath.Dir(e.configPath))
}

// ValidatePolicyPath validates that a file path is safe to access.
// It prevents path traversal attacks by ensuring the path stays within
// the specified base directory.
func ValidatePolicyPath(path, basePath string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	// Check for null bytes (common attack vector)
	if strings.ContainsRune(path, '\x00') {
		return fmt.Errorf("path contains null bytes")
	}

	// Reject paths with parent directory references BEFORE cleaning
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains parent directory reference")
	}

	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(path)

	// If path is absolute, check it's within base
	if filepath.IsAbs(path) {
		absBase, err := filepath.Abs(basePath)
		if err != nil {
			return fmt.Errorf("resolve absolute base: %w", err)
		}
		if !strings.HasPrefix(cleanPath, absBase) {
			return fmt.Errorf("absolute path '%s' is outside allowed directory '%s'", path, basePath)
		}
		return nil
	}

	// For relative paths, resolve and check
	baseDir := basePath
	if baseDir == "." || baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
	}
	baseDir = filepath.Clean(baseDir)

	// Resolve to absolute paths for comparison
	absPath, err := filepath.Abs(filepath.Join(baseDir, cleanPath))
	if err != nil {
		return fmt.Errorf("resolve absolute path: %w", err)
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("resolve absolute base: %w", err)
	}

	// Ensure the path is within the base directory
	if !strings.HasPrefix(absPath, absBase) {
		return fmt.Errorf("path '%s' is outside allowed directory '%s'", path, baseDir)
	}

	return nil
}
