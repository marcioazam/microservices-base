namespace LoggingService.Core.Interfaces;

/// <summary>
/// Service for managing log retention policies.
/// </summary>
public interface IRetentionService
{
    /// <summary>
    /// Applies retention policies to delete old logs.
    /// </summary>
    /// <param name="ct">Cancellation token.</param>
    Task ApplyRetentionPoliciesAsync(CancellationToken ct = default);

    /// <summary>
    /// Archives logs older than the specified date to cold storage.
    /// </summary>
    /// <param name="olderThan">Archive logs older than this date.</param>
    /// <param name="ct">Cancellation token.</param>
    Task ArchiveLogsAsync(DateTimeOffset olderThan, CancellationToken ct = default);
}
