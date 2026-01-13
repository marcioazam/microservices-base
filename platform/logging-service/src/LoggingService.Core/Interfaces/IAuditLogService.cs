using LoggingService.Core.Models;

namespace LoggingService.Core.Interfaces;

/// <summary>
/// Service for logging audit events.
/// </summary>
public interface IAuditLogService
{
    /// <summary>
    /// Logs a query operation for audit purposes.
    /// </summary>
    /// <param name="userId">The user who executed the query.</param>
    /// <param name="query">The query that was executed.</param>
    /// <param name="resultCount">Number of results returned.</param>
    /// <param name="ct">Cancellation token.</param>
    Task LogQueryAsync(string userId, LogQuery query, int resultCount, CancellationToken ct = default);

    /// <summary>
    /// Logs an export operation for audit purposes.
    /// </summary>
    /// <param name="userId">The user who executed the export.</param>
    /// <param name="query">The query used for export.</param>
    /// <param name="format">Export format (JSON, CSV).</param>
    /// <param name="resultCount">Number of results exported.</param>
    /// <param name="ct">Cancellation token.</param>
    Task LogExportAsync(string userId, LogQuery query, string format, int resultCount, CancellationToken ct = default);

    /// <summary>
    /// Logs an access operation for audit purposes.
    /// </summary>
    /// <param name="userId">The user who accessed the log.</param>
    /// <param name="logId">The ID of the accessed log.</param>
    /// <param name="ct">Cancellation token.</param>
    Task LogAccessAsync(string userId, string logId, CancellationToken ct = default);
}
