// Package logging provides centralized logging via platform logging-service.
package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/authcorp/libs/go/src/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// LogLevel represents log severity.
type LogLevel int32

const (
	LogLevelDebug LogLevel = 1
	LogLevelInfo  LogLevel = 2
	LogLevelWarn  LogLevel = 3
	LogLevelError LogLevel = 4
	LogLevelFatal LogLevel = 5
)

// LogEntry represents a structured log entry.
type LogEntry struct {
	Timestamp     time.Time
	Level         LogLevel
	Message       string
	CorrelationID string
	TenantID      string
	UserID        string
	ServiceID     string
	TraceID       string
	SpanID        string
	Metadata      map[string]string
	Error         error
}

// Client provides centralized logging with fallback.
type Client struct {
	conn          *grpc.ClientConn
	serviceID     string
	fallback      *observability.Logger
	buffer        chan *LogEntry
	circuitOpen   bool
	failures      int
	failThreshold int
	mu            sync.RWMutex
	wg            sync.WaitGroup
	done          chan struct{}
}

// Config holds logging client configuration.
type Config struct {
	Address       string
	ServiceID     string
	BufferSize    int
	FailThreshold int
	Timeout       time.Duration
}

// NewClient creates a new logging client with circuit breaker and fallback.
func NewClient(cfg Config) (*Client, error) {
	fallback := observability.NewLogger(cfg.ServiceID)

	client := &Client{
		serviceID:     cfg.ServiceID,
		fallback:      fallback,
		buffer:        make(chan *LogEntry, cfg.BufferSize),
		failThreshold: cfg.FailThreshold,
		done:          make(chan struct{}),
	}

	// Try to connect to logging service
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		// Start with circuit open, use fallback
		client.circuitOpen = true
		fallback.Warn("logging service unavailable, using fallback", map[string]any{
			"address": cfg.Address,
			"error":   err.Error(),
		})
	} else {
		client.conn = conn
	}

	// Start background worker
	client.wg.Add(1)
	go client.worker()

	return client, nil
}

// worker processes log entries in background.
func (c *Client) worker() {
	defer c.wg.Done()

	for {
		select {
		case entry := <-c.buffer:
			c.sendLog(entry)
		case <-c.done:
			// Drain remaining entries
			for len(c.buffer) > 0 {
				entry := <-c.buffer
				c.sendLog(entry)
			}
			return
		}
	}
}

// sendLog sends a log entry to the logging service or fallback.
func (c *Client) sendLog(entry *LogEntry) {
	c.mu.RLock()
	circuitOpen := c.circuitOpen
	c.mu.RUnlock()

	if circuitOpen || c.conn == nil {
		c.logToFallback(entry)
		return
	}

	// In production, this would call the gRPC service
	// For now, we use the fallback logger
	c.logToFallback(entry)
}

// logToFallback writes to local structured JSON logger.
func (c *Client) logToFallback(entry *LogEntry) {
	logger := c.fallback.
		WithCorrelationID(entry.CorrelationID).
		WithTraceContext(entry.TraceID, entry.SpanID)

	fields := map[string]any{
		"tenant_id":  entry.TenantID,
		"user_id":    entry.UserID,
		"service_id": entry.ServiceID,
	}
	for k, v := range entry.Metadata {
		fields[k] = v
	}

	switch entry.Level {
	case LogLevelDebug:
		logger.Debug(entry.Message, fields)
	case LogLevelInfo:
		logger.Info(entry.Message, fields)
	case LogLevelWarn:
		logger.Warn(entry.Message, fields)
	case LogLevelError, LogLevelFatal:
		if entry.Error != nil {
			fields["error"] = entry.Error.Error()
		}
		logger.Error(entry.Message, fields)
	}
}

// recordFailure records a failure and potentially opens circuit.
func (c *Client) recordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failures++
	if c.failures >= c.failThreshold {
		c.circuitOpen = true
	}
}

// recordSuccess records a success and potentially closes circuit.
func (c *Client) recordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failures = 0
	c.circuitOpen = false
}

// Log sends a log entry asynchronously.
func (c *Client) Log(ctx context.Context, entry *LogEntry) {
	// Enrich with context
	if entry.CorrelationID == "" {
		entry.CorrelationID = observability.CorrelationIDFromContext(ctx)
	}
	if entry.TraceID == "" || entry.SpanID == "" {
		entry.TraceID, entry.SpanID = observability.TraceContextFromContext(ctx)
	}
	if user, ok := observability.UserContextFromContext(ctx); ok {
		if entry.TenantID == "" {
			entry.TenantID = user.TenantID
		}
		if entry.UserID == "" {
			entry.UserID = user.UserID
		}
	}
	entry.ServiceID = c.serviceID
	entry.Timestamp = time.Now()

	select {
	case c.buffer <- entry:
	default:
		// Buffer full, log directly to fallback
		c.logToFallback(entry)
	}
}

// Debug logs at debug level.
func (c *Client) Debug(ctx context.Context, msg string, metadata map[string]string) {
	c.Log(ctx, &LogEntry{Level: LogLevelDebug, Message: msg, Metadata: metadata})
}

// Info logs at info level.
func (c *Client) Info(ctx context.Context, msg string, metadata map[string]string) {
	c.Log(ctx, &LogEntry{Level: LogLevelInfo, Message: msg, Metadata: metadata})
}

// Warn logs at warn level.
func (c *Client) Warn(ctx context.Context, msg string, metadata map[string]string) {
	c.Log(ctx, &LogEntry{Level: LogLevelWarn, Message: msg, Metadata: metadata})
}

// Error logs at error level.
func (c *Client) Error(ctx context.Context, msg string, err error, metadata map[string]string) {
	c.Log(ctx, &LogEntry{Level: LogLevelError, Message: msg, Error: err, Metadata: metadata})
}

// Close shuts down the logging client.
func (c *Client) Close() error {
	close(c.done)
	c.wg.Wait()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsCircuitOpen returns true if circuit breaker is open.
func (c *Client) IsCircuitOpen() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.circuitOpen
}

// FallbackLogger is a simple JSON logger for when logging service is unavailable.
type FallbackLogger struct {
	output    io.Writer
	serviceID string
	mu        sync.Mutex
}

// NewFallbackLogger creates a fallback logger.
func NewFallbackLogger(serviceID string) *FallbackLogger {
	return &FallbackLogger{
		output:    os.Stdout,
		serviceID: serviceID,
	}
}

// Log writes a structured JSON log entry.
func (l *FallbackLogger) Log(entry *LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	logData := map[string]any{
		"timestamp":  entry.Timestamp.Format(time.RFC3339Nano),
		"level":      levelToString(entry.Level),
		"message":    entry.Message,
		"service_id": l.serviceID,
	}

	if entry.CorrelationID != "" {
		logData["correlation_id"] = entry.CorrelationID
	}
	if entry.TenantID != "" {
		logData["tenant_id"] = entry.TenantID
	}
	if entry.UserID != "" {
		logData["user_id"] = entry.UserID
	}
	if entry.TraceID != "" {
		logData["trace_id"] = entry.TraceID
	}
	if entry.SpanID != "" {
		logData["span_id"] = entry.SpanID
	}
	if entry.Error != nil {
		logData["error"] = entry.Error.Error()
	}
	for k, v := range entry.Metadata {
		logData[k] = v
	}

	data, _ := json.Marshal(logData)
	fmt.Fprintln(l.output, string(data))
}

func levelToString(level LogLevel) string {
	switch level {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
