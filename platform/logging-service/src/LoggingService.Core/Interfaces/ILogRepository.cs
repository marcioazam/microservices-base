using LoggingService.Core.Models;

namespace LoggingService.Core.Interfaces;

/// <summary>
/// Repository for persisting and querying log entries.
/// </summary>
public interface ILogRepository
{
    /// <summary>
    /// Saves a single log entry to storage.
    /// </summary>
    /// <param name="entry">The log entry to save.</param>
    /// <param name="ct">Cancellation token.</param>
    /// <returns>The ID of the saved entry.</returns>
    Task<string> SaveAsync(LogEntry entry, CancellationToken ct = default);

    /// <summary>
    /// Saves a batch of log entries to storage.
    /// </summary>
    /// <param name="entries">The log entries to save.</param>
    /// <param name="ct">Cancellation token.</param>
    Task SaveBatchAsync(IEnumerable<LogEntry> entries, CancellationToken ct = default);

    /// <summary>
    /// Retrieves a log entry by its ID.
    /// </summary>
    /// <param name="id">The log entry ID.</param>
    /// <param name="ct">Cancellation token.</param>
    /// <returns>The log entry if found, null otherwise.</returns>
    Task<LogEntry?> GetByIdAsync(string id, CancellationToken ct = default);

    /// <summary>
    /// Queries log entries based on filter criteria.
    /// </summary>
    /// <param name="query">The query parameters.</param>
    /// <param name="ct">Cancellation token.</param>
    /// <returns>Paged result of matching log entries.</returns>
    Task<PagedResult<LogEntry>> QueryAsync(LogQuery query, CancellationToken ct = default);

    /// <summary>
    /// Deletes log entries older than the specified date.
    /// </summary>
    /// <param name="olderThan">Delete entries older than this date.</param>
    /// <param name="level">Optional log level filter.</param>
    /// <param name="ct">Cancellation token.</param>
    /// <returns>Number of deleted entries.</returns>
    Task<long> DeleteOlderThanAsync(DateTimeOffset olderThan, LogLevel? level = null, CancellationToken ct = default);

    /// <summary>
    /// Archives log entries older than the specified date to cold storage.
    /// </summary>
    /// <param name="olderThan">Archive entries older than this date.</param>
    /// <param name="ct">Cancellation token.</param>
    /// <returns>Number of archived entries.</returns>
    Task<long> ArchiveOlderThanAsync(DateTimeOffset olderThan, CancellationToken ct = default);
}
