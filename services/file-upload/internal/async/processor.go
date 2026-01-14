// Package async provides async task processing for file upload service.
package async

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// TaskType represents the type of async task.
type TaskType string

const (
	TaskVirusScan       TaskType = "virus_scan"
	TaskThumbnail       TaskType = "thumbnail"
	TaskMetadataExtract TaskType = "metadata_extract"
)

// Task represents an async task.
type Task struct {
	ID        string          `json:"id"`
	Type      TaskType        `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Priority  int             `json:"priority"`
	Retries   int             `json:"retries"`
	MaxRetry  int             `json:"max_retry"`
	CreatedAt time.Time       `json:"created_at"`
	Error     string          `json:"error,omitempty"`
}

// TaskHandler processes a specific task type.
type TaskHandler func(ctx context.Context, task *Task) error

// Config holds processor configuration.
type Config struct {
	Workers       int
	QueueSize     int
	MaxRetries    int
	RetryBaseDelay time.Duration
}

// Processor handles async task processing.
type Processor struct {
	handlers map[TaskType]TaskHandler
	queue    chan *Task
	workers  int
	maxRetry int
	baseDelay time.Duration

	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	mu      sync.RWMutex
}

// NewProcessor creates a new async processor.
func NewProcessor(cfg Config) *Processor {
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBaseDelay <= 0 {
		cfg.RetryBaseDelay = time.Second
	}

	return &Processor{
		handlers:  make(map[TaskType]TaskHandler),
		queue:     make(chan *Task, cfg.QueueSize),
		workers:   cfg.Workers,
		maxRetry:  cfg.MaxRetries,
		baseDelay: cfg.RetryBaseDelay,
	}
}

// RegisterHandler registers a handler for a task type.
func (p *Processor) RegisterHandler(taskType TaskType, handler TaskHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[taskType] = handler
}

// Submit submits a task for processing.
func (p *Processor) Submit(ctx context.Context, task *Task) error {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	if !running {
		return ErrProcessorNotRunning
	}

	task.MaxRetry = p.maxRetry
	task.CreatedAt = time.Now().UTC()

	select {
	case p.queue <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrQueueFull
	}
}

// Start starts the processor workers.
func (p *Processor) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return ErrProcessorAlreadyRunning
	}

	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.running = true

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	return nil
}

// Stop stops the processor gracefully.
func (p *Processor) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.cancel()
	p.mu.Unlock()

	p.wg.Wait()
	return nil
}

// QueueDepth returns the current queue depth.
func (p *Processor) QueueDepth() int {
	return len(p.queue)
}

func (p *Processor) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task := <-p.queue:
			p.processTask(task)
		}
	}
}

func (p *Processor) processTask(task *Task) {
	p.mu.RLock()
	handler, exists := p.handlers[task.Type]
	p.mu.RUnlock()

	if !exists {
		return
	}

	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Minute)
	defer cancel()

	err := handler(ctx, task)
	if err != nil {
		task.Error = err.Error()
		task.Retries++

		if task.Retries < task.MaxRetry {
			// Exponential backoff
			delay := p.calculateRetryDelay(task.Retries)
			time.Sleep(delay)

			// Requeue
			select {
			case p.queue <- task:
			default:
				// Queue full, task lost
			}
		}
	}
}

func (p *Processor) calculateRetryDelay(retries int) time.Duration {
	// Exponential backoff: base * 2^retries
	delay := p.baseDelay
	for i := 0; i < retries; i++ {
		delay *= 2
	}
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	return delay
}

// Errors
var (
	ErrProcessorNotRunning     = &ProcessorError{Code: "NOT_RUNNING", Message: "processor not running"}
	ErrProcessorAlreadyRunning = &ProcessorError{Code: "ALREADY_RUNNING", Message: "processor already running"}
	ErrQueueFull               = &ProcessorError{Code: "QUEUE_FULL", Message: "task queue is full"}
)

// ProcessorError represents a processor error.
type ProcessorError struct {
	Code    string
	Message string
}

func (e *ProcessorError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
