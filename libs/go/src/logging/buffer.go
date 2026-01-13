package logging

import (
	"sync"
	"time"
)

// LogEntry represents a structured log entry.
type LogEntry struct {
	Timestamp     time.Time
	Level         Level
	Message       string
	Service       string
	CorrelationID string
	TraceID       string
	SpanID        string
	Fields        map[string]any
}

// logBuffer buffers log entries for batch shipping.
type logBuffer struct {
	mu            sync.Mutex
	entries       []LogEntry
	capacity      int
	flushInterval time.Duration
	flushFn       func([]LogEntry) error
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// newLogBuffer creates a new log buffer.
func newLogBuffer(capacity int, flushInterval time.Duration, flushFn func([]LogEntry) error) *logBuffer {
	b := &logBuffer{
		entries:       make([]LogEntry, 0, capacity),
		capacity:      capacity,
		flushInterval: flushInterval,
		flushFn:       flushFn,
		stopCh:        make(chan struct{}),
	}

	// Start background flush goroutine
	b.wg.Add(1)
	go b.flushLoop()

	return b
}

// Add adds an entry to the buffer.
func (b *logBuffer) Add(entry LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.entries = append(b.entries, entry)

	// Flush if at capacity
	if len(b.entries) >= b.capacity {
		b.flushLocked()
	}
}

// Flush flushes all buffered entries.
func (b *logBuffer) Flush() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flushLocked()
}

func (b *logBuffer) flushLocked() error {
	if len(b.entries) == 0 {
		return nil
	}

	// Copy entries
	entries := make([]LogEntry, len(b.entries))
	copy(entries, b.entries)
	b.entries = b.entries[:0]

	// Flush outside lock
	if b.flushFn != nil {
		return b.flushFn(entries)
	}
	return nil
}

func (b *logBuffer) flushLoop() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.Flush()
		case <-b.stopCh:
			b.Flush()
			return
		}
	}
}

// Close stops the buffer and flushes remaining entries.
func (b *logBuffer) Close() error {
	close(b.stopCh)
	b.wg.Wait()
	return nil
}

// Size returns the number of buffered entries.
func (b *logBuffer) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}
