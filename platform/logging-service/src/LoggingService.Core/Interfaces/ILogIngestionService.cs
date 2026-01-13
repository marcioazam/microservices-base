using LoggingService.Core.Models;

namespace LoggingService.Core.Interfaces;

/// <summary>
/// Service for ingesting log entries into the logging system.
/// </summary>
public interface ILogIngestionService
{
    /// <summary>
    /// Ingests a single log entry.
    /// </summary>
    /// <param name="entry">The log entry to ingest.</param>
    /// <param name="ct">Cancellation token.</param>
    /// <returns>Result of the ingestion operation.</returns>
    Task<IngestResult> IngestAsync(LogEntry entry, CancellationToken ct = default);

    /// <summary>
    /// Ingests a batch of log entries.
    /// </summary>
    /// <param name="entries">The log entries to ingest.</param>
    /// <param name="ct">Cancellation token.</param>
    /// <returns>Result of the batch ingestion operation.</returns>
    Task<BatchIngestResult> IngestBatchAsync(IEnumerable<LogEntry> entries, CancellationToken ct = default);
}
