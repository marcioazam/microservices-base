using LoggingService.Core.Models;

namespace LoggingService.Core.Interfaces;

/// <summary>
/// Enriches log entries with additional data.
/// </summary>
public interface ILogEntryEnricher
{
    /// <summary>
    /// Enriches a log entry with additional data (e.g., correlation ID, normalized timestamp).
    /// </summary>
    /// <param name="entry">The log entry to enrich.</param>
    /// <returns>The enriched log entry.</returns>
    LogEntry Enrich(LogEntry entry);
}
