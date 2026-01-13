using LoggingService.Core.Interfaces;
using LoggingService.Core.Models;
using LoggingService.Core.Observability;
using Microsoft.Extensions.Logging;

namespace LoggingService.Core.Services;

/// <summary>
/// Service for ingesting log entries into the logging system.
/// </summary>
public sealed class LogIngestionService : ILogIngestionService
{
    private readonly ILogQueue _queue;
    private readonly ILogEntryValidator _validator;
    private readonly ILogEntryEnricher _enricher;
    private readonly IPiiMasker _piiMasker;
    private readonly ILogger<LogIngestionService> _logger;

    private const int MaxBatchSize = 1000;

    public LogIngestionService(
        ILogQueue queue,
        ILogEntryValidator validator,
        ILogEntryEnricher enricher,
        IPiiMasker piiMasker,
        ILogger<LogIngestionService> logger)
    {
        _queue = queue;
        _validator = validator;
        _enricher = enricher;
        _piiMasker = piiMasker;
        _logger = logger;
    }

    /// <inheritdoc />
    public async Task<IngestResult> IngestAsync(LogEntry entry, CancellationToken ct = default)
    {
        var stopwatch = System.Diagnostics.Stopwatch.StartNew();
        try
        {
            // Check queue capacity first
            if (_queue.IsFull())
            {
                _logger.LogWarning("Queue is full, applying backpressure");
                LoggingMetrics.LogsRejected.WithLabels("queue_full").Inc();
                return IngestResult.QueueFull();
            }

            // 1. Validate entry
            var validationResult = _validator.Validate(entry);
            if (!validationResult.IsValid)
            {
                _logger.LogDebug("Log entry validation failed: {Errors}",
                    string.Join(", ", validationResult.Errors.Select(e => e.Message)));
                LoggingMetrics.LogsRejected.WithLabels("validation_failed").Inc();
                return IngestResult.ValidationFailed(validationResult.Errors);
            }

            // 2. Enrich (add correlation_id if missing, normalize timestamp)
            var enrichedEntry = _enricher.Enrich(entry);

            // 3. Mask PII
            var maskedEntry = _piiMasker.MaskSensitiveData(enrichedEntry);

            // 4. Enqueue for async processing
            await _queue.EnqueueAsync(maskedEntry, ct);

            LoggingMetrics.LogsReceived.WithLabels(maskedEntry.ServiceId, maskedEntry.Level.ToString()).Inc();
            _logger.LogDebug("Log entry ingested: {Id}", maskedEntry.Id);

            return IngestResult.Success(maskedEntry.Id);
        }
        finally
        {
            stopwatch.Stop();
            LoggingMetrics.IngestLatency.Observe(stopwatch.Elapsed.TotalSeconds);
        }
    }

    /// <inheritdoc />
    public async Task<BatchIngestResult> IngestBatchAsync(
        IEnumerable<LogEntry> entries,
        CancellationToken ct = default)
    {
        var entriesList = entries.ToList();

        // Check batch size limit
        if (entriesList.Count > MaxBatchSize)
        {
            _logger.LogWarning("Batch size {Size} exceeds maximum {Max}",
                entriesList.Count, MaxBatchSize);
            LoggingMetrics.LogsRejected.WithLabels("batch_too_large").Inc(entriesList.Count);
            return BatchIngestResult.BatchTooLarge(entriesList.Count, MaxBatchSize);
        }

        // Check queue capacity
        if (_queue.IsFull())
        {
            _logger.LogWarning("Queue is full, rejecting batch");
            LoggingMetrics.LogsRejected.WithLabels("queue_full").Inc(entriesList.Count);
            return new BatchIngestResult(
                entriesList.Select(_ => IngestResult.QueueFull()).ToList(),
                0,
                entriesList.Count);
        }

        var results = new List<IngestResult>();
        var validEntries = new List<LogEntry>();

        foreach (var entry in entriesList)
        {
            // Validate
            var validationResult = _validator.Validate(entry);
            if (!validationResult.IsValid)
            {
                results.Add(IngestResult.ValidationFailed(validationResult.Errors));
                LoggingMetrics.LogsRejected.WithLabels("validation_failed").Inc();
                continue;
            }

            // Enrich and mask
            var enriched = _enricher.Enrich(entry);
            var masked = _piiMasker.MaskSensitiveData(enriched);

            validEntries.Add(masked);
            results.Add(IngestResult.Success(masked.Id));

            LoggingMetrics.LogsReceived.WithLabels(masked.ServiceId, masked.Level.ToString()).Inc();
        }

        // Enqueue valid entries
        if (validEntries.Count > 0)
        {
            await _queue.EnqueueBatchAsync(validEntries, ct);
            _logger.LogDebug("Batch of {Count} entries ingested", validEntries.Count);
        }

        return new BatchIngestResult(results, validEntries.Count, entriesList.Count - validEntries.Count);
    }
}
