using Elastic.Clients.Elasticsearch;
using LoggingService.Core.Configuration;
using LoggingService.Core.Interfaces;
using LoggingService.Core.Observability;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Extensions.Options;

namespace LoggingService.Api.Controllers;

/// <summary>
/// Health check endpoints for Kubernetes probes.
/// </summary>
[ApiController]
[Route("health")]
public sealed class HealthController : ControllerBase
{
    private readonly ILogQueue _queue;
    private readonly ElasticsearchClient _elasticClient;
    private readonly LoggingServiceOptions _options;
    private readonly ILogger<HealthController> _logger;

    public HealthController(
        ILogQueue queue,
        ElasticsearchClient elasticClient,
        IOptions<LoggingServiceOptions> options,
        ILogger<HealthController> logger)
    {
        _queue = queue;
        _elasticClient = elasticClient;
        _options = options.Value;
        _logger = logger;
    }

    /// <summary>
    /// Liveness probe - checks if the service is running.
    /// </summary>
    [HttpGet("live")]
    [ProducesResponseType(StatusCodes.Status200OK)]
    public IActionResult LivenessProbe()
    {
        LoggingMetrics.ServiceHealth.WithLabels("api").Set(1);
        return Ok(new { status = "healthy", timestamp = DateTimeOffset.UtcNow });
    }

    /// <summary>
    /// Readiness probe - checks if the service can handle requests.
    /// </summary>
    [HttpGet("ready")]
    [ProducesResponseType(StatusCodes.Status200OK)]
    [ProducesResponseType(StatusCodes.Status503ServiceUnavailable)]
    public async Task<IActionResult> ReadinessProbe(CancellationToken ct)
    {
        var checks = new Dictionary<string, object>();

        // Check Elasticsearch
        try
        {
            var pingResponse = await _elasticClient.PingAsync(ct);
            var esHealthy = pingResponse.IsValidResponse;
            checks["elasticsearch"] = new { status = esHealthy ? "healthy" : "unhealthy" };
            LoggingMetrics.ServiceHealth.WithLabels("elasticsearch").Set(esHealthy ? 1 : 0);
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "Elasticsearch health check failed");
            checks["elasticsearch"] = new { status = "unhealthy", error = ex.Message };
            LoggingMetrics.ServiceHealth.WithLabels("elasticsearch").Set(0);
        }

        // Check Queue
        try
        {
            var depth = _queue.GetQueueDepth();
            var maxSize = _options.Queue.MaxQueueSize;
            var capacityPercent = _queue.GetCapacityPercentage();
            var queueHealthy = !_queue.IsFull();
            var queueStatus = queueHealthy ? "healthy" : "degraded";

            checks["queue"] = new
            {
                status = queueStatus,
                depth,
                maxSize,
                capacityPercent = $"{capacityPercent:F1}%"
            };

            LoggingMetrics.ServiceHealth.WithLabels("queue").Set(queueHealthy ? 1 : 0);
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "Queue health check failed");
            checks["queue"] = new { status = "unhealthy", error = ex.Message };
            LoggingMetrics.ServiceHealth.WithLabels("queue").Set(0);
        }

        var allHealthy = checks.Values
            .Cast<dynamic>()
            .All(c => c.status == "healthy");

        var statusCode = allHealthy
            ? StatusCodes.Status200OK
            : StatusCodes.Status503ServiceUnavailable;

        return StatusCode(statusCode, new
        {
            status = allHealthy ? "healthy" : "unhealthy",
            checks,
            timestamp = DateTimeOffset.UtcNow
        });
    }
}
