package errors

import (
	"regexp"
	"strings"
)

// APIResponse represents an error response for APIs.
type APIResponse struct {
	Error   string         `json:"error"`
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	TraceID string         `json:"trace_id,omitempty"`
}

// sensitivePatterns defines patterns to redact from error messages.
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*\S+`),
	regexp.MustCompile(`(?i)(token|api[_-]?key|secret)\s*[:=]\s*\S+`),
	regexp.MustCompile(`(?i)(bearer)\s+\S+`),
	regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
	regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
	regexp.MustCompile(`\b\d{3}[-]?\d{2}[-]?\d{4}\b`), // SSN
	regexp.MustCompile(`\b\d{16}\b`),                   // Credit card
}

// ToAPIResponse converts an AppError to an API response with redaction.
func (e *AppError) ToAPIResponse() APIResponse {
	// Redact sensitive information from message
	message := redactSensitive(e.Message)

	// For internal errors, don't expose details
	if e.Code == ErrCodeInternal || e.Code == ErrCodeDependency {
		return APIResponse{
			Error:   "Internal Server Error",
			Code:    e.Code,
			Message: "An internal error occurred",
			TraceID: e.CorrelationID,
		}
	}

	// Redact sensitive details
	safeDetails := redactDetails(e.Details)

	return APIResponse{
		Error:   string(e.Code),
		Code:    e.Code,
		Message: message,
		Details: safeDetails,
		TraceID: e.CorrelationID,
	}
}

// redactSensitive removes sensitive information from a string.
func redactSensitive(s string) string {
	result := s
	for _, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllString(result, "[REDACTED]")
	}
	return result
}

// redactDetails removes sensitive information from details map.
func redactDetails(details map[string]any) map[string]any {
	if details == nil {
		return nil
	}

	sensitiveKeys := map[string]bool{
		"password": true, "passwd": true, "pwd": true,
		"token": true, "api_key": true, "apikey": true,
		"secret": true, "credential": true, "auth": true,
		"ssn": true, "credit_card": true, "card_number": true,
	}

	safe := make(map[string]any)
	for k, v := range details {
		lowerKey := strings.ToLower(k)
		if sensitiveKeys[lowerKey] {
			safe[k] = "[REDACTED]"
		} else if str, ok := v.(string); ok {
			safe[k] = redactSensitive(str)
		} else {
			safe[k] = v
		}
	}
	return safe
}

// LogEntry represents a structured log entry for errors.
type LogEntry struct {
	Level         string         `json:"level"`
	Code          ErrorCode      `json:"code"`
	Message       string         `json:"message"`
	Details       map[string]any `json:"details,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Timestamp     string         `json:"timestamp"`
	Stack         []string       `json:"stack,omitempty"`
}

// ToLogEntry converts an AppError to a structured log entry.
func (e *AppError) ToLogEntry() LogEntry {
	level := "error"
	if e.HTTPStatus() < 500 {
		level = "warn"
	}

	// Build error chain for stack
	var stack []string
	for _, err := range Chain(e) {
		stack = append(stack, err.Error())
	}

	return LogEntry{
		Level:         level,
		Code:          e.Code,
		Message:       e.Message,
		Details:       e.Details,
		CorrelationID: e.CorrelationID,
		Timestamp:     e.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
		Stack:         stack,
	}
}
