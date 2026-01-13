using LoggingService.Core.Configuration;
using LoggingService.Core.Interfaces;
using LoggingService.Core.Observability;
using Microsoft.Extensions.Diagnostics.HealthChecks;
using Microsoft.Extensions.Options;

namespace LoggingService.Api.Extensions;

/// <summary>
/// Health check for RabbitMQ connectivity and queue status.
/// </summary>
public sealed class RabbitMqHealthCheck : IHealthCheck
{
    private readonly ILogQueue _queue;
    private readonly QueueOptions _options;
    private readonly ILogger<RabbitMqHealthCheck> _logger;

    public RabbitMqHealthCheck(
        ILogQueue queue,
        IOptions<QueueOptions> options,
        ILogger<RabbitMqHealthCheck> logger)
    {
        _queue = queue;
        _options = options.Value;
        _logger = logger;
    }

    public Task<HealthCheckResult> CheckHealthAsync(
        HealthCheckContext context,
        CancellationToken cancellationToken = default)
    {
        try
        {
            var depth = _queue.GetQueueDepth();
            var capacityPercent = _queue.GetCapacityPercentage();
            var warningThreshold = _options.WarningThresholdPercent;

            var data = new Dictionary<string, object>
            {
                ["queueDepth"] = depth,
                ["capacityPercent"] = capacityPercent,
                ["maxQueueSize"] = _options.MaxQueueSize
            };

            if (_queue.IsFull())
            {
                LoggingMetrics.ServiceHealth.WithLabels("rabbitmq").Set(0);
                return Task.FromResult(HealthCheckResult.Unhealthy(
                    "Queue is full - backpressure active", data: data));
            }

            if (capacityPercent >= warningThreshold)
            {
                LoggingMetrics.ServiceHealth.WithLabels("rabbitmq").Set(0.5);
                return Task.FromResult(HealthCheckResult.Degraded(
                    $"Queue at {capacityPercent:F1}% capacity", data: data));
            }

            LoggingMetrics.ServiceHealth.WithLabels("rabbitmq").Set(1);
            return Task.FromResult(HealthCheckResult.Healthy(
                "Queue is healthy", data: data));
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "RabbitMQ health check failed");
            LoggingMetrics.ServiceHealth.WithLabels("rabbitmq").Set(0);
            return Task.FromResult(HealthCheckResult.Unhealthy(
                "Failed to check queue status", ex));
        }
    }
}
