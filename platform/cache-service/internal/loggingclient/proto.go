package loggingclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// LogLevel enum matching logging-service proto.
type LogLevel int32

const (
	LogLevel_LOG_LEVEL_UNSPECIFIED LogLevel = 0
	LogLevel_LOG_LEVEL_DEBUG       LogLevel = 1
	LogLevel_LOG_LEVEL_INFO        LogLevel = 2
	LogLevel_LOG_LEVEL_WARN        LogLevel = 3
	LogLevel_LOG_LEVEL_ERROR       LogLevel = 4
	LogLevel_LOG_LEVEL_FATAL       LogLevel = 5
)

// String returns the string representation of LogLevel.
func (l LogLevel) String() string {
	switch l {
	case LogLevel_LOG_LEVEL_DEBUG:
		return "DEBUG"
	case LogLevel_LOG_LEVEL_INFO:
		return "INFO"
	case LogLevel_LOG_LEVEL_WARN:
		return "WARN"
	case LogLevel_LOG_LEVEL_ERROR:
		return "ERROR"
	case LogLevel_LOG_LEVEL_FATAL:
		return "FATAL"
	default:
		return "UNSPECIFIED"
	}
}

// LogEntryMessage represents a log entry.
type LogEntryMessage struct {
	Id            string                 `json:"id"`
	Timestamp     *timestamppb.Timestamp `json:"timestamp"`
	CorrelationId string                 `json:"correlation_id"`
	ServiceId     string                 `json:"service_id"`
	Level         LogLevel               `json:"level"`
	Message       string                 `json:"message"`
	TraceId       *string                `json:"trace_id,omitempty"`
	SpanId        *string                `json:"span_id,omitempty"`
	UserId        *string                `json:"user_id,omitempty"`
	RequestId     *string                `json:"request_id,omitempty"`
	Method        *string                `json:"method,omitempty"`
	Path          *string                `json:"path,omitempty"`
	StatusCode    *int32                 `json:"status_code,omitempty"`
	DurationMs    *int64                 `json:"duration_ms,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
}

// IngestLogRequest represents a single log ingest request.
type IngestLogRequest struct {
	Entry *LogEntryMessage `json:"entry"`
}

// IngestLogResponse represents a single log ingest response.
type IngestLogResponse struct {
	Success bool   `json:"success"`
	Id      string `json:"id"`
}

// IngestLogBatchRequest represents a batch log ingest request.
type IngestLogBatchRequest struct {
	Entries []*LogEntryMessage `json:"entries"`
}

// IngestLogBatchResponse represents a batch log ingest response.
type IngestLogBatchResponse struct {
	AcceptedCount int32                `json:"accepted_count"`
	RejectedCount int32                `json:"rejected_count"`
	Results       []*IngestLogResponse `json:"results"`
}

// LoggingServiceClient is the client interface for logging service.
type LoggingServiceClient interface {
	IngestLog(ctx context.Context, in *IngestLogRequest, opts ...grpc.CallOption) (*IngestLogResponse, error)
	IngestLogBatch(ctx context.Context, in *IngestLogBatchRequest, opts ...grpc.CallOption) (*IngestLogBatchResponse, error)
}

// loggingServiceClient implements LoggingServiceClient.
type loggingServiceClient struct {
	cc grpc.ClientConnInterface
}

// NewLoggingServiceClient creates a new logging service client.
func NewLoggingServiceClient(cc grpc.ClientConnInterface) LoggingServiceClient {
	return &loggingServiceClient{cc}
}

func (c *loggingServiceClient) IngestLog(ctx context.Context, in *IngestLogRequest, opts ...grpc.CallOption) (*IngestLogResponse, error) {
	out := new(IngestLogResponse)
	err := c.cc.Invoke(ctx, "/logging.v1.LoggingService/IngestLog", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *loggingServiceClient) IngestLogBatch(ctx context.Context, in *IngestLogBatchRequest, opts ...grpc.CallOption) (*IngestLogBatchResponse, error) {
	out := new(IngestLogBatchResponse)
	err := c.cc.Invoke(ctx, "/logging.v1.LoggingService/IngestLogBatch", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
