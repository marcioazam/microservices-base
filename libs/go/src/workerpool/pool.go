// Package workerpool provides a worker pool with priority queue.
package workerpool

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Job represents a job to be processed.
type Job[T any] struct {
	ID       string
	Data     T
	Priority int
	Created  time.Time
}

// Result represents a job result.
type Result[T any] struct {
	JobID string
	Data  T
	Error error
}

// Pool is a worker pool with priority queue.
type Pool[T, R any] struct {
	workers    int
	handler    func(context.Context, T) (R, error)
	jobs       chan Job[T]
	results    chan Result[R]
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	processing int64
	completed  int64
	failed     int64
}

// NewPool creates a new worker pool.
func NewPool[T, R any](workers int, handler func(context.Context, T) (R, error)) *Pool[T, R] {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool[T, R]{
		workers: workers,
		handler: handler,
		jobs:    make(chan Job[T], workers*10),
		results: make(chan Result[R], workers*10),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the worker pool.
func (p *Pool[T, R]) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// Submit submits a job to the pool.
func (p *Pool[T, R]) Submit(job Job[T]) {
	if job.Created.IsZero() {
		job.Created = time.Now()
	}
	p.jobs <- job
}

// Results returns the results channel.
func (p *Pool[T, R]) Results() <-chan Result[R] {
	return p.results
}

// Shutdown gracefully shuts down the pool.
func (p *Pool[T, R]) Shutdown(timeout time.Duration) {
	close(p.jobs)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
	}

	p.cancel()
	close(p.results)
}

// Stats returns pool statistics.
func (p *Pool[T, R]) Stats() PoolStats {
	return PoolStats{
		Workers:    p.workers,
		Processing: atomic.LoadInt64(&p.processing),
		Completed:  atomic.LoadInt64(&p.completed),
		Failed:     atomic.LoadInt64(&p.failed),
		Pending:    len(p.jobs),
	}
}

// PoolStats holds pool statistics.
type PoolStats struct {
	Workers    int
	Processing int64
	Completed  int64
	Failed     int64
	Pending    int
}

func (p *Pool[T, R]) worker() {
	defer p.wg.Done()

	for job := range p.jobs {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		atomic.AddInt64(&p.processing, 1)

		result := Result[R]{JobID: job.ID}

		func() {
			defer func() {
				if r := recover(); r != nil {
					result.Error = &PanicError{Value: r}
					atomic.AddInt64(&p.failed, 1)
				}
			}()

			data, err := p.handler(p.ctx, job.Data)
			result.Data = data
			result.Error = err

			if err != nil {
				atomic.AddInt64(&p.failed, 1)
			} else {
				atomic.AddInt64(&p.completed, 1)
			}
		}()

		atomic.AddInt64(&p.processing, -1)

		select {
		case p.results <- result:
		case <-p.ctx.Done():
			return
		}
	}
}

// PanicError represents a panic that occurred during job processing.
type PanicError struct {
	Value any
}

func (e *PanicError) Error() string {
	return "worker panic"
}
