using LoggingService.Core.Configuration;
using LoggingService.Core.Interfaces;
using LoggingService.Core.Models;
using LoggingService.Core.Observability;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;

namespace LoggingService.Worker;

/// <summary>
/// Background worker that processes logs from the queue and saves them to storage.
/// Supports graceful shutdown with queue draining.
/// </summary>
public sealed class LogProcessorWorker : BackgroundService
{
    private readonly ILogQueue _queue;
    private readonly ILogRepository _repository;
    private readonly QueueOptions _options;
    private readonly ILogger<LogProcessorWorker> _logger;
    private readonly IHostApplicationLifetime _lifetime;

    public LogProcessorWorker(
        ILogQueue queue,
        ILogRepository repository,
        IOptions<QueueOptions> options,
        ILogger<LogProcessorWorker> logger,
        IHostApplicationLifetime lifetime)
    {
        _queue = queue;
        _repository = repository;
        _options = options.Value;
        _logger = logger;
        _lifetime = lifetime;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        _logger.LogInformation("Log processor worker starting");

        // Register shutdown handler for graceful drain
        _lifetime.ApplicationStopping.Register(() =>
        {
            _logger.LogInformation("Shutdown requested, draining queue...");
        });

        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                var batch = await CollectBatchAsync(stoppingToken);

                if (batch.Count > 0)
                {
                    var stopwatch = System.Diagnostics.Stopwatch.StartNew();
                    try
                    {
                        await ProcessBatchAsync(batch, stoppingToken);
                    }
                    finally
                    {
                        stopwatch.Stop();
                        LoggingMetrics.ProcessingLatency.Observe(stopwatch.Elapsed.TotalSeconds);
                    }
                }
                else
                {
                    // No items in queue, wait before checking again
                    await Task.Delay(_options.ProcessingInterval, stoppingToken);
                }
            }
            catch (OperationCanceledException) when (stoppingToken.IsCancellationRequested)
            {
                // Graceful shutdown - drain remaining items
                await DrainQueueAsync();
                break;
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Error in log processor worker");
                LoggingMetrics.LogsFailed.WithLabels("worker_error").Inc();

                // Wait before retrying to avoid tight loop on persistent errors
                await Task.Delay(TimeSpan.FromSeconds(5), stoppingToken);
            }
        }

        _logger.LogInformation("Log processor worker stopped");
    }

    private async Task DrainQueueAsync()
    {
        _logger.LogInformation("Draining remaining queue items before shutdown");

        var drainedCount = 0;
        var maxDrainAttempts = 100; // Prevent infinite loop

        for (int attempt = 0; attempt < maxDrainAttempts; attempt++)
        {
            try
            {
                var batch = await CollectBatchAsync(CancellationToken.None);
                if (batch.Count == 0)
                {
                    break;
                }

                await ProcessBatchAsync(batch, CancellationToken.None);
                drainedCount += batch.Count;
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "Error during queue drain, stopping");
                break;
            }
        }

        _logger.LogInformation("Drained {Count} items from queue during shutdown", drainedCount);
    }

    private async Task<List<LogEntry>> CollectBatchAsync(CancellationToken ct)
    {
        var batch = new List<LogEntry>();

        for (int i = 0; i < _options.BatchSize; i++)
        {
            var entry = await _queue.DequeueAsync(ct);
            if (entry == null)
            {
                break;
            }
            batch.Add(entry);
        }

        return batch;
    }

    private async Task ProcessBatchAsync(List<LogEntry> batch, CancellationToken ct)
    {
        try
        {
            await _repository.SaveBatchAsync(batch, ct);
            LoggingMetrics.LogsProcessed.Inc(batch.Count);
            _logger.LogDebug("Processed batch of {Count} log entries", batch.Count);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to save batch of {Count} entries", batch.Count);
            LoggingMetrics.LogsFailed.WithLabels("storage_error").Inc(batch.Count);
            throw;
        }
    }
}
