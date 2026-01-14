// Package server provides HTTP server with graceful shutdown.
package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ShutdownConfig holds shutdown configuration.
type ShutdownConfig struct {
	Timeout time.Duration
	Signals []os.Signal
}

// DefaultShutdownConfig returns default shutdown configuration.
func DefaultShutdownConfig() ShutdownConfig {
	return ShutdownConfig{
		Timeout: 30 * time.Second,
		Signals: []os.Signal{syscall.SIGTERM, syscall.SIGINT},
	}
}

// ShutdownHandler manages graceful shutdown.
type ShutdownHandler struct {
	config   ShutdownConfig
	handlers []func(ctx context.Context) error
	mu       sync.Mutex
}

// NewShutdownHandler creates a new shutdown handler.
func NewShutdownHandler(config ShutdownConfig) *ShutdownHandler {
	return &ShutdownHandler{
		config:   config,
		handlers: make([]func(ctx context.Context) error, 0),
	}
}

// Register registers a cleanup handler.
func (s *ShutdownHandler) Register(handler func(ctx context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, handler)
}

// Shutdown executes all cleanup handlers.
func (s *ShutdownHandler) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	handlers := make([]func(ctx context.Context) error, len(s.handlers))
	copy(handlers, s.handlers)
	s.mu.Unlock()

	var lastErr error
	for _, handler := range handlers {
		if err := handler(ctx); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Server wraps http.Server with graceful shutdown.
type Server struct {
	httpServer *http.Server
	shutdown   *ShutdownHandler
	accepting  bool
	mu         sync.RWMutex
}

// NewServer creates a new server.
func NewServer(addr string, handler http.Handler, shutdownConfig ShutdownConfig) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		shutdown:  NewShutdownHandler(shutdownConfig),
		accepting: true,
	}
}

// RegisterShutdownHandler registers a cleanup handler.
func (s *Server) RegisterShutdownHandler(handler func(ctx context.Context) error) {
	s.shutdown.Register(handler)
}

// IsAccepting returns true if server is accepting new requests.
func (s *Server) IsAccepting() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accepting
}

// ListenAndServe starts the server and handles graceful shutdown.
func (s *Server) ListenAndServe() error {
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, s.shutdown.config.Signals...)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for signal or error
	select {
	case err := <-errChan:
		return err
	case <-sigChan:
		return s.GracefulShutdown()
	}
}

// GracefulShutdown performs graceful shutdown.
func (s *Server) GracefulShutdown() error {
	// Stop accepting new requests
	s.mu.Lock()
	s.accepting = false
	s.mu.Unlock()

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdown.config.Timeout)
	defer cancel()

	// Shutdown HTTP server (waits for in-flight requests)
	if err := s.httpServer.Shutdown(ctx); err != nil {
		// Force close if timeout exceeded
		s.httpServer.Close()
	}

	// Run cleanup handlers
	return s.shutdown.Shutdown(ctx)
}

// RejectingMiddleware returns middleware that rejects requests during shutdown.
func (s *Server) RejectingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.IsAccepting() {
			w.Header().Set("Connection", "close")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"server shutting down"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
