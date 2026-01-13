namespace LoggingService.Core.Configuration;

/// <summary>
/// Root configuration options for the logging service.
/// </summary>
public sealed record LoggingServiceOptions
{
    /// <summary>
    /// Configuration section name.
    /// </summary>
    public const string SectionName = "LoggingService";

    /// <summary>
    /// ElasticSearch configuration.
    /// </summary>
    public ElasticSearchOptions ElasticSearch { get; init; } = new();

    /// <summary>
    /// Queue configuration.
    /// </summary>
    public QueueOptions Queue { get; init; } = new();

    /// <summary>
    /// Retention policy configuration.
    /// </summary>
    public RetentionOptions Retention { get; init; } = new();

    /// <summary>
    /// Security configuration.
    /// </summary>
    public SecurityOptions Security { get; init; } = new();

    /// <summary>
    /// Observability configuration.
    /// </summary>
    public ObservabilityOptions Observability { get; init; } = new();
}

/// <summary>
/// Observability configuration options for OpenTelemetry and metrics.
/// </summary>
public sealed record ObservabilityOptions
{
    /// <summary>
    /// Configuration section name.
    /// </summary>
    public const string SectionName = "LoggingService:Observability";

    /// <summary>
    /// OTLP endpoint for trace export.
    /// </summary>
    public string OtlpEndpoint { get; init; } = "http://localhost:4317";

    /// <summary>
    /// Service name for telemetry.
    /// </summary>
    public string ServiceName { get; init; } = "logging-service";

    /// <summary>
    /// Service version for telemetry.
    /// </summary>
    public string ServiceVersion { get; init; } = "1.0.0";

    /// <summary>
    /// Whether to enable distributed tracing.
    /// </summary>
    public bool EnableTracing { get; init; } = true;

    /// <summary>
    /// Whether to enable metrics collection.
    /// </summary>
    public bool EnableMetrics { get; init; } = true;

    /// <summary>
    /// Sampling ratio for traces (0.0 to 1.0).
    /// </summary>
    public double TraceSamplingRatio { get; init; } = 1.0;
}

/// <summary>
/// ElasticSearch configuration options.
/// </summary>
public sealed record ElasticSearchOptions
{
    /// <summary>
    /// Configuration section name.
    /// </summary>
    public const string SectionName = "LoggingService:ElasticSearch";

    /// <summary>
    /// ElasticSearch node URLs.
    /// </summary>
    public string[] Nodes { get; init; } = ["http://localhost:9200"];

    /// <summary>
    /// Index name prefix.
    /// </summary>
    public string IndexPrefix { get; init; } = "logs";

    /// <summary>
    /// Number of shards per index.
    /// </summary>
    public int NumberOfShards { get; init; } = 3;

    /// <summary>
    /// Number of replicas per index.
    /// </summary>
    public int NumberOfReplicas { get; init; } = 1;

    /// <summary>
    /// Username for authentication (optional).
    /// </summary>
    public string? Username { get; init; }

    /// <summary>
    /// Password for authentication (optional).
    /// </summary>
    public string? Password { get; init; }
}

/// <summary>
/// Queue configuration options.
/// </summary>
public sealed record QueueOptions
{
    /// <summary>
    /// Configuration section name.
    /// </summary>
    public const string SectionName = "LoggingService:Queue";

    /// <summary>
    /// RabbitMQ connection string.
    /// </summary>
    public string ConnectionString { get; init; } = "amqp://guest:guest@localhost:5672";

    /// <summary>
    /// Queue name for log ingestion.
    /// </summary>
    public string QueueName { get; init; } = "logs-ingestion";

    /// <summary>
    /// Maximum queue size before backpressure is applied.
    /// </summary>
    public int MaxQueueSize { get; init; } = 100_000;

    /// <summary>
    /// Batch size for processing.
    /// </summary>
    public int BatchSize { get; init; } = 100;

    /// <summary>
    /// Interval between processing batches.
    /// </summary>
    public TimeSpan ProcessingInterval { get; init; } = TimeSpan.FromMilliseconds(100);

    /// <summary>
    /// Warning threshold percentage (0-100).
    /// </summary>
    public int WarningThresholdPercent { get; init; } = 80;
}

/// <summary>
/// Retention policy configuration options.
/// </summary>
public sealed record RetentionOptions
{
    /// <summary>
    /// Configuration section name.
    /// </summary>
    public const string SectionName = "LoggingService:Retention";

    /// <summary>
    /// Retention period for DEBUG logs.
    /// </summary>
    public TimeSpan DebugRetention { get; init; } = TimeSpan.FromDays(7);

    /// <summary>
    /// Retention period for INFO logs.
    /// </summary>
    public TimeSpan InfoRetention { get; init; } = TimeSpan.FromDays(30);

    /// <summary>
    /// Retention period for WARN logs.
    /// </summary>
    public TimeSpan WarnRetention { get; init; } = TimeSpan.FromDays(90);

    /// <summary>
    /// Retention period for ERROR logs.
    /// </summary>
    public TimeSpan ErrorRetention { get; init; } = TimeSpan.FromDays(365);

    /// <summary>
    /// Retention period for FATAL logs.
    /// </summary>
    public TimeSpan FatalRetention { get; init; } = TimeSpan.FromDays(365);

    /// <summary>
    /// Whether to archive logs to cold storage before deletion.
    /// </summary>
    public bool ArchiveBeforeDelete { get; init; } = true;

    /// <summary>
    /// Cold storage connection string (S3, Azure Blob, etc.).
    /// </summary>
    public string? ColdStorageConnectionString { get; init; }
}
