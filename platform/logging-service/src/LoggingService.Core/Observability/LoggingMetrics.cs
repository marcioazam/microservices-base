using Prometheus;

namespace LoggingService.Core.Observability;

/// <summary>
/// Centralized Prometheus metrics registry for the logging service.
/// All metrics are defined here to eliminate duplication across services.
/// </summary>
public static class LoggingMetrics
{
    // ============================================
    // Ingestion Metrics
    // ============================================

    /// <summary>
    /// Total number of logs received by the ingestion service.
    /// </summary>
    public static readonly Counter LogsReceived = Metrics.CreateCounter(
        "logging_logs_received_total",
        "Total number of logs received",
        new CounterConfiguration
        {
            LabelNames = ["service_id", "level"]
        });

    /// <summary>
    /// Total number of logs rejected due to validation or backpressure.
    /// </summary>
    public static readonly Counter LogsRejected = Metrics.CreateCounter(
        "logging_logs_rejected_total",
        "Total number of logs rejected",
        new CounterConfiguration
        {
            LabelNames = ["reason"]
        });

    /// <summary>
    /// Time taken to ingest a log entry.
    /// </summary>
    public static readonly Histogram IngestLatency = Metrics.CreateHistogram(
        "logging_ingest_latency_seconds",
        "Time taken to ingest a log entry",
        new HistogramConfiguration
        {
            Buckets = [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5]
        });

    // ============================================
    // Processing Metrics
    // ============================================

    /// <summary>
    /// Total number of logs processed and stored.
    /// </summary>
    public static readonly Counter LogsProcessed = Metrics.CreateCounter(
        "logging_logs_processed_total",
        "Total number of logs processed and stored");

    /// <summary>
    /// Total number of logs that failed processing.
    /// </summary>
    public static readonly Counter LogsFailed = Metrics.CreateCounter(
        "logging_logs_failed_total",
        "Total number of logs that failed processing",
        new CounterConfiguration
        {
            LabelNames = ["error_type"]
        });

    /// <summary>
    /// Time taken to process a batch of logs.
    /// </summary>
    public static readonly Histogram ProcessingLatency = Metrics.CreateHistogram(
        "logging_processing_latency_seconds",
        "Time taken to process a batch of logs",
        new HistogramConfiguration
        {
            Buckets = [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5]
        });

    // ============================================
    // Queue Metrics
    // ============================================

    /// <summary>
    /// Current number of logs in the processing queue.
    /// </summary>
    public static readonly Gauge QueueDepth = Metrics.CreateGauge(
        "logging_queue_depth",
        "Current number of logs in the processing queue");

    /// <summary>
    /// Queue capacity percentage (0-100).
    /// </summary>
    public static readonly Gauge QueueCapacity = Metrics.CreateGauge(
        "logging_queue_capacity_percent",
        "Queue capacity percentage");

    /// <summary>
    /// Total number of logs enqueued.
    /// </summary>
    public static readonly Counter QueueEnqueued = Metrics.CreateCounter(
        "logging_queue_enqueued_total",
        "Total number of logs enqueued");

    /// <summary>
    /// Total number of logs dequeued.
    /// </summary>
    public static readonly Counter QueueDequeued = Metrics.CreateCounter(
        "logging_queue_dequeued_total",
        "Total number of logs dequeued");

    // ============================================
    // Storage Metrics
    // ============================================

    /// <summary>
    /// Total number of logs saved to storage.
    /// </summary>
    public static readonly Counter StorageSaved = Metrics.CreateCounter(
        "logging_storage_saved_total",
        "Total number of logs saved to storage");

    /// <summary>
    /// Alias for StorageSaved for backward compatibility.
    /// </summary>
    public static Counter LogsStored => StorageSaved;

    /// <summary>
    /// Total number of failed save operations.
    /// </summary>
    public static readonly Counter StorageFailed = Metrics.CreateCounter(
        "logging_storage_save_failed_total",
        "Total number of failed save operations");

    /// <summary>
    /// Alias for StorageFailed for backward compatibility.
    /// </summary>
    public static Counter StorageErrors => StorageFailed;

    /// <summary>
    /// Time taken to execute a log query.
    /// </summary>
    public static readonly Histogram QueryLatency = Metrics.CreateHistogram(
        "logging_query_latency_seconds",
        "Time taken to execute a log query",
        new HistogramConfiguration
        {
            Buckets = [0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5]
        });

    /// <summary>
    /// Total number of queries executed.
    /// </summary>
    public static readonly Counter QueriesExecuted = Metrics.CreateCounter(
        "logging_queries_executed_total",
        "Total number of queries executed");

    // ============================================
    // Validation Metrics
    // ============================================

    /// <summary>
    /// Total number of validation errors.
    /// </summary>
    public static readonly Counter ValidationErrors = Metrics.CreateCounter(
        "logging_validation_errors_total",
        "Total number of validation errors",
        new CounterConfiguration
        {
            LabelNames = ["field", "error_code"]
        });

    // ============================================
    // PII Masking Metrics
    // ============================================

    /// <summary>
    /// Total number of PII patterns masked.
    /// </summary>
    public static readonly Counter PiiMasked = Metrics.CreateCounter(
        "logging_pii_masked_total",
        "Total number of PII patterns masked",
        new CounterConfiguration
        {
            LabelNames = ["pattern_type"]
        });

    // ============================================
    // Health Metrics
    // ============================================

    /// <summary>
    /// Indicates if the service is healthy (1) or unhealthy (0).
    /// </summary>
    public static readonly Gauge ServiceHealth = Metrics.CreateGauge(
        "logging_service_health",
        "Service health status (1=healthy, 0=unhealthy)",
        new GaugeConfiguration
        {
            LabelNames = ["component"]
        });
}
