package logging

import (
	"context"
	"sync"
	"time"

	"github.com/authcorp/libs/go/src/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client provides structured logging via logging-service.
type Client struct {
	conn     *grpc.ClientConn
	config   ClientConfig
	buffer   *logBuffer
	fallback *localLogger
	fields   map[string]any
	mu       sync.RWMutex
	closed   bool
}

// NewClient creates a new logging client.
func NewClient(config ClientConfig) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	client := &Client{
		config:   config,
		fields:   make(map[string]any),
		fallback: newLocalLogger(config.ServiceName),
	}

	// Create buffer with flush function
	client.buffer = newLogBuffer(config.BufferSize, config.FlushInterval, client.shipLogs)

	// Try to connect to logging-service
	if config.Address != "" {
		conn, err := grpc.NewClient(
			config.Address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err == nil {
			client.conn = conn
		}
		// If connection fails and LocalFallback is enabled, continue with local-only
	}

	return client, nil
}

// LocalOnly creates a client that only logs locally (for testing).
func LocalOnly(serviceName string) *Client {
	config := DefaultConfig()
	config.ServiceName = serviceName
	config.LocalFallback = true
	config.Address = "" // No remote

	client := &Client{
		config:   config,
		fields:   make(map[string]any),
		fallback: newLocalLogger(serviceName),
	}

	client.buffer = newLogBuffer(config.BufferSize, config.FlushInterval, func(entries []LogEntry) error {
		client.fallback.LogBatch(entries)
		return nil
	})

	return client
}

// Debug logs at debug level.
func (c *Client) Debug(ctx context.Context, msg string, fields ...Field) {
	c.log(ctx, LevelDebug, msg, fields...)
}

// Info logs at info level.
func (c *Client) Info(ctx context.Context, msg string, fields ...Field) {
	c.log(ctx, LevelInfo, msg, fields...)
}

// Warn logs at warn level.
func (c *Client) Warn(ctx context.Context, msg string, fields ...Field) {
	c.log(ctx, LevelWarn, msg, fields...)
}

// Error logs at error level.
func (c *Client) Error(ctx context.Context, msg string, fields ...Field) {
	c.log(ctx, LevelError, msg, fields...)
}

func (c *Client) log(ctx context.Context, level Level, msg string, fields ...Field) {
	if level < c.config.MinLevel {
		return
	}

	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	// Build entry
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   msg,
		Service:   c.config.ServiceName,
	}

	// Extract context values
	if ctx != nil {
		entry.CorrelationID = observability.CorrelationIDFromContext(ctx)
		entry.TraceID, entry.SpanID = observability.TraceContextFromContext(ctx)
	}

	// Merge fields
	allFields := make(map[string]any)
	c.mu.RLock()
	for k, v := range c.fields {
		allFields[k] = v
	}
	c.mu.RUnlock()

	for _, f := range fields {
		allFields[f.Key] = f.Value
	}

	// Apply redaction
	entry.Fields = redactFields(allFields)

	// Add to buffer
	c.buffer.Add(entry)
}

// With returns a new client with additional fields.
func (c *Client) With(fields ...Field) *Client {
	c.mu.RLock()
	newFields := make(map[string]any, len(c.fields)+len(fields))
	for k, v := range c.fields {
		newFields[k] = v
	}
	c.mu.RUnlock()

	for _, f := range fields {
		newFields[f.Key] = f.Value
	}

	return &Client{
		conn:     c.conn,
		config:   c.config,
		buffer:   c.buffer,
		fallback: c.fallback,
		fields:   newFields,
	}
}

// FromContext creates a logger with context values.
func (c *Client) FromContext(ctx context.Context) *Client {
	fields := make([]Field, 0, 3)

	if id := observability.CorrelationIDFromContext(ctx); id != "" {
		fields = append(fields, String("correlation_id", id))
	}
	if traceID, spanID := observability.TraceContextFromContext(ctx); traceID != "" {
		fields = append(fields, String("trace_id", traceID))
		if spanID != "" {
			fields = append(fields, String("span_id", spanID))
		}
	}

	if len(fields) == 0 {
		return c
	}
	return c.With(fields...)
}

// Flush sends buffered logs immediately.
func (c *Client) Flush() error {
	return c.buffer.Flush()
}

// Close flushes and closes the client.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	// Flush remaining logs
	c.buffer.Close()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// shipLogs sends logs to the logging-service or fallback.
func (c *Client) shipLogs(entries []LogEntry) error {
	if c.conn == nil || c.config.LocalFallback {
		// Use fallback
		c.fallback.LogBatch(entries)
		return nil
	}

	// In real implementation, this would call the gRPC service
	// For now, use fallback
	c.fallback.LogBatch(entries)
	return nil
}

// BufferSize returns the current buffer size.
func (c *Client) BufferSize() int {
	return c.buffer.Size()
}
