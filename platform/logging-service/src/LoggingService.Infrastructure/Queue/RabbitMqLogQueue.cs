using System.Text.Json;
using LoggingService.Core.Configuration;
using LoggingService.Core.Interfaces;
using LoggingService.Core.Models;
using LoggingService.Core.Observability;
using LoggingService.Core.Serialization;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using RabbitMQ.Client;

namespace LoggingService.Infrastructure.Queue;

/// <summary>
/// RabbitMQ 7.x implementation of the log queue with async-first API.
/// </summary>
public sealed class RabbitMqLogQueue : ILogQueue, IAsyncDisposable
{
    private readonly QueueOptions _options;
    private readonly ILogger<RabbitMqLogQueue> _logger;
    private IConnection? _connection;
    private IChannel? _channel;
    private bool _initialized;
    private readonly SemaphoreSlim _initLock = new(1, 1);

    public RabbitMqLogQueue(
        IOptions<QueueOptions> options,
        ILogger<RabbitMqLogQueue> logger)
    {
        _options = options.Value;
        _logger = logger;
    }

    /// <summary>
    /// Initializes the RabbitMQ connection and channel asynchronously.
    /// </summary>
    public async Task InitializeAsync(CancellationToken ct = default)
    {
        if (_initialized) return;

        await _initLock.WaitAsync(ct);
        try
        {
            if (_initialized) return;

            var factory = new ConnectionFactory
            {
                Uri = new Uri(_options.ConnectionString),
                AutomaticRecoveryEnabled = true,
                NetworkRecoveryInterval = TimeSpan.FromSeconds(10),
                TopologyRecoveryEnabled = true
            };

            _connection = await factory.CreateConnectionAsync(ct);
            _channel = await _connection.CreateChannelAsync(cancellationToken: ct);

            await _channel.QueueDeclareAsync(
                queue: _options.QueueName,
                durable: true,
                exclusive: false,
                autoDelete: false,
                arguments: new Dictionary<string, object?>
                {
                    ["x-max-length"] = _options.MaxQueueSize
                },
                cancellationToken: ct);

            _initialized = true;
            _logger.LogInformation("RabbitMQ queue initialized: {QueueName}", _options.QueueName);
        }
        finally
        {
            _initLock.Release();
        }
    }

    private async Task EnsureInitializedAsync(CancellationToken ct)
    {
        if (!_initialized)
        {
            await InitializeAsync(ct);
        }
    }

    /// <inheritdoc />
    public async Task EnqueueAsync(LogEntry entry, CancellationToken ct = default)
    {
        await EnsureInitializedAsync(ct);
        ct.ThrowIfCancellationRequested();

        var body = LogEntrySerializer.SerializeToUtf8Bytes(entry);

        var properties = new BasicProperties
        {
            Persistent = true,
            ContentType = "application/json",
            MessageId = entry.Id,
            CorrelationId = entry.CorrelationId,
            Timestamp = new AmqpTimestamp(entry.Timestamp.ToUnixTimeSeconds())
        };

        await _channel!.BasicPublishAsync(
            exchange: string.Empty,
            routingKey: _options.QueueName,
            mandatory: false,
            basicProperties: properties,
            body: body,
            cancellationToken: ct);

        LoggingMetrics.QueueEnqueued.Inc();
        UpdateQueueMetrics();
    }

    /// <inheritdoc />
    public async Task EnqueueBatchAsync(IEnumerable<LogEntry> entries, CancellationToken ct = default)
    {
        await EnsureInitializedAsync(ct);
        ct.ThrowIfCancellationRequested();

        var count = 0;
        foreach (var entry in entries)
        {
            var body = LogEntrySerializer.SerializeToUtf8Bytes(entry);

            var properties = new BasicProperties
            {
                Persistent = true,
                ContentType = "application/json",
                MessageId = entry.Id,
                CorrelationId = entry.CorrelationId,
                Timestamp = new AmqpTimestamp(entry.Timestamp.ToUnixTimeSeconds())
            };

            await _channel!.BasicPublishAsync(
                exchange: string.Empty,
                routingKey: _options.QueueName,
                mandatory: false,
                basicProperties: properties,
                body: body,
                cancellationToken: ct);

            count++;
        }

        LoggingMetrics.QueueEnqueued.Inc(count);
        UpdateQueueMetrics();

        _logger.LogDebug("Enqueued batch of {Count} entries", count);
    }

    /// <inheritdoc />
    public async Task<LogEntry?> DequeueAsync(CancellationToken ct = default)
    {
        await EnsureInitializedAsync(ct);
        ct.ThrowIfCancellationRequested();

        var result = await _channel!.BasicGetAsync(_options.QueueName, autoAck: false, ct);

        if (result == null)
        {
            return null;
        }

        try
        {
            var entry = LogEntrySerializer.Deserialize(result.Body.Span);

            if (entry != null)
            {
                await _channel.BasicAckAsync(result.DeliveryTag, multiple: false, ct);
                LoggingMetrics.QueueDequeued.Inc();
                UpdateQueueMetrics();
            }
            else
            {
                await _channel.BasicRejectAsync(result.DeliveryTag, requeue: false, ct);
                _logger.LogWarning("Failed to deserialize message, rejecting");
            }

            return entry;
        }
        catch (JsonException ex)
        {
            _logger.LogError(ex, "Failed to deserialize queue message");
            await _channel.BasicRejectAsync(result.DeliveryTag, requeue: false, ct);
            return null;
        }
    }

    /// <inheritdoc />
    public int GetQueueDepth()
    {
        if (!_initialized || _channel == null)
        {
            return 0;
        }

        try
        {
            // Note: In RabbitMQ 7.x, we need to use a different approach
            // For now, return 0 as passive declare is synchronous
            // In production, use management API or consumer-based tracking
            return 0;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to get queue depth");
            return 0;
        }
    }

    /// <inheritdoc />
    public bool IsFull()
    {
        var depth = GetQueueDepth();
        return depth >= _options.MaxQueueSize;
    }

    /// <inheritdoc />
    public double GetCapacityPercentage()
    {
        var depth = GetQueueDepth();
        return (double)depth / _options.MaxQueueSize * 100;
    }

    private void UpdateQueueMetrics()
    {
        var depth = GetQueueDepth();
        LoggingMetrics.QueueDepth.Set(depth);
        LoggingMetrics.QueueCapacity.Set(GetCapacityPercentage());
    }

    /// <inheritdoc />
    public async ValueTask DisposeAsync()
    {
        if (_channel != null)
        {
            await _channel.CloseAsync();
            await _channel.DisposeAsync();
        }

        if (_connection != null)
        {
            await _connection.CloseAsync();
            await _connection.DisposeAsync();
        }

        _initLock.Dispose();
    }
}
