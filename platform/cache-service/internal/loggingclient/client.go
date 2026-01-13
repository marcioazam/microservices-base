package loggingclient

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	libobs "github.com/authcorp/libs/go/src/observability"
)

// Client sends logs to the centralized logging-service via gRPC.
type Client struct {
	conn          *grpc.ClientConn
	client        LoggingServiceClient
	config        Config
	mu            sync.Mutex
	buffer        []*LogEntryMessage
	flushTimer    *time.Timer
	closed        atomic.Bool
	circuitOpen   atomic.Bool
	failures      atomic.Int32
	lastFailure   time.Time
	failureMu     sync.Mutex
	noop          bool // suppresses all output
}

// New creates a new logging client.
func New(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	c := &Client{
		config: cfg,
		buffer: make([]*LogEntryMessage, 0, cfg.BatchSize),
	}

	if cfg.Enabled {
		conn, err := grpc.NewClient(cfg.Address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to logging-service: %w", err)
		}
		c.conn = conn
		c.client = NewLoggingServiceClient(conn)
	}

	c.startFlushTimer()
	return c, nil
}

// NewNoop creates a no-op client for testing (suppresses all output).
func NewNoop() *Client {
	return &Client{
		config: Config{Enabled: false, BatchSize: 100, FlushInterval: 5 * time.Second},
		buffer: make([]*LogEntryMessage, 0, 100),
		noop:   true,
	}
}

func (c *Client) startFlushTimer() {
	c.flushTimer = time.AfterFunc(c.config.FlushInterval, func() {
		c.flush()
		if !c.closed.Load() {
			c.startFlushTimer()
		}
	})
}

// Log sends a log entry (buffered for batching).
func (c *Client) Log(ctx context.Context, level Level, msg string, fields ...Field) {
	if c.closed.Load() {
		return
	}

	entry := c.createEntry(ctx, level, msg, fields)

	c.mu.Lock()
	c.buffer = append(c.buffer, entry)
	shouldFlush := len(c.buffer) >= c.config.BatchSize
	c.mu.Unlock()

	if shouldFlush {
		go c.flush()
	}
}

func (c *Client) createEntry(ctx context.Context, level Level, msg string, fields []Field) *LogEntryMessage {
	entry := &LogEntryMessage{
		Id:        uuid.New().String(),
		Timestamp: timestamppb.Now(),
		ServiceId: c.config.ServiceID,
		Level:     LogLevel(level),
		Message:   msg,
		Metadata:  fieldsToMap(fields),
	}

	// Extract context values using lib observability
	if correlationID := libobs.CorrelationIDFromContext(ctx); correlationID != "" {
		entry.CorrelationId = correlationID
	}
	if traceID, spanID := libobs.TraceContextFromContext(ctx); traceID != "" {
		entry.TraceId = &traceID
		if spanID != "" {
			entry.SpanId = &spanID
		}
	}
	if user, ok := libobs.UserContextFromContext(ctx); ok && user.UserID != "" {
		entry.Metadata["user_id"] = user.UserID
	}

	return entry
}

func (c *Client) flush() {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return
	}
	entries := c.buffer
	c.buffer = make([]*LogEntryMessage, 0, c.config.BatchSize)
	c.mu.Unlock()

	// Noop client discards all logs
	if c.noop {
		return
	}

	if !c.config.Enabled || c.client == nil {
		c.writeToStderr(entries)
		return
	}

	if c.isCircuitOpen() {
		c.writeToStderr(entries)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.client.IngestLogBatch(ctx, &IngestLogBatchRequest{Entries: entries})
	if err != nil {
		c.recordFailure()
		c.writeToStderr(entries)
		return
	}

	c.recordSuccess()
}

func (c *Client) writeToStderr(entries []*LogEntryMessage) {
	for _, e := range entries {
		fmt.Fprintf(os.Stderr, "[%s] %s: %s %v\n",
			e.Timestamp.AsTime().Format(time.RFC3339),
			e.Level.String(),
			e.Message,
			e.Metadata,
		)
	}
}

func (c *Client) isCircuitOpen() bool {
	if !c.circuitOpen.Load() {
		return false
	}

	c.failureMu.Lock()
	defer c.failureMu.Unlock()

	if time.Since(c.lastFailure) >= c.config.CircuitBreakerTimeout {
		c.circuitOpen.Store(false)
		c.failures.Store(0)
		return false
	}
	return true
}

func (c *Client) recordFailure() {
	c.failureMu.Lock()
	defer c.failureMu.Unlock()

	c.lastFailure = time.Now()
	// #nosec G115 - CircuitBreakerThreshold is validated to be small (typically < 100)
	threshold := int32(min(c.config.CircuitBreakerThreshold, 1000))
	if c.failures.Add(1) >= threshold {
		c.circuitOpen.Store(true)
	}
}

func (c *Client) recordSuccess() {
	c.failures.Store(0)
	c.circuitOpen.Store(false)
}

// Debug logs at debug level.
func (c *Client) Debug(ctx context.Context, msg string, fields ...Field) {
	c.Log(ctx, LevelDebug, msg, fields...)
}

// Info logs at info level.
func (c *Client) Info(ctx context.Context, msg string, fields ...Field) {
	c.Log(ctx, LevelInfo, msg, fields...)
}

// Warn logs at warn level.
func (c *Client) Warn(ctx context.Context, msg string, fields ...Field) {
	c.Log(ctx, LevelWarn, msg, fields...)
}

// Error logs at error level.
func (c *Client) Error(ctx context.Context, msg string, fields ...Field) {
	c.Log(ctx, LevelError, msg, fields...)
}

// Fatal logs at fatal level.
func (c *Client) Fatal(ctx context.Context, msg string, fields ...Field) {
	c.Log(ctx, LevelFatal, msg, fields...)
	_ = c.Close() // Error intentionally ignored - exiting anyway
	os.Exit(1)
}

// Close flushes remaining logs and closes the connection.
func (c *Client) Close() error {
	if c.closed.Swap(true) {
		return nil
	}

	if c.flushTimer != nil {
		c.flushTimer.Stop()
	}

	c.flush()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Sync flushes any buffered log entries.
func (c *Client) Sync() error {
	c.flush()
	return nil
}

// IsCircuitOpen returns whether the circuit breaker is open.
func (c *Client) IsCircuitOpen() bool {
	return c.circuitOpen.Load()
}

// BufferSize returns the current buffer size.
func (c *Client) BufferSize() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.buffer)
}
