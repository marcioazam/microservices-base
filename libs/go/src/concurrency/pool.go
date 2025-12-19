package concurrency

import (
	"context"
	"sync"

	"github.com/authcorp/libs/go/src/functional"
)

// WorkerPool manages a pool of workers for parallel task execution.
type WorkerPool[T, R any] struct {
	workers    int
	tasks      chan T
	results    chan functional.Result[R]
	processor  func(T) (R, error)
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	started    bool
	mu         sync.Mutex
}

// NewWorkerPool creates a new worker pool.
func NewWorkerPool[T, R any](workers int, processor func(T) (R, error)) *WorkerPool[T, R] {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool[T, R]{
		workers:   workers,
		tasks:     make(chan T, workers*2),
		results:   make(chan functional.Result[R], workers*2),
		processor: processor,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins processing tasks.
func (p *WorkerPool[T, R]) Start() {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return
	}
	p.started = true
	p.mu.Unlock()

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *WorkerPool[T, R]) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			result, err := p.processor(task)
			if err != nil {
				p.results <- functional.Err[R](err)
			} else {
				p.results <- functional.Ok(result)
			}
		}
	}
}

// Submit adds a task to the pool.
func (p *WorkerPool[T, R]) Submit(task T) bool {
	select {
	case <-p.ctx.Done():
		return false
	case p.tasks <- task:
		return true
	}
}

// SubmitContext adds a task with context.
func (p *WorkerPool[T, R]) SubmitContext(ctx context.Context, task T) bool {
	select {
	case <-ctx.Done():
		return false
	case <-p.ctx.Done():
		return false
	case p.tasks <- task:
		return true
	}
}

// Results returns the results channel.
func (p *WorkerPool[T, R]) Results() <-chan functional.Result[R] {
	return p.results
}

// Stop gracefully stops the pool.
func (p *WorkerPool[T, R]) Stop() {
	p.cancel()
	close(p.tasks)
	p.wg.Wait()
	close(p.results)
}

// StopAndWait stops and waits for all workers.
func (p *WorkerPool[T, R]) StopAndWait() {
	close(p.tasks)
	p.wg.Wait()
	p.cancel()
	close(p.results)
}

// ErrGroup manages a group of goroutines with error handling.
type ErrGroup struct {
	wg      sync.WaitGroup
	errOnce sync.Once
	err     error
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewErrGroup creates a new error group.
func NewErrGroup(ctx context.Context) *ErrGroup {
	ctx, cancel := context.WithCancel(ctx)
	return &ErrGroup{ctx: ctx, cancel: cancel}
}

// Go runs a function in a goroutine.
func (g *ErrGroup) Go(fn func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := fn(); err != nil {
			g.errOnce.Do(func() {
				g.err = err
				g.cancel()
			})
		}
	}()
}

// Wait waits for all goroutines and returns first error.
func (g *ErrGroup) Wait() error {
	g.wg.Wait()
	return g.err
}

// Context returns the group's context.
func (g *ErrGroup) Context() context.Context {
	return g.ctx
}
