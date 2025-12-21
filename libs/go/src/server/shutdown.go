package server

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ShutdownHandler manages graceful shutdown.
type ShutdownHandler struct {
	hooks   []ShutdownHook
	mu      sync.Mutex
	timeout time.Duration
	signals []os.Signal
}

// ShutdownHook is called during shutdown.
type ShutdownHook func(ctx context.Context) error

// NewShutdownHandler creates a new shutdown handler.
func NewShutdownHandler() *ShutdownHandler {
	return &ShutdownHandler{
		timeout: time.Second * 30,
		signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
	}
}

// WithTimeout sets the shutdown timeout.
func (s *ShutdownHandler) WithTimeout(d time.Duration) *ShutdownHandler {
	s.timeout = d
	return s
}

// WithSignals sets the signals to listen for.
func (s *ShutdownHandler) WithSignals(signals ...os.Signal) *ShutdownHandler {
	s.signals = signals
	return s
}

// OnShutdown registers a shutdown hook.
func (s *ShutdownHandler) OnShutdown(hook ShutdownHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hooks = append(s.hooks, hook)
}

// Wait blocks until shutdown signal is received.
func (s *ShutdownHandler) Wait() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, s.signals...)
	<-sigChan
}

// Shutdown executes all shutdown hooks.
func (s *ShutdownHandler) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	s.mu.Lock()
	hooks := make([]ShutdownHook, len(s.hooks))
	copy(hooks, s.hooks)
	s.mu.Unlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(hooks))

	// Execute hooks in reverse order (LIFO)
	for i := len(hooks) - 1; i >= 0; i-- {
		wg.Add(1)
		go func(hook ShutdownHook) {
			defer wg.Done()
			if err := hook(ctx); err != nil {
				errChan <- err
			}
		}(hooks[i])
	}

	wg.Wait()
	close(errChan)

	// Return first error if any
	for err := range errChan {
		return err
	}

	return nil
}

// WaitAndShutdown waits for signal then shuts down.
func (s *ShutdownHandler) WaitAndShutdown() error {
	s.Wait()
	return s.Shutdown(context.Background())
}

// GracefulServer wraps a server with graceful shutdown.
type GracefulServer struct {
	start    func() error
	stop     func(context.Context) error
	shutdown *ShutdownHandler
}

// NewGracefulServer creates a new graceful server.
func NewGracefulServer(start func() error, stop func(context.Context) error) *GracefulServer {
	return &GracefulServer{
		start:    start,
		stop:     stop,
		shutdown: NewShutdownHandler(),
	}
}

// OnShutdown registers a shutdown hook.
func (g *GracefulServer) OnShutdown(hook ShutdownHook) {
	g.shutdown.OnShutdown(hook)
}

// Run starts the server and handles graceful shutdown.
func (g *GracefulServer) Run() error {
	errChan := make(chan error, 1)

	go func() {
		errChan <- g.start()
	}()

	g.shutdown.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), g.shutdown.timeout)
	defer cancel()

	if err := g.stop(ctx); err != nil {
		return err
	}

	if err := g.shutdown.Shutdown(ctx); err != nil {
		return err
	}

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

// DrainManager manages graceful shutdown with request draining.
type DrainManager struct {
	mu           sync.RWMutex
	inFlight     int64
	shuttingDown int32
	done         chan struct{}
}

// NewDrainManager creates a new drain manager.
func NewDrainManager() *DrainManager {
	return &DrainManager{
		done: make(chan struct{}),
	}
}

// RequestStarted should be called when a request starts.
func (m *DrainManager) RequestStarted() bool {
	if m.shuttingDown == 1 {
		return false
	}
	m.mu.Lock()
	m.inFlight++
	m.mu.Unlock()
	return true
}

// RequestFinished should be called when a request finishes.
func (m *DrainManager) RequestFinished() {
	m.mu.Lock()
	m.inFlight--
	if m.inFlight == 0 && m.shuttingDown == 1 {
		select {
		case <-m.done:
		default:
			close(m.done)
		}
	}
	m.mu.Unlock()
}

// Drain initiates graceful shutdown and waits for in-flight requests.
func (m *DrainManager) Drain(ctx context.Context) error {
	m.mu.Lock()
	m.shuttingDown = 1
	if m.inFlight == 0 {
		select {
		case <-m.done:
		default:
			close(m.done)
		}
	}
	m.mu.Unlock()

	select {
	case <-m.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsShuttingDown returns true if shutdown has been initiated.
func (m *DrainManager) IsShuttingDown() bool {
	return m.shuttingDown == 1
}

// InFlightCount returns the number of in-flight requests.
func (m *DrainManager) InFlightCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.inFlight
}

// Done returns a channel that is closed when shutdown is complete.
func (m *DrainManager) Done() <-chan struct{} {
	return m.done
}
