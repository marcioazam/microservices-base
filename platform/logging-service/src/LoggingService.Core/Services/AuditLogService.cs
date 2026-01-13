using LoggingService.Core.Interfaces;
using LoggingService.Core.Models;
using Microsoft.Extensions.Logging;
using LogLevel = LoggingService.Core.Models.LogLevel;

namespace LoggingService.Core.Services;

/// <summary>
/// Service for recording audit logs of query operations.
/// </summary>
public sealed class AuditLogService : IAuditLogService
{
    private readonly ILogRepository _repository;
    private readonly ILogger<AuditLogService> _logger;
    private const string AuditServiceId = "logging-service-audit";

    public AuditLogService(
        ILogRepository repository,
        ILogger<AuditLogService> logger)
    {
        _repository = repository;
        _logger = logger;
    }

    public async Task LogQueryAsync(
        string userId,
        LogQuery query,
        int resultCount,
        CancellationToken ct = default)
    {
        var auditEntry = CreateAuditEntry(
            userId,
            "QUERY",
            new Dictionary<string, object>
            {
                ["startTime"] = query.StartTime?.ToString("O") ?? "null",
                ["endTime"] = query.EndTime?.ToString("O") ?? "null",
                ["serviceId"] = query.ServiceId ?? "null",
                ["minLevel"] = query.MinLevel?.ToString() ?? "null",
                ["correlationId"] = query.CorrelationId ?? "null",
                ["searchText"] = query.SearchText ?? "null",
                ["page"] = query.Page,
                ["pageSize"] = query.PageSize,
                ["resultCount"] = resultCount
            });

        await SaveAuditEntryAsync(auditEntry, ct);
    }

    public async Task LogExportAsync(
        string userId,
        LogQuery query,
        string format,
        int exportedCount,
        CancellationToken ct = default)
    {
        var auditEntry = CreateAuditEntry(
            userId,
            "EXPORT",
            new Dictionary<string, object>
            {
                ["format"] = format,
                ["startTime"] = query.StartTime?.ToString("O") ?? "null",
                ["endTime"] = query.EndTime?.ToString("O") ?? "null",
                ["serviceId"] = query.ServiceId ?? "null",
                ["exportedCount"] = exportedCount
            });

        await SaveAuditEntryAsync(auditEntry, ct);
    }

    public async Task LogAccessAsync(
        string userId,
        string logId,
        CancellationToken ct = default)
    {
        var auditEntry = CreateAuditEntry(
            userId,
            "ACCESS",
            new Dictionary<string, object>
            {
                ["logId"] = logId
            });

        await SaveAuditEntryAsync(auditEntry, ct);
    }

    private LogEntry CreateAuditEntry(
        string userId,
        string operation,
        Dictionary<string, object> metadata)
    {
        return new LogEntry
        {
            Id = Guid.NewGuid().ToString(),
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = AuditServiceId,
            Level = LogLevel.Info,
            Message = $"Audit: {operation} by {userId}",
            UserId = userId,
            Metadata = metadata
        };
    }

    private async Task SaveAuditEntryAsync(LogEntry entry, CancellationToken ct)
    {
        try
        {
            await _repository.SaveAsync(entry, ct);
            _logger.LogDebug("Audit entry saved: {Operation} by {UserId}", 
                entry.Metadata?["operation"] ?? "unknown", entry.UserId);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to save audit entry");
        }
    }
}
