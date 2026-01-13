// Package server provides server lifecycle management for IAM Policy Service.
package server

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/logging"
)

// ShutdownManager manages graceful shutdown.
type ShutdownManager struct {
	mu           sync.Mutex
	hooks        []ShutdownHook
	logger       *logging.Logger
	timeout      time.Duration
	shuttingDown bool
}

// ShutdownHook is a function called during shutdown.
type ShutdownHook struct {
	Name     string
	Priority int
	Fn       func(context.Context) error
}

// ShutdownConfig holds configuration for shutdown manager.
type ShutdownConfig struct {
	Timeout time.Duration
	Logger  *logging.Logger
}

// NewShutdownManager creates a new shutdown manager.
func NewShutdownManager(cfg ShutdownConfig) *ShutdownManager {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &ShutdownManager{
		hooks:   make([]ShutdownHook, 0),
		logger:  cfg.Logger,
		timeout: timeout,
	}
}

// RegisterHook registers a shutdown hook.
func (m *ShutdownManager) RegisterHook(hook ShutdownHook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks = append(m.hooks, hook)
}

// IsShuttingDown returns whether shutdown is in progress.
func (m *ShutdownManager) IsShuttingDown() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.shuttingDown
}

// WaitForSignal waits for shutdown signals and initiates shutdown.
func (m *ShutdownManager) WaitForSignal(ctx context.Context) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigChan:
		if m.logger != nil {
			m.logger.Info(ctx, "received shutdown signal", logging.String("signal", sig.String()))
		}
		return m.Shutdown(ctx)
	case <-ctx.Done():
		return m.Shutdown(ctx)
	}
}

// Shutdown initiates graceful shutdown.
func (m *ShutdownManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	if m.shuttingDown {
		m.mu.Unlock()
		return nil
	}
	m.shuttingDown = true
	m.mu.Unlock()

	if m.logger != nil {
		m.logger.Info(ctx, "starting graceful shutdown", logging.Duration("timeout", m.timeout))
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// Sort hooks by priority (higher priority first)
	m.mu.Lock()
	hooks := make([]ShutdownHook, len(m.hooks))
	copy(hooks, m.hooks)
	m.mu.Unlock()

	sortHooksByPriority(hooks)

	// Execute hooks
	var lastErr error
	for _, hook := range hooks {
		if m.logger != nil {
			m.logger.Debug(shutdownCtx, "executing shutdown hook", logging.String("name", hook.Name))
		}

		if err := hook.Fn(shutdownCtx); err != nil {
			lastErr = err
			if m.logger != nil {
				m.logger.Error(shutdownCtx, "shutdown hook failed",
					logging.String("name", hook.Name), logging.Error(err))
			}
		}
	}

	if m.logger != nil {
		m.logger.Info(ctx, "graceful shutdown complete")
	}

	return lastErr
}

func sortHooksByPriority(hooks []ShutdownHook) {
	for i := 0; i < len(hooks)-1; i++ {
		for j := i + 1; j < len(hooks); j++ {
			if hooks[j].Priority > hooks[i].Priority {
				hooks[i], hooks[j] = hooks[j], hooks[i]
			}
		}
	}
}

// Common shutdown hook priorities.
const (
	PriorityHealthCheck = 100 // Mark unhealthy first
	PriorityGRPCServer  = 90  // Stop accepting new requests
	PriorityHTTPServer  = 90  // Stop accepting new requests
	PriorityInFlight    = 80  // Wait for in-flight requests
	PriorityFlushLogs   = 70  // Flush log buffers
	PriorityCloseCache  = 60  // Close cache connections
	PriorityCloseDB     = 50  // Close database connections
)

// NewHealthCheckHook creates a hook to mark service unhealthy.
func NewHealthCheckHook(markUnhealthy func()) ShutdownHook {
	return ShutdownHook{
		Name:     "health-check",
		Priority: PriorityHealthCheck,
		Fn: func(ctx context.Context) error {
			markUnhealthy()
			return nil
		},
	}
}

// NewGRPCServerHook creates a hook to stop gRPC server.
func NewGRPCServerHook(stop func()) ShutdownHook {
	return ShutdownHook{
		Name:     "grpc-server",
		Priority: PriorityGRPCServer,
		Fn: func(ctx context.Context) error {
			stop()
			return nil
		},
	}
}

// NewFlushLogsHook creates a hook to flush log buffers.
func NewFlushLogsHook(flush func() error) ShutdownHook {
	return ShutdownHook{
		Name:     "flush-logs",
		Priority: PriorityFlushLogs,
		Fn: func(ctx context.Context) error {
			return flush()
		},
	}
}

// NewCloseCacheHook creates a hook to close cache connections.
func NewCloseCacheHook(close func() error) ShutdownHook {
	return ShutdownHook{
		Name:     "close-cache",
		Priority: PriorityCloseCache,
		Fn: func(ctx context.Context) error {
			return close()
		},
	}
}
