using System.Text.Json.Serialization;

namespace LoggingService.Core.Models;

/// <summary>
/// Represents a structured log entry from a microservice.
/// </summary>
public sealed record LogEntry
{
    /// <summary>
    /// Unique identifier for the log entry.
    /// </summary>
    public required string Id { get; init; }

    /// <summary>
    /// Timestamp when the log was generated (ISO 8601 UTC).
    /// </summary>
    public required DateTimeOffset Timestamp { get; init; }

    /// <summary>
    /// Correlation ID for tracing requests across services.
    /// </summary>
    public required string CorrelationId { get; init; }

    /// <summary>
    /// Identifier of the service that generated the log.
    /// </summary>
    public required string ServiceId { get; init; }

    /// <summary>
    /// Severity level of the log entry.
    /// </summary>
    public required LogLevel Level { get; init; }

    /// <summary>
    /// Log message content.
    /// </summary>
    public required string Message { get; init; }

    /// <summary>
    /// OpenTelemetry trace ID for distributed tracing.
    /// </summary>
    public string? TraceId { get; init; }

    /// <summary>
    /// OpenTelemetry span ID for distributed tracing.
    /// </summary>
    public string? SpanId { get; init; }

    /// <summary>
    /// User identifier associated with the log entry.
    /// </summary>
    public string? UserId { get; init; }

    /// <summary>
    /// Request identifier for the originating request.
    /// </summary>
    public string? RequestId { get; init; }

    /// <summary>
    /// HTTP method of the request (GET, POST, etc.).
    /// </summary>
    public string? Method { get; init; }

    /// <summary>
    /// Request path or endpoint.
    /// </summary>
    public string? Path { get; init; }

    /// <summary>
    /// HTTP status code of the response.
    /// </summary>
    public int? StatusCode { get; init; }

    /// <summary>
    /// Duration of the operation in milliseconds.
    /// </summary>
    public long? DurationMs { get; init; }

    /// <summary>
    /// Additional metadata as key-value pairs.
    /// </summary>
    public Dictionary<string, object>? Metadata { get; init; }

    /// <summary>
    /// Exception information if the log is related to an error.
    /// </summary>
    public ExceptionInfo? Exception { get; init; }
}

/// <summary>
/// Log severity levels.
/// </summary>
[JsonConverter(typeof(JsonStringEnumConverter<LogLevel>))]
public enum LogLevel
{
    Debug = 0,
    Info = 1,
    Warn = 2,
    Error = 3,
    Fatal = 4
}

/// <summary>
/// Exception information for error logs.
/// </summary>
public sealed record ExceptionInfo
{
    /// <summary>
    /// Exception type name.
    /// </summary>
    public required string Type { get; init; }

    /// <summary>
    /// Exception message.
    /// </summary>
    public required string Message { get; init; }

    /// <summary>
    /// Stack trace of the exception.
    /// </summary>
    public string? StackTrace { get; init; }

    /// <summary>
    /// Inner exception information.
    /// </summary>
    public ExceptionInfo? InnerException { get; init; }
}
