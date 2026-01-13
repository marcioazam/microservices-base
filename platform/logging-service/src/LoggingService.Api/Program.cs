using Elastic.Clients.Elasticsearch;
using LoggingService.Api.Extensions;
using LoggingService.Api.Services;
using LoggingService.Core.Configuration;
using LoggingService.Core.Interfaces;
using LoggingService.Core.Services;
using LoggingService.Infrastructure.Queue;
using LoggingService.Infrastructure.Storage;
using Prometheus;
using Serilog;

var builder = WebApplication.CreateBuilder(args);

// Configure Serilog
Log.Logger = new LoggerConfiguration()
    .ReadFrom.Configuration(builder.Configuration)
    .Enrich.FromLogContext()
    .WriteTo.Console()
    .CreateLogger();

builder.Host.UseSerilog();

// Configuration - centralized options binding
builder.Services.Configure<LoggingServiceOptions>(
    builder.Configuration.GetSection(LoggingServiceOptions.SectionName));
builder.Services.Configure<ElasticSearchOptions>(
    builder.Configuration.GetSection(ElasticSearchOptions.SectionName));
builder.Services.Configure<QueueOptions>(
    builder.Configuration.GetSection(QueueOptions.SectionName));
builder.Services.Configure<SecurityOptions>(
    builder.Configuration.GetSection(SecurityOptions.SectionName));
builder.Services.Configure<RetentionOptions>(
    builder.Configuration.GetSection(RetentionOptions.SectionName));
builder.Services.Configure<ObservabilityOptions>(
    builder.Configuration.GetSection(ObservabilityOptions.SectionName));

// Elasticsearch 8.x client
var esOptions = builder.Configuration
    .GetSection(ElasticSearchOptions.SectionName)
    .Get<ElasticSearchOptions>() ?? new ElasticSearchOptions();
builder.Services.AddSingleton(ElasticsearchClientFactory.Create(esOptions));

// Core services
builder.Services.AddSingleton<ILogEntryValidator, LogEntryValidator>();
builder.Services.AddSingleton<ILogEntryEnricher, LogEntryEnricher>();
builder.Services.AddSingleton<IPiiMasker, PiiMasker>();
builder.Services.AddSingleton<ILogQueue, RabbitMqLogQueue>();
builder.Services.AddSingleton<ILogRepository, ElasticsearchLogRepository>();
builder.Services.AddSingleton<ILogIngestionService, LogIngestionService>();
builder.Services.AddSingleton<IAuditLogService, AuditLogService>();
builder.Services.AddSingleton<IRetentionService, RetentionService>();

// Health checks
builder.Services.AddHealthChecks()
    .AddCheck<ElasticsearchHealthCheck>("elasticsearch")
    .AddCheck<RabbitMqHealthCheck>("rabbitmq");

// OpenTelemetry 1.10+
builder.Services.AddLoggingServiceTelemetry(builder.Configuration);

// gRPC
builder.Services.AddGrpc();

// REST API
builder.Services.AddControllers();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen(c =>
{
    c.SwaggerDoc("v1", new() { Title = "Logging Service API", Version = "v1" });
});

var app = builder.Build();

// Initialize RabbitMQ connection
var queue = app.Services.GetRequiredService<ILogQueue>();
if (queue is RabbitMqLogQueue rabbitQueue)
{
    await rabbitQueue.InitializeAsync();
}

// Middleware pipeline
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseSerilogRequestLogging();
app.UseRouting();

// Prometheus metrics endpoint
app.UseHttpMetrics();
app.MapMetrics();

// Health check endpoints
app.MapHealthChecks("/health/live", new Microsoft.AspNetCore.Diagnostics.HealthChecks.HealthCheckOptions
{
    Predicate = _ => false // Liveness just checks if app is running
});

app.MapHealthChecks("/health/ready", new Microsoft.AspNetCore.Diagnostics.HealthChecks.HealthCheckOptions
{
    Predicate = _ => true // Readiness checks all dependencies
});

// Map endpoints
app.MapControllers();
app.MapGrpcService<LoggingGrpcService>();

app.MapGet("/", () => "Logging Service API v1");

Log.Information("Starting Logging Service API on .NET 9");
await app.RunAsync();
