// Package server contains unit tests for server components.
package server

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/server"
)

func TestShutdownManager_RegisterHook(t *testing.T) {
	cfg := server.ShutdownConfig{Timeout: 5 * time.Second}
	mgr := server.NewShutdownManager(cfg)

	called := false
	mgr.RegisterHook(server.ShutdownHook{
		Name:     "test",
		Priority: 10,
		Fn: func(ctx context.Context) error {
			called = true
			return nil
		},
	})

	err := mgr.Shutdown(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !called {
		t.Error("hook should have been called")
	}
}

func TestShutdownManager_HookPriority(t *testing.T) {
	cfg := server.ShutdownConfig{Timeout: 5 * time.Second}
	mgr := server.NewShutdownManager(cfg)

	var order []string

	mgr.RegisterHook(server.ShutdownHook{
		Name:     "low",
		Priority: 10,
		Fn: func(ctx context.Context) error {
			order = append(order, "low")
			return nil
		},
	})

	mgr.RegisterHook(server.ShutdownHook{
		Name:     "high",
		Priority: 100,
		Fn: func(ctx context.Context) error {
			order = append(order, "high")
			return nil
		},
	})

	mgr.RegisterHook(server.ShutdownHook{
		Name:     "medium",
		Priority: 50,
		Fn: func(ctx context.Context) error {
			order = append(order, "medium")
			return nil
		},
	})

	_ = mgr.Shutdown(context.Background())

	if len(order) != 3 {
		t.Fatalf("expected 3 hooks, got %d", len(order))
	}

	// Higher priority should execute first
	if order[0] != "high" {
		t.Errorf("expected high first, got %s", order[0])
	}
	if order[1] != "medium" {
		t.Errorf("expected medium second, got %s", order[1])
	}
	if order[2] != "low" {
		t.Errorf("expected low third, got %s", order[2])
	}
}

func TestShutdownManager_HookError(t *testing.T) {
	cfg := server.ShutdownConfig{Timeout: 5 * time.Second}
	mgr := server.NewShutdownManager(cfg)

	expectedErr := errors.New("hook error")
	mgr.RegisterHook(server.ShutdownHook{
		Name:     "failing",
		Priority: 10,
		Fn: func(ctx context.Context) error {
			return expectedErr
		},
	})

	err := mgr.Shutdown(context.Background())
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestShutdownManager_IsShuttingDown(t *testing.T) {
	cfg := server.ShutdownConfig{Timeout: 5 * time.Second}
	mgr := server.NewShutdownManager(cfg)

	if mgr.IsShuttingDown() {
		t.Error("should not be shutting down initially")
	}

	_ = mgr.Shutdown(context.Background())

	if !mgr.IsShuttingDown() {
		t.Error("should be shutting down after Shutdown()")
	}
}

func TestShutdownManager_DoubleShutdown(t *testing.T) {
	cfg := server.ShutdownConfig{Timeout: 5 * time.Second}
	mgr := server.NewShutdownManager(cfg)

	var callCount atomic.Int32
	mgr.RegisterHook(server.ShutdownHook{
		Name:     "counter",
		Priority: 10,
		Fn: func(ctx context.Context) error {
			callCount.Add(1)
			return nil
		},
	})

	_ = mgr.Shutdown(context.Background())
	_ = mgr.Shutdown(context.Background())

	if callCount.Load() != 1 {
		t.Errorf("hook should only be called once, got %d", callCount.Load())
	}
}

func TestShutdownManager_Timeout(t *testing.T) {
	cfg := server.ShutdownConfig{Timeout: 100 * time.Millisecond}
	mgr := server.NewShutdownManager(cfg)

	mgr.RegisterHook(server.ShutdownHook{
		Name:     "slow",
		Priority: 10,
		Fn: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				return nil
			}
		},
	})

	start := time.Now()
	err := mgr.Shutdown(context.Background())
	duration := time.Since(start)

	if err == nil {
		t.Error("expected timeout error")
	}

	if duration > 500*time.Millisecond {
		t.Errorf("shutdown took too long: %v", duration)
	}
}

func TestNewHealthCheckHook(t *testing.T) {
	called := false
	hook := server.NewHealthCheckHook(func() {
		called = true
	})

	if hook.Name != "health-check" {
		t.Errorf("expected name 'health-check', got %s", hook.Name)
	}

	if hook.Priority != server.PriorityHealthCheck {
		t.Errorf("expected priority %d, got %d", server.PriorityHealthCheck, hook.Priority)
	}

	_ = hook.Fn(context.Background())
	if !called {
		t.Error("hook function should have been called")
	}
}

func TestNewFlushLogsHook(t *testing.T) {
	called := false
	hook := server.NewFlushLogsHook(func() error {
		called = true
		return nil
	})

	if hook.Name != "flush-logs" {
		t.Errorf("expected name 'flush-logs', got %s", hook.Name)
	}

	_ = hook.Fn(context.Background())
	if !called {
		t.Error("hook function should have been called")
	}
}
