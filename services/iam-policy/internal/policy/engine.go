package policy

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/open-policy-agent/opa/rego"
)

type Engine struct {
	mu       sync.RWMutex
	queries  map[string]*rego.PreparedEvalQuery
	policies map[string]string
}

func NewEngine(policyPath string) (*Engine, error) {
	e := &Engine{
		queries:  make(map[string]*rego.PreparedEvalQuery),
		policies: make(map[string]string),
	}

	if err := e.loadPolicies(policyPath); err != nil {
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

		// Prepare query for authorization
		query, err := rego.New(
			rego.Query("data.authz.allow"),
			rego.Module(name, string(content)),
		).PrepareForEval(context.Background())

		if err != nil {
			log.Printf("Warning: failed to prepare policy %s: %v", name, err)
			continue
		}

		e.queries[name] = &query
	}

	log.Printf("Loaded %d policies", len(e.policies))
	return nil
}

func (e *Engine) Evaluate(ctx context.Context, input map[string]interface{}) (bool, string, []string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for name, query := range e.queries {
		results, err := query.Eval(ctx, rego.EvalInput(input))
		if err != nil {
			continue
		}

		if len(results) > 0 && len(results[0].Expressions) > 0 {
			if allowed, ok := results[0].Expressions[0].Value.(bool); ok && allowed {
				return true, name, []string{name}, nil
			}
		}
	}

	return false, "", nil, nil
}

func (e *Engine) WatchPolicies(ctx context.Context, policyPath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Failed to create watcher: %v", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(policyPath); err != nil {
		log.Printf("Failed to watch policy path: %v", err)
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
				log.Printf("Policy change detected: %s", event.Name)
				if err := e.loadPolicies(policyPath); err != nil {
					log.Printf("Failed to reload policies: %v", err)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (e *Engine) GetPolicyCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.policies)
}
