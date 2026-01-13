using LoggingService.Core.Interfaces;
using LoggingService.Core.Models;

namespace LoggingService.Core.Services;

/// <summary>
/// Enriches log entries with additional data like correlation IDs and normalized timestamps.
/// </summary>
public sealed class LogEntryEnricher : ILogEntryEnricher
{
    /// <inheritdoc />
    public LogEntry Enrich(LogEntry entry)
    {
        var enrichedEntry = entry;

        // Generate ID if not provided
        if (string.IsNullOrWhiteSpace(entry.Id))
        {
            enrichedEntry = enrichedEntry with { Id = Guid.NewGuid().ToString() };
        }

        // Generate correlation ID if not provided
        if (string.IsNullOrWhiteSpace(entry.CorrelationId))
        {
            enrichedEntry = enrichedEntry with { CorrelationId = Guid.NewGuid().ToString() };
        }

        // Normalize timestamp to UTC
        if (entry.Timestamp.Offset != TimeSpan.Zero)
        {
            enrichedEntry = enrichedEntry with { Timestamp = entry.Timestamp.ToUniversalTime() };
        }

        return enrichedEntry;
    }
}
