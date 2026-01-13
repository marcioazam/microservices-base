package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// localLogger provides stdout logging as fallback.
type localLogger struct {
	output  io.Writer
	service string
}

// newLocalLogger creates a new local logger.
func newLocalLogger(service string) *localLogger {
	return &localLogger{
		output:  os.Stdout,
		service: service,
	}
}

// Log writes a log entry to stdout.
func (l *localLogger) Log(entry LogEntry) {
	output := map[string]any{
		"timestamp": entry.Timestamp.Format(time.RFC3339Nano),
		"level":     entry.Level.String(),
		"message":   entry.Message,
		"service":   entry.Service,
	}

	if entry.CorrelationID != "" {
		output["correlation_id"] = entry.CorrelationID
	}
	if entry.TraceID != "" {
		output["trace_id"] = entry.TraceID
	}
	if entry.SpanID != "" {
		output["span_id"] = entry.SpanID
	}
	if len(entry.Fields) > 0 {
		output["fields"] = entry.Fields
	}

	data, _ := json.Marshal(output)
	fmt.Fprintln(l.output, string(data))
}

// LogBatch writes multiple log entries to stdout.
func (l *localLogger) LogBatch(entries []LogEntry) {
	for _, entry := range entries {
		l.Log(entry)
	}
}
