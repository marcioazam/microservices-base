using Elastic.Clients.Elasticsearch;
using LoggingService.Core.Observability;
using Microsoft.Extensions.Diagnostics.HealthChecks;

namespace LoggingService.Api.Extensions;

/// <summary>
/// Health check for Elasticsearch connectivity.
/// </summary>
public sealed class ElasticsearchHealthCheck : IHealthCheck
{
    private readonly ElasticsearchClient _client;
    private readonly ILogger<ElasticsearchHealthCheck> _logger;

    public ElasticsearchHealthCheck(
        ElasticsearchClient client,
        ILogger<ElasticsearchHealthCheck> logger)
    {
        _client = client;
        _logger = logger;
    }

    public async Task<HealthCheckResult> CheckHealthAsync(
        HealthCheckContext context,
        CancellationToken cancellationToken = default)
    {
        try
        {
            var response = await _client.PingAsync(cancellationToken);

            if (response.IsValidResponse)
            {
                LoggingMetrics.ServiceHealth.WithLabels("elasticsearch").Set(1);
                return HealthCheckResult.Healthy("Elasticsearch is reachable");
            }

            LoggingMetrics.ServiceHealth.WithLabels("elasticsearch").Set(0);
            return HealthCheckResult.Unhealthy(
                $"Elasticsearch ping failed: {response.DebugInformation}");
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Elasticsearch health check failed");
            LoggingMetrics.ServiceHealth.WithLabels("elasticsearch").Set(0);
            return HealthCheckResult.Unhealthy("Elasticsearch is unreachable", ex);
        }
    }
}
