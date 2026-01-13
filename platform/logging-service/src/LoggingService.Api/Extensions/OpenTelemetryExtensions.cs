using LoggingService.Core.Configuration;
using OpenTelemetry.Metrics;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;

namespace LoggingService.Api.Extensions;

/// <summary>
/// OpenTelemetry 1.10+ configuration extensions.
/// </summary>
public static class OpenTelemetryExtensions
{
    public static IServiceCollection AddLoggingServiceTelemetry(
        this IServiceCollection services,
        IConfiguration configuration)
    {
        var observabilityOptions = configuration
            .GetSection(ObservabilityOptions.SectionName)
            .Get<ObservabilityOptions>() ?? new ObservabilityOptions();

        var environment = configuration["Environment"] ?? "development";

        services.AddOpenTelemetry()
            .ConfigureResource(resource => resource
                .AddService(
                    serviceName: observabilityOptions.ServiceName,
                    serviceVersion: observabilityOptions.ServiceVersion)
                .AddAttributes(new Dictionary<string, object>
                {
                    ["deployment.environment"] = environment,
                    ["service.namespace"] = "logging-platform"
                }))
            .WithTracing(tracing =>
            {
                if (!observabilityOptions.EnableTracing) return;

                tracing
                    .SetSampler(new TraceIdRatioBasedSampler(observabilityOptions.TraceSamplingRatio))
                    .AddAspNetCoreInstrumentation(options =>
                    {
                        options.RecordException = true;
                        options.Filter = ctx => !ctx.Request.Path.StartsWithSegments("/health")
                                              && !ctx.Request.Path.StartsWithSegments("/metrics");
                    })
                    .AddHttpClientInstrumentation(options =>
                    {
                        options.RecordException = true;
                    })
                    .AddSource("LoggingService")
                    .AddSource("LoggingService.Ingestion")
                    .AddSource("LoggingService.Storage")
                    .AddSource("LoggingService.Queue")
                    .AddOtlpExporter(options =>
                    {
                        options.Endpoint = new Uri(observabilityOptions.OtlpEndpoint);
                    });
            })
            .WithMetrics(metrics =>
            {
                if (!observabilityOptions.EnableMetrics) return;

                metrics
                    .AddAspNetCoreInstrumentation()
                    .AddHttpClientInstrumentation()
                    .AddRuntimeInstrumentation()
                    .AddMeter("LoggingService")
                    .AddPrometheusExporter();
            });

        return services;
    }
}
