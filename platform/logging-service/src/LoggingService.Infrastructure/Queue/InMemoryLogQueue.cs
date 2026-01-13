using System.Collections.Concurrent;
using LoggingService.Core.Configuration;
using LoggingService.Core.Interfaces;
using LoggingService.Core.Models;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using Prometheus;

namespace LoggingService.Infrastructure.Queue;

/// <summary>
/// In-memory implementation of the log queue for testing and development.
/// </summary>
public sealed class InMemoryLogQueue : ILogQueue
{
    private readonly ConcurrentQueue<LogEntry> _queue = new();
    private readonly QueueOptions _options;
    private readonly ILogger<InMemoryLogQueue> _logger;
    private int _count;

    private static readonly Gauge QueueDepthGauge = Metrics.CreateGauge(
        "logging_queue_depth",
        "Current number of logs in the processing queue");

    private static readonly Gauge QueueCapacityGauge = Metrics.CreateGauge(
        "logging_queue_capacity_percent",
        "Current queue capacity percentage");

    private static readonly Counter QueueWarningCounter = Metrics.CreateCounter(
        "logging_queue_warning_total",
        "Number of times queue reached warning threshold");

    public InMemoryLogQueue(IOptions<QueueOptions> options, ILogger<InMemoryLogQueue> logger)
    {
        _options = options.Value;
        _logger = logger;
    }

    /// <inheritdoc />
    public Task EnqueueAsync(LogEntry entry, CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();

        if (IsFull())
        {
            throw new InvalidOperationException("Queue is full");
        }

        _queue.Enqueue(entry);
        Interlocked.Increment(ref _count);

        CheckQueueThreshold();
        UpdateMetrics();

        return Task.CompletedTask;
    }

    /// <inheritdoc />
    public Task EnqueueBatchAsync(IEnumerable<LogEntry> entries, CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();

        foreach (var entry in entries)
        {
            if (IsFull())
            {
                throw new InvalidOperationException("Queue is full");
            }

            _queue.Enqueue(entry);
            Interlocked.Increment(ref _count);
        }

        CheckQueueThreshold();
        UpdateMetrics();

        return Task.CompletedTask;
    }

    /// <inheritdoc />
    public Task<LogEntry?> DequeueAsync(CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();

        if (_queue.TryDequeue(out var entry))
        {
            Interlocked.Decrement(ref _count);
            UpdateMetrics();
            return Task.FromResult<LogEntry?>(entry);
        }

        return Task.FromResult<LogEntry?>(null);
    }

    /// <inheritdoc />
    public int GetQueueDepth() => _count;

    /// <inheritdoc />
    public bool IsFull() => _count >= _options.MaxQueueSize;

    /// <inheritdoc />
    public double GetCapacityPercentage() => (double)_count / _options.MaxQueueSize * 100;

    private void CheckQueueThreshold()
    {
        var capacityPercent = GetCapacityPercentage();

        if (capacityPercent >= _options.WarningThresholdPercent)
        {
            QueueWarningCounter.Inc();
            _logger.LogWarning(
                "Queue capacity at {Capacity:F1}% ({Count}/{Max})",
                capacityPercent,
                _count,
                _options.MaxQueueSize);
        }
    }

    private void UpdateMetrics()
    {
        QueueDepthGauge.Set(_count);
        QueueCapacityGauge.Set(GetCapacityPercentage());
    }
}
