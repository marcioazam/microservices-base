package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditEvent represents a structured audit log entry
type AuditEvent struct {
	EventID       string            `json:"event_id"`
	Timestamp     time.Time         `json:"timestamp"`
	CorrelationID string            `json:"correlation_id"`
	TraceID       string            `json:"trace_id,omitempty"`
	SpanID        string            `json:"span_id,omitempty"`
	EventType     string            `json:"event_type"`
	Action        string            `json:"action"`
	Result        string            `json:"result"`
	UserID        string            `json:"user_id,omitempty"`
	SessionID     string            `json:"session_id,omitempty"`
	ClientID      string            `json:"client_id,omitempty"`
	ServiceName   string            `json:"service_name"`
	IPAddress     string            `json:"ip_address,omitempty"`
	UserAgent     string            `json:"user_agent,omitempty"`
	ResourceType  string            `json:"resource_type,omitempty"`
	ResourceID    string            `json:"resource_id,omitempty"`
	RequestData   map[string]any    `json:"request_data,omitempty"`
	ResponseData  map[string]any    `json:"response_data,omitempty"`
	ErrorCode     string            `json:"error_code,omitempty"`
	ErrorMessage  string            `json:"error_message,omitempty"`
	DurationMs    int64             `json:"duration_ms"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// AuditLogger handles audit event logging
type AuditLogger struct {
	serviceName string
	publisher   EventPublisher
}

// EventPublisher interface for publishing audit events
type EventPublisher interface {
	Publish(ctx context.Context, topic string, event []byte) error
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(serviceName string, publisher EventPublisher) *AuditLogger {
	return &AuditLogger{
		serviceName: serviceName,
		publisher:   publisher,
	}
}

// LogAuthEvent logs an authentication event
func (l *AuditLogger) LogAuthEvent(ctx context.Context, event *AuditEvent) error {
	event.EventID = uuid.New().String()
	event.Timestamp = time.Now().UTC()
	event.ServiceName = l.serviceName
	event.EventType = "authentication"

	if event.CorrelationID == "" {
		event.CorrelationID = uuid.New().String()
	}

	return l.publish(ctx, "auth.events", event)
}

// LogAuthzEvent logs an authorization event
func (l *AuditLogger) LogAuthzEvent(ctx context.Context, event *AuditEvent) error {
	event.EventID = uuid.New().String()
	event.Timestamp = time.Now().UTC()
	event.ServiceName = l.serviceName
	event.EventType = "authorization"

	if event.CorrelationID == "" {
		event.CorrelationID = uuid.New().String()
	}

	return l.publish(ctx, "authz.events", event)
}

// LogSecurityEvent logs a security-related event
func (l *AuditLogger) LogSecurityEvent(ctx context.Context, event *AuditEvent) error {
	event.EventID = uuid.New().String()
	event.Timestamp = time.Now().UTC()
	event.ServiceName = l.serviceName
	event.EventType = "security"

	if event.CorrelationID == "" {
		event.CorrelationID = uuid.New().String()
	}

	return l.publish(ctx, "security.events", event)
}

func (l *AuditLogger) publish(ctx context.Context, topic string, event *AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return l.publisher.Publish(ctx, topic, data)
}

// ValidateEvent checks if an audit event has all required fields
func ValidateEvent(event *AuditEvent) bool {
	return event.Timestamp != (time.Time{}) &&
		event.CorrelationID != "" &&
		event.Action != "" &&
		event.Result != "" &&
		event.ServiceName != ""
}

// FilterEvents filters audit events based on criteria
func FilterEvents(events []AuditEvent, filter EventFilter) []AuditEvent {
	var result []AuditEvent

	for _, event := range events {
		if matchesFilter(event, filter) {
			result = append(result, event)
		}
	}

	return result
}

// EventFilter defines criteria for filtering audit events
type EventFilter struct {
	UserID        string
	Action        string
	StartTime     *time.Time
	EndTime       *time.Time
	CorrelationID string
	EventType     string
}

func matchesFilter(event AuditEvent, filter EventFilter) bool {
	if filter.UserID != "" && event.UserID != filter.UserID {
		return false
	}
	if filter.Action != "" && event.Action != filter.Action {
		return false
	}
	if filter.CorrelationID != "" && event.CorrelationID != filter.CorrelationID {
		return false
	}
	if filter.EventType != "" && event.EventType != filter.EventType {
		return false
	}
	if filter.StartTime != nil && event.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && event.Timestamp.After(*filter.EndTime) {
		return false
	}
	return true
}
