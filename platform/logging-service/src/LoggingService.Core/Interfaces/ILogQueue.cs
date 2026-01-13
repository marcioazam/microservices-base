using LoggingService.Core.Models;

namespace LoggingService.Core.Interfaces;

/// <summary>
/// Queue for asynchronous log processing.
/// </summary>
public interface ILogQueue
{
    /// <summary>
    /// Enqueues a single log entry for processing.
    /// </summary>
    /// <param name="entry">The log entry to enqueue.</param>
    /// <param name="ct">Cancellation token.</param>
    Task EnqueueAsync(LogEntry entry, CancellationToken ct = default);

    /// <summary>
    /// Enqueues a batch of log entries for processing.
    /// </summary>
    /// <param name="entries">The log entries to enqueue.</param>
    /// <param name="ct">Cancellation token.</param>
    Task EnqueueBatchAsync(IEnumerable<LogEntry> entries, CancellationToken ct = default);

    /// <summary>
    /// Dequeues a log entry for processing.
    /// </summary>
    /// <param name="ct">Cancellation token.</param>
    /// <returns>The next log entry, or null if queue is empty.</returns>
    Task<LogEntry?> DequeueAsync(CancellationToken ct = default);

    /// <summary>
    /// Gets the current depth of the queue.
    /// </summary>
    /// <returns>Number of entries in the queue.</returns>
    int GetQueueDepth();

    /// <summary>
    /// Checks if the queue is full.
    /// </summary>
    /// <returns>True if queue is at capacity.</returns>
    bool IsFull();

    /// <summary>
    /// Gets the queue capacity percentage (0-100).
    /// </summary>
    /// <returns>Current capacity percentage.</returns>
    double GetCapacityPercentage();
}
