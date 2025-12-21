// Package observability provides structured logging and tracing.
package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Level represents log severity.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry.
type LogEntry struct {
	Timestamp     string         `json:"timestamp"`
	Level         string         `json:"level"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	TraceID       string         `json:"trace_id,omitempty"`
	SpanID        string         `json:"span_id,omitempty"`
	Fields        map[string]any `json:"fields,omitempty"`
	Service       string         `json:"service,omitempty"`
}

// Logger provides structured JSON logging.
type Logger struct {
	mu            sync.Mutex
	output        io.Writer
	level         Level
	fields        map[string]any
	service       string
	correlationID string
	traceID       string
	spanID        string
}

// NewLogger creates a new logger.
func NewLogger(service string) *Logger {
	return &Logger{
		output:  os.Stdout,
		level:   LevelInfo,
		fields:  make(map[string]any),
		service: service,
	}
}

// WithOutput sets the output writer.
func (l *Logger) WithOutput(w io.Writer) *Logger {
	l.output = w
	return l
}

// WithLevel sets the minimum log level.
func (l *Logger) WithLevel(level Level) *Logger {
	l.level = level
	return l
}

// With adds fields to the logger.
func (l *Logger) With(fields map[string]any) *Logger {
	newLogger := &Logger{
		output:        l.output,
		level:         l.level,
		fields:        make(map[string]any),
		service:       l.service,
		correlationID: l.correlationID,
		traceID:       l.traceID,
		spanID:        l.spanID,
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// WithCorrelationID sets the correlation ID.
func (l *Logger) WithCorrelationID(id string) *Logger {
	newLogger := l.With(nil)
	newLogger.correlationID = id
	return newLogger
}

// WithTraceContext sets trace context.
func (l *Logger) WithTraceContext(traceID, spanID string) *Logger {
	newLogger := l.With(nil)
	newLogger.traceID = traceID
	newLogger.spanID = spanID
	return newLogger
}

// Debug logs at debug level.
func (l *Logger) Debug(msg string, fields ...map[string]any) {
	l.log(LevelDebug, msg, fields...)
}

// Info logs at info level.
func (l *Logger) Info(msg string, fields ...map[string]any) {
	l.log(LevelInfo, msg, fields...)
}

// Warn logs at warn level.
func (l *Logger) Warn(msg string, fields ...map[string]any) {
	l.log(LevelWarn, msg, fields...)
}

// Error logs at error level.
func (l *Logger) Error(msg string, fields ...map[string]any) {
	l.log(LevelError, msg, fields...)
}

func (l *Logger) log(level Level, msg string, extraFields ...map[string]any) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Level:         level.String(),
		Message:       msg,
		Service:       l.service,
		CorrelationID: l.correlationID,
		TraceID:       l.traceID,
		SpanID:        l.spanID,
	}

	// Merge fields
	if len(l.fields) > 0 || len(extraFields) > 0 {
		entry.Fields = make(map[string]any)
		for k, v := range l.fields {
			entry.Fields[k] = RedactSensitive(k, v)
		}
		for _, fields := range extraFields {
			for k, v := range fields {
				entry.Fields[k] = RedactSensitive(k, v)
			}
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	data, _ := json.Marshal(entry)
	fmt.Fprintln(l.output, string(data))
}

// FromContext creates a logger with context values.
func (l *Logger) FromContext(ctx context.Context) *Logger {
	newLogger := l.With(nil)
	if id := CorrelationIDFromContext(ctx); id != "" {
		newLogger.correlationID = id
	}
	if traceID, spanID := TraceContextFromContext(ctx); traceID != "" {
		newLogger.traceID = traceID
		newLogger.spanID = spanID
	}
	return newLogger
}

// sensitivePatterns for PII redaction.
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|passwd|pwd)`),
	regexp.MustCompile(`(?i)(token|api[_-]?key|secret|credential)`),
	regexp.MustCompile(`(?i)(ssn|social[_-]?security)`),
	regexp.MustCompile(`(?i)(credit[_-]?card|card[_-]?number)`),
}

// RedactSensitive redacts sensitive field values.
func RedactSensitive(key string, value any) any {
	lowerKey := strings.ToLower(key)
	for _, pattern := range sensitivePatterns {
		if pattern.MatchString(lowerKey) {
			return "[REDACTED]"
		}
	}
	if str, ok := value.(string); ok {
		return redactPII(str)
	}
	return value
}

var piiPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
	regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
	regexp.MustCompile(`\b\d{3}[-]?\d{2}[-]?\d{4}\b`),
}

func redactPII(s string) string {
	result := s
	for _, pattern := range piiPatterns {
		result = pattern.ReplaceAllString(result, "[PII]")
	}
	return result
}
