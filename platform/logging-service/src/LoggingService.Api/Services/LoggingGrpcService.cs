using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using LoggingService.Api.Grpc;
using LoggingService.Core.Interfaces;
using LoggingService.Core.Models;
using GrpcLogLevel = LoggingService.Api.Grpc.LogLevel;
using CoreLogLevel = LoggingService.Core.Models.LogLevel;
using GrpcSortDirection = LoggingService.Api.Grpc.SortDirection;
using CoreSortDirection = LoggingService.Core.Models.SortDirection;

namespace LoggingService.Api.Services;

/// <summary>
/// gRPC service implementation for log ingestion and querying.
/// </summary>
public sealed class LoggingGrpcService : Grpc.LoggingService.LoggingServiceBase
{
    private readonly ILogIngestionService _ingestionService;
    private readonly ILogRepository _repository;
    private readonly ILogger<LoggingGrpcService> _logger;

    public LoggingGrpcService(
        ILogIngestionService ingestionService,
        ILogRepository repository,
        ILogger<LoggingGrpcService> logger)
    {
        _ingestionService = ingestionService;
        _repository = repository;
        _logger = logger;
    }

    public override async Task<IngestLogResponse> IngestLog(
        IngestLogRequest request,
        ServerCallContext context)
    {
        var entry = ToLogEntry(request.Entry);
        var result = await _ingestionService.IngestAsync(entry, context.CancellationToken);

        return new IngestLogResponse
        {
            Success = result.IsSuccess,
            Id = result.LogId ?? string.Empty,
            Errors = { result.FieldErrors?.Select(ToFieldErrorMessage) ?? [] }
        };
    }

    public override async Task<IngestLogBatchResponse> IngestLogBatch(
        IngestLogBatchRequest request,
        ServerCallContext context)
    {
        if (request.Entries.Count > 1000)
        {
            throw new RpcException(new Status(
                StatusCode.InvalidArgument,
                $"Batch size {request.Entries.Count} exceeds maximum of 1000"));
        }

        var entries = request.Entries.Select(ToLogEntry);
        var result = await _ingestionService.IngestBatchAsync(entries, context.CancellationToken);

        return new IngestLogBatchResponse
        {
            AcceptedCount = result.SuccessCount,
            RejectedCount = result.FailedCount,
            Results =
            {
                result.Results.Select(r => new IngestLogResponse
                {
                    Success = r.IsSuccess,
                    Id = r.LogId ?? string.Empty,
                    Errors = { r.FieldErrors?.Select(ToFieldErrorMessage) ?? [] }
                })
            }
        };
    }

    public override async Task<QueryLogsResponse> QueryLogs(
        QueryLogsRequest request,
        ServerCallContext context)
    {
        var query = ToLogQuery(request);
        var result = await _repository.QueryAsync(query, context.CancellationToken);

        return new QueryLogsResponse
        {
            Items = { result.Items.Select(ToLogEntryMessage) },
            TotalCount = result.TotalCount,
            Page = result.Page,
            PageSize = result.PageSize,
            HasMore = result.HasMore
        };
    }

    public override async Task StreamLogs(
        StreamLogsRequest request,
        IServerStreamWriter<LogEntryMessage> responseStream,
        ServerCallContext context)
    {
        var query = new LogQuery
        {
            ServiceId = request.ServiceId,
            MinLevel = request.HasMinLevel ? ToCoreLogLevel(request.MinLevel) : null,
            Page = 1,
            PageSize = 100,
            SortDirection = CoreSortDirection.Descending
        };

        while (!context.CancellationToken.IsCancellationRequested)
        {
            var result = await _repository.QueryAsync(query, context.CancellationToken);

            foreach (var entry in result.Items)
            {
                await responseStream.WriteAsync(ToLogEntryMessage(entry));
            }

            await Task.Delay(1000, context.CancellationToken);
        }
    }

    private static LogEntry ToLogEntry(LogEntryMessage msg) => new()
    {
        Id = string.IsNullOrEmpty(msg.Id) ? Guid.NewGuid().ToString() : msg.Id,
        Timestamp = msg.Timestamp?.ToDateTimeOffset() ?? DateTimeOffset.UtcNow,
        CorrelationId = msg.CorrelationId ?? string.Empty,
        ServiceId = msg.ServiceId,
        Level = ToCoreLogLevel(msg.Level),
        Message = msg.Message,
        TraceId = msg.HasTraceId ? msg.TraceId : null,
        SpanId = msg.HasSpanId ? msg.SpanId : null,
        UserId = msg.HasUserId ? msg.UserId : null,
        RequestId = msg.HasRequestId ? msg.RequestId : null,
        Method = msg.HasMethod ? msg.Method : null,
        Path = msg.HasPath ? msg.Path : null,
        StatusCode = msg.HasStatusCode ? msg.StatusCode : null,
        DurationMs = msg.HasDurationMs ? msg.DurationMs : null,
        Metadata = msg.Metadata.Count > 0
            ? msg.Metadata.ToDictionary(k => k.Key, v => (object)v.Value)
            : null,
        Exception = msg.Exception != null ? ToExceptionInfo(msg.Exception) : null
    };

    private static ExceptionInfo ToExceptionInfo(ExceptionInfoMessage msg) => new()
    {
        Type = msg.Type,
        Message = msg.Message,
        StackTrace = msg.HasStackTrace ? msg.StackTrace : null,
        InnerException = msg.InnerException != null ? ToExceptionInfo(msg.InnerException) : null
    };

    private static LogEntryMessage ToLogEntryMessage(LogEntry entry)
    {
        var msg = new LogEntryMessage
        {
            Id = entry.Id,
            Timestamp = Timestamp.FromDateTimeOffset(entry.Timestamp),
            CorrelationId = entry.CorrelationId,
            ServiceId = entry.ServiceId,
            Level = ToGrpcLogLevel(entry.Level),
            Message = entry.Message
        };

        if (entry.TraceId != null) msg.TraceId = entry.TraceId;
        if (entry.SpanId != null) msg.SpanId = entry.SpanId;
        if (entry.UserId != null) msg.UserId = entry.UserId;
        if (entry.RequestId != null) msg.RequestId = entry.RequestId;
        if (entry.Method != null) msg.Method = entry.Method;
        if (entry.Path != null) msg.Path = entry.Path;
        if (entry.StatusCode.HasValue) msg.StatusCode = entry.StatusCode.Value;
        if (entry.DurationMs.HasValue) msg.DurationMs = entry.DurationMs.Value;
        if (entry.Metadata != null)
        {
            foreach (var kvp in entry.Metadata)
                msg.Metadata[kvp.Key] = kvp.Value?.ToString() ?? string.Empty;
        }
        if (entry.Exception != null) msg.Exception = ToExceptionInfoMessage(entry.Exception);

        return msg;
    }

    private static ExceptionInfoMessage ToExceptionInfoMessage(ExceptionInfo ex)
    {
        var msg = new ExceptionInfoMessage { Type = ex.Type, Message = ex.Message };
        if (ex.StackTrace != null) msg.StackTrace = ex.StackTrace;
        if (ex.InnerException != null) msg.InnerException = ToExceptionInfoMessage(ex.InnerException);
        return msg;
    }

    private static FieldErrorMessage ToFieldErrorMessage(FieldError err) => new()
    {
        Field = err.Field,
        Code = err.Code,
        Message = err.Message
    };

    private static LogQuery ToLogQuery(QueryLogsRequest req) => new()
    {
        StartTime = req.StartTime?.ToDateTimeOffset(),
        EndTime = req.EndTime?.ToDateTimeOffset(),
        ServiceId = req.HasServiceId ? req.ServiceId : null,
        MinLevel = req.HasMinLevel ? ToCoreLogLevel(req.MinLevel) : null,
        CorrelationId = req.HasCorrelationId ? req.CorrelationId : null,
        SearchText = req.HasSearchText ? req.SearchText : null,
        Page = req.Page > 0 ? req.Page : 1,
        PageSize = Math.Min(req.PageSize > 0 ? req.PageSize : 100, 1000),
        SortDirection = ToCoreSortDirection(req.SortDirection)
    };

    private static CoreLogLevel ToCoreLogLevel(GrpcLogLevel level) => level switch
    {
        GrpcLogLevel.Debug => CoreLogLevel.Debug,
        GrpcLogLevel.Info => CoreLogLevel.Info,
        GrpcLogLevel.Warn => CoreLogLevel.Warn,
        GrpcLogLevel.Error => CoreLogLevel.Error,
        GrpcLogLevel.Fatal => CoreLogLevel.Fatal,
        _ => CoreLogLevel.Info
    };

    private static GrpcLogLevel ToGrpcLogLevel(CoreLogLevel level) => level switch
    {
        CoreLogLevel.Debug => GrpcLogLevel.Debug,
        CoreLogLevel.Info => GrpcLogLevel.Info,
        CoreLogLevel.Warn => GrpcLogLevel.Warn,
        CoreLogLevel.Error => GrpcLogLevel.Error,
        CoreLogLevel.Fatal => GrpcLogLevel.Fatal,
        _ => GrpcLogLevel.Info
    };

    private static CoreSortDirection ToCoreSortDirection(GrpcSortDirection dir) => dir switch
    {
        GrpcSortDirection.Ascending => CoreSortDirection.Ascending,
        GrpcSortDirection.Descending => CoreSortDirection.Descending,
        _ => CoreSortDirection.Descending
    };
}
