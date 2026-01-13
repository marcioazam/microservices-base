using LoggingService.Core.Models;
using LogLevel = LoggingService.Core.Models.LogLevel;

namespace LoggingService.Api.Models;

/// <summary>
/// Request model for single log entry ingestion.
/// </summary>
public sealed record LogEntryRequest
{
    public DateTimeOffset? Timestamp { get; init; }
    public string? CorrelationId { get; init; }
    public required string ServiceId { get; init; }
    public required LogLevel Level { get; init; }
    public required string Message { get; init; }
    public string? TraceId { get; init; }
    public string? SpanId { get; init; }
    public string? UserId { get; init; }
    public string? RequestId { get; init; }
    public string? Method { get; init; }
    public string? Path { get; init; }
    public int? StatusCode { get; init; }
    public long? DurationMs { get; init; }
    public Dictionary<string, object>? Metadata { get; init; }
    public ExceptionInfoRequest? Exception { get; init; }

    public LogEntry ToLogEntry() => new()
    {
        Id = Guid.NewGuid().ToString(),
        Timestamp = Timestamp ?? DateTimeOffset.UtcNow,
        CorrelationId = CorrelationId ?? string.Empty,
        ServiceId = ServiceId,
        Level = Level,
        Message = Message,
        TraceId = TraceId,
        SpanId = SpanId,
        UserId = UserId,
        RequestId = RequestId,
        Method = Method,
        Path = Path,
        StatusCode = StatusCode,
        DurationMs = DurationMs,
        Metadata = Metadata,
        Exception = Exception?.ToExceptionInfo()
    };
}

/// <summary>
/// Request model for exception information.
/// </summary>
public sealed record ExceptionInfoRequest
{
    public required string Type { get; init; }
    public required string Message { get; init; }
    public string? StackTrace { get; init; }
    public ExceptionInfoRequest? InnerException { get; init; }

    public ExceptionInfo ToExceptionInfo() => new()
    {
        Type = Type,
        Message = Message,
        StackTrace = StackTrace,
        InnerException = InnerException?.ToExceptionInfo()
    };
}

/// <summary>
/// Request model for batch log entry ingestion.
/// </summary>
public sealed record BatchLogEntryRequest
{
    public required IReadOnlyList<LogEntryRequest> Entries { get; init; }
}

/// <summary>
/// Request model for log queries.
/// </summary>
public sealed record LogQueryRequest
{
    public DateTimeOffset? StartTime { get; init; }
    public DateTimeOffset? EndTime { get; init; }
    public string? ServiceId { get; init; }
    public LogLevel? MinLevel { get; init; }
    public string? CorrelationId { get; init; }
    public string? SearchText { get; init; }
    public int Page { get; init; } = 1;
    public int PageSize { get; init; } = 100;
    public SortDirection SortDirection { get; init; } = SortDirection.Descending;

    public LogQuery ToLogQuery() => new()
    {
        StartTime = StartTime,
        EndTime = EndTime,
        ServiceId = ServiceId,
        MinLevel = MinLevel,
        CorrelationId = CorrelationId,
        SearchText = SearchText,
        Page = Page,
        PageSize = Math.Min(PageSize, 1000),
        SortDirection = SortDirection
    };
}
