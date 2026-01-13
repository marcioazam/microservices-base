using LoggingService.Core.Configuration;
using LoggingService.Core.Interfaces;
using LoggingService.Core.Models;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using LogLevel = LoggingService.Core.Models.LogLevel;

namespace LoggingService.Core.Services;

/// <summary>
/// Service for applying log retention policies.
/// </summary>
public sealed class RetentionService : IRetentionService
{
    private readonly ILogRepository _repository;
    private readonly LoggingServiceOptions _options;
    private readonly ILogger<RetentionService> _logger;

    public RetentionService(
        ILogRepository repository,
        IOptions<LoggingServiceOptions> options,
        ILogger<RetentionService> logger)
    {
        _repository = repository;
        _options = options.Value;
        _logger = logger;
    }

    public async Task ApplyRetentionPoliciesAsync(CancellationToken ct = default)
    {
        _logger.LogInformation("Starting retention policy enforcement");

        var retention = _options.Retention;
        var now = DateTimeOffset.UtcNow;

        var policies = new Dictionary<LogLevel, TimeSpan>
        {
            [LogLevel.Debug] = retention?.DebugRetention ?? TimeSpan.FromDays(7),
            [LogLevel.Info] = retention?.InfoRetention ?? TimeSpan.FromDays(30),
            [LogLevel.Warn] = retention?.WarnRetention ?? TimeSpan.FromDays(90),
            [LogLevel.Error] = retention?.ErrorRetention ?? TimeSpan.FromDays(365),
            [LogLevel.Fatal] = retention?.FatalRetention ?? TimeSpan.FromDays(365)
        };

        foreach (var (level, retentionPeriod) in policies)
        {
            var cutoffDate = now - retentionPeriod;

            try
            {
                if (retention?.ArchiveBeforeDelete == true)
                {
                    await ArchiveLogsAsync(cutoffDate, ct);
                }

                var deletedCount = await _repository.DeleteOlderThanAsync(cutoffDate, level, ct);
                _logger.LogInformation(
                    "Deleted {Count} {Level} logs older than {CutoffDate}",
                    deletedCount, level, cutoffDate);
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Failed to apply retention for {Level}", level);
            }
        }

        _logger.LogInformation("Retention policy enforcement completed");
    }

    public async Task ArchiveLogsAsync(DateTimeOffset olderThan, CancellationToken ct = default)
    {
        _logger.LogInformation("Archiving logs older than {OlderThan}", olderThan);

        try
        {
            var archivedCount = await _repository.ArchiveOlderThanAsync(olderThan, ct);
            _logger.LogInformation("Archived {Count} logs to cold storage", archivedCount);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to archive logs older than {OlderThan}", olderThan);
            throw;
        }
    }

    public TimeSpan GetRetentionPeriod(LogLevel level)
    {
        var retention = _options.Retention;
        return level switch
        {
            LogLevel.Debug => retention?.DebugRetention ?? TimeSpan.FromDays(7),
            LogLevel.Info => retention?.InfoRetention ?? TimeSpan.FromDays(30),
            LogLevel.Warn => retention?.WarnRetention ?? TimeSpan.FromDays(90),
            LogLevel.Error => retention?.ErrorRetention ?? TimeSpan.FromDays(365),
            LogLevel.Fatal => retention?.FatalRetention ?? TimeSpan.FromDays(365),
            _ => TimeSpan.FromDays(30)
        };
    }
}
