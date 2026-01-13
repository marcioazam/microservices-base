using Elastic.Clients.Elasticsearch;
using LoggingService.Core.Configuration;
using LoggingService.Core.Interfaces;
using LoggingService.Infrastructure.Queue;
using LoggingService.Infrastructure.Storage;
using LoggingService.Worker;
using Microsoft.Extensions.Options;
using Prometheus;
using Serilog;

var builder = Host.CreateApplicationBuilder(args);

// Configure Serilog
Log.Logger = new LoggerConfiguration()
    .MinimumLevel.Information()
    .Enrich.FromLogContext()
    .WriteTo.Console()
    .CreateLogger();

builder.Logging.ClearProviders();
builder.Logging.AddSerilog();

// Configure options
builder.Services.Configure<LoggingServiceOptions>(
    builder.Configuration.GetSection(LoggingServiceOptions.SectionName));
builder.Services.Configure<QueueOptions>(
    builder.Configuration.GetSection(QueueOptions.SectionName));
builder.Services.Configure<ElasticSearchOptions>(
    builder.Configuration.GetSection(ElasticSearchOptions.SectionName));
builder.Services.Configure<ObservabilityOptions>(
    builder.Configuration.GetSection(ObservabilityOptions.SectionName));

// Configure Elasticsearch 8.x client
builder.Services.AddSingleton<ElasticsearchClient>(sp =>
{
    var options = sp.GetRequiredService<IOptions<ElasticSearchOptions>>().Value;
    return ElasticsearchClientFactory.Create(options);
});

// Register services
builder.Services.AddSingleton<ILogQueue, RabbitMqLogQueue>();
builder.Services.AddSingleton<ILogRepository, ElasticsearchLogRepository>();

// Register worker with graceful shutdown support
builder.Services.AddHostedService<LogProcessorWorker>();

// Configure Prometheus metrics
builder.Services.AddSingleton<MetricServer>(sp =>
{
    var server = new MetricServer(port: 9090);
    server.Start();
    return server;
});

// Configure graceful shutdown
builder.Services.Configure<HostOptions>(options =>
{
    options.ShutdownTimeout = TimeSpan.FromSeconds(30);
});

var host = builder.Build();

// Initialize RabbitMQ connection
var queue = host.Services.GetRequiredService<ILogQueue>();
if (queue is RabbitMqLogQueue rabbitQueue)
{
    await rabbitQueue.InitializeAsync();
}

try
{
    Log.Information("Starting Log Processor Worker on .NET 9");
    await host.RunAsync();
}
catch (Exception ex)
{
    Log.Fatal(ex, "Worker terminated unexpectedly");
}
finally
{
    // Cleanup
    if (queue is IAsyncDisposable asyncDisposable)
    {
        await asyncDisposable.DisposeAsync();
    }
    await Log.CloseAndFlushAsync();
}
