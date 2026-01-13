# Design Document: Logging Service Modernization

## Overview

This design document describes the modernization of the `platform/logging-service` to December 2025 state-of-the-art standards. The modernization focuses on:

1. **Platform Upgrade**: .NET 8 → .NET 9 with C# 13
2. **Client Modernization**: NEST → Elastic.Clients.Elasticsearch 8.x, RabbitMQ.Client 6.x → 7.x
3. **Redundancy Elimination**: Centralized metrics, configuration, and shared types
4. **Testing Modernization**: xUnit 3.x, Testcontainers 4.x, FsCheck 3.x
5. **Observability Enhancement**: OpenTelemetry 1.10+

## Architecture

The service maintains a clean layered architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                      API Layer                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ REST API    │  │ gRPC API    │  │ Health Endpoints    │  │
│  │ Controllers │  │ Services    │  │ /health/live,ready  │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Core Layer                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Services    │  │ Validators  │  │ Models & Interfaces │  │
│  │ Ingestion   │  │ Enricher    │  │ LogEntry, LogQuery  │  │
│  │ PII Masker  │  │ Retention   │  │ Configuration       │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────┐│
│  │              Centralized Observability                  ││
│  │  Metrics Registry │ Tracing │ Structured Logging        ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Infrastructure Layer                        │
│  ┌─────────────────────┐  ┌─────────────────────────────┐   │
│  │ Elasticsearch 8.x   │  │ RabbitMQ 7.x                │   │
│  │ Elastic.Clients     │  │ Async Channels              │   │
│  │ ElasticsearchClient │  │ IConnection, IChannel       │   │
│  └─────────────────────┘  └─────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Worker Layer                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ LogProcessorWorker - Background Service                 ││
│  │ Batch Processing │ Graceful Shutdown │ Metrics          ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### Core Interfaces (Unchanged)

The existing interfaces remain stable to maintain backward compatibility:

```csharp
public interface ILogIngestionService
{
    Task<IngestResult> IngestAsync(LogEntry entry, CancellationToken ct = default);
    Task<BatchIngestResult> IngestBatchAsync(IEnumerable<LogEntry> entries, CancellationToken ct = default);
}

public interface ILogRepository
{
    Task<string> SaveAsync(LogEntry entry, CancellationToken ct = default);
    Task SaveBatchAsync(IEnumerable<LogEntry> entries, CancellationToken ct = default);
    Task<LogEntry?> GetByIdAsync(string id, CancellationToken ct = default);
    Task<PagedResult<LogEntry>> QueryAsync(LogQuery query, CancellationToken ct = default);
    Task<long> DeleteOlderThanAsync(DateTimeOffset olderThan, LogLevel? level = null, CancellationToken ct = default);
    Task<long> ArchiveOlderThanAsync(DateTimeOffset olderThan, CancellationToken ct = default);
}

public interface ILogQueue
{
    Task EnqueueAsync(LogEntry entry, CancellationToken ct = default);
    Task EnqueueBatchAsync(IEnumerable<LogEntry> entries, CancellationToken ct = default);
    Task<LogEntry?> DequeueAsync(CancellationToken ct = default);
    int GetQueueDepth();
    bool IsFull();
    double GetCapacityPercentage();
}
```

### Elasticsearch Client Modernization

Replace NEST with Elastic.Clients.Elasticsearch 8.x:

```csharp
// Before (NEST 7.x - DEPRECATED)
var client = new ElasticClient(connectionSettings);
var response = await client.IndexAsync(entry, i => i.Index(indexName).Id(entry.Id));

// After (Elastic.Clients.Elasticsearch 8.x)
var client = new ElasticsearchClient(settings);
var response = await client.IndexAsync(entry, idx => idx.Index(indexName).Id(entry.Id));
```

New repository implementation:

```csharp
public sealed class ElasticsearchLogRepository : ILogRepository
{
    private readonly ElasticsearchClient _client;
    
    public async Task<string> SaveAsync(LogEntry entry, CancellationToken ct = default)
    {
        var response = await _client.IndexAsync(entry, idx => idx
            .Index(GetIndexName(entry))
            .Id(entry.Id), ct);
        
        if (!response.IsValidResponse)
            throw new StorageException($"Failed to index: {response.DebugInformation}");
        
        return entry.Id;
    }
    
    public async Task<PagedResult<LogEntry>> QueryAsync(LogQuery query, CancellationToken ct = default)
    {
        var response = await _client.SearchAsync<LogEntry>(s => s
            .Index($"{_options.IndexPrefix}-*")
            .From((query.Page - 1) * query.PageSize)
            .Size(Math.Min(query.PageSize, 1000))
            .Sort(sort => query.SortDirection == SortDirection.Ascending
                ? sort.Field(f => f.Timestamp, d => d.Order(SortOrder.Asc))
                : sort.Field(f => f.Timestamp, d => d.Order(SortOrder.Desc)))
            .Query(q => BuildQuery(q, query))
            .TrackTotalHits(new TrackHits(true)), ct);
        
        return new PagedResult<LogEntry>
        {
            Items = response.Documents.ToList(),
            TotalCount = (int)response.Total,
            Page = query.Page,
            PageSize = query.PageSize
        };
    }
}
```

### RabbitMQ Client Modernization

Upgrade to RabbitMQ.Client 7.x with async-first API:

```csharp
// Before (RabbitMQ.Client 6.x)
var factory = new ConnectionFactory { Uri = new Uri(connectionString) };
var connection = factory.CreateConnection();
var channel = connection.CreateModel();

// After (RabbitMQ.Client 7.x)
var factory = new ConnectionFactory { Uri = new Uri(connectionString) };
await using var connection = await factory.CreateConnectionAsync();
await using var channel = await connection.CreateChannelAsync();
```

New queue implementation:

```csharp
public sealed class RabbitMqLogQueue : ILogQueue, IAsyncDisposable
{
    private IConnection? _connection;
    private IChannel? _channel;
    
    public async Task InitializeAsync(CancellationToken ct = default)
    {
        var factory = new ConnectionFactory { Uri = new Uri(_options.ConnectionString) };
        _connection = await factory.CreateConnectionAsync(ct);
        _channel = await _connection.CreateChannelAsync(ct);
        
        await _channel.QueueDeclareAsync(
            queue: _options.QueueName,
            durable: true,
            exclusive: false,
            autoDelete: false,
            arguments: new Dictionary<string, object?>
            {
                ["x-max-length"] = _options.MaxQueueSize
            }, ct);
    }
    
    public async Task EnqueueAsync(LogEntry entry, CancellationToken ct = default)
    {
        var body = LogEntrySerializer.SerializeToUtf8Bytes(entry);
        var properties = new BasicProperties
        {
            Persistent = true,
            ContentType = "application/json",
            MessageId = entry.Id,
            CorrelationId = entry.CorrelationId
        };
        
        await _channel!.BasicPublishAsync(
            exchange: string.Empty,
            routingKey: _options.QueueName,
            mandatory: false,
            basicProperties: properties,
            body: body, ct);
    }
    
    public async ValueTask DisposeAsync()
    {
        if (_channel != null) await _channel.DisposeAsync();
        if (_connection != null) await _connection.DisposeAsync();
    }
}
```

### Centralized Metrics Registry

Consolidate all metrics in a single location:

```csharp
// Core/Observability/LoggingMetrics.cs
public static class LoggingMetrics
{
    private static readonly string[] ServiceLevelLabels = ["service_id", "level"];
    private static readonly string[] ErrorTypeLabels = ["error_type"];
    private static readonly string[] FieldCodeLabels = ["field", "error_code"];
    
    // Ingestion metrics
    public static readonly Counter<long> LogsReceived = CreateCounter<long>(
        "logging_logs_received_total", "Total logs received", ServiceLevelLabels);
    
    public static readonly Counter<long> LogsRejected = CreateCounter<long>(
        "logging_logs_rejected_total", "Total logs rejected", ["reason"]);
    
    // Processing metrics
    public static readonly Counter<long> LogsProcessed = CreateCounter<long>(
        "logging_logs_processed_total", "Total logs processed");
    
    public static readonly Counter<long> LogsFailed = CreateCounter<long>(
        "logging_logs_failed_total", "Total logs failed", ErrorTypeLabels);
    
    // Queue metrics
    public static readonly Gauge<int> QueueDepth = CreateGauge<int>(
        "logging_queue_depth", "Current queue depth");
    
    public static readonly Gauge<double> QueueCapacity = CreateGauge<double>(
        "logging_queue_capacity_percent", "Queue capacity percentage");
    
    // Latency metrics
    public static readonly Histogram<double> IngestLatency = CreateHistogram<double>(
        "logging_ingest_latency_seconds", "Ingest latency",
        [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5]);
    
    public static readonly Histogram<double> ProcessingLatency = CreateHistogram<double>(
        "logging_processing_latency_seconds", "Processing latency",
        [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5]);
    
    public static readonly Histogram<double> QueryLatency = CreateHistogram<double>(
        "logging_query_latency_seconds", "Query latency",
        [0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5]);
    
    // Storage metrics
    public static readonly Counter<long> StorageSaved = CreateCounter<long>(
        "logging_storage_saved_total", "Total logs saved");
    
    public static readonly Counter<long> StorageFailed = CreateCounter<long>(
        "logging_storage_save_failed_total", "Total save failures");
    
    // Validation metrics
    public static readonly Counter<long> ValidationErrors = CreateCounter<long>(
        "logging_validation_errors_total", "Validation errors", FieldCodeLabels);
}
```

### Consolidated Configuration

Single configuration model with nested options:

```csharp
public sealed record LoggingServiceOptions
{
    public const string SectionName = "LoggingService";
    
    public ElasticsearchOptions Elasticsearch { get; init; } = new();
    public QueueOptions Queue { get; init; } = new();
    public RetentionOptions Retention { get; init; } = new();
    public SecurityOptions Security { get; init; } = new();
    public ObservabilityOptions Observability { get; init; } = new();
}

public sealed record ObservabilityOptions
{
    public string OtlpEndpoint { get; init; } = "http://localhost:4317";
    public string ServiceName { get; init; } = "logging-service";
    public string ServiceVersion { get; init; } = "1.0.0";
    public bool EnableTracing { get; init; } = true;
    public bool EnableMetrics { get; init; } = true;
}
```

## Data Models

The existing data models remain unchanged to maintain backward compatibility:

```csharp
public sealed record LogEntry
{
    public required string Id { get; init; }
    public required DateTimeOffset Timestamp { get; init; }
    public required string CorrelationId { get; init; }
    public required string ServiceId { get; init; }
    public required LogLevel Level { get; init; }
    public required string Message { get; init; }
    public string? TraceId { get; init; }
    public string? SpanId { get; init; }
    public string? UserId { get; init; }
    public string? RequestId { get; init; }
    public string? Method { get; init; }
    public string? Path { get; init; }
    public int? StatusCode { get; init; }
    public long? DurationMs { get; init; }
    public Dictionary<string, object>? Metadata { get; init; }
    public ExceptionInfo? Exception { get; init; }
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Serialization Round-Trip

*For any* valid LogEntry object, serializing it to JSON and then deserializing back SHALL produce an object that is equivalent to the original.

**Validates: Requirements 2.5, 9.4**

### Property 2: Correlation ID Presence

*For any* log entry processed through the enrichment pipeline, the output SHALL contain a valid non-empty correlation ID (UUID format).

**Validates: Requirements 7.3**

### Property 3: PII Masking Completeness

*For any* log entry containing PII patterns (email addresses, phone numbers, IP addresses, SSNs), after processing by the PII_Masker, those patterns SHALL NOT be present in the output message, path, userId, or metadata fields.

**Validates: Requirements 8.1**

### Property 4: Input Validation Rejects Invalid Entries

*For any* log entry that is missing required fields (timestamp, service_id, level, message) or contains invalid values, the validator SHALL reject it with field-specific validation messages.

**Validates: Requirements 8.2**

### Property 5: Internal Error Hiding

*For any* internal error that occurs during processing, the client-facing error response SHALL NOT contain stack traces, internal exception messages, or implementation details.

**Validates: Requirements 8.5**

### Property 6: Backpressure on Queue Full

*For any* attempt to enqueue a log entry when the queue is at capacity, the ingestion service SHALL return a QueueFull result without blocking indefinitely.

**Validates: Requirements 9.5**

### Property 7: API Backward Compatibility

*For any* valid request that was accepted by the previous API version, the modernized API SHALL accept it and produce a compatible response structure.

**Validates: Requirements 1.4**

## Error Handling

### Error Categories

1. **Validation Errors**: Return 400 Bad Request with field-specific errors
2. **Queue Full**: Return 503 Service Unavailable with retry-after header
3. **Storage Errors**: Log error, increment failure metric, return 500 Internal Server Error
4. **Authentication Errors**: Return 401 Unauthorized or 403 Forbidden

### Error Response Format

```csharp
public sealed record ErrorResponse
{
    public required string Code { get; init; }
    public required string Message { get; init; }
    public string? CorrelationId { get; init; }
    public IReadOnlyList<FieldError>? FieldErrors { get; init; }
    public DateTimeOffset Timestamp { get; init; } = DateTimeOffset.UtcNow;
}
```

### Exception Handling Strategy

- Use Result pattern for expected failures (validation, queue full)
- Use exceptions for unexpected failures (storage errors, network issues)
- Never expose internal exception details to clients
- Always log exceptions with correlation ID for debugging

## Testing Strategy

### Dual Testing Approach

The testing strategy combines unit tests and property-based tests for comprehensive coverage:

1. **Unit Tests**: Verify specific examples, edge cases, and error conditions
2. **Property Tests**: Verify universal properties across all valid inputs

### Testing Framework

- **xUnit 3.x**: Primary test framework
- **FsCheck 3.x**: Property-based testing library
- **FluentAssertions 7.x**: Assertion library (or TUnit as alternative due to licensing)
- **NSubstitute 5.x**: Mocking library
- **Testcontainers 4.x**: Integration test containers

### Property Test Configuration

Each property test MUST:
- Run minimum 100 iterations
- Use smart generators that constrain to valid input space
- Reference the design document property number
- Tag format: `**Feature: logging-service-modernization, Property {number}: {property_text}**`

### Test Organization

```
tests/
├── LoggingService.Unit.Tests/
│   ├── Services/
│   │   ├── LogEntryValidatorTests.cs
│   │   ├── LogEntryEnricherTests.cs
│   │   ├── PiiMaskerTests.cs
│   │   └── LogIngestionServiceTests.cs
│   └── Serialization/
│       └── LogEntrySerializerTests.cs
├── LoggingService.Property.Tests/
│   ├── Generators/
│   │   └── LogEntryGenerator.cs
│   └── Properties/
│       ├── SerializationProperties.cs
│       ├── ValidationProperties.cs
│       ├── PiiMaskingProperties.cs
│       ├── EnricherProperties.cs
│       └── BackpressureProperties.cs
└── LoggingService.Integration.Tests/
    ├── ElasticsearchRepositoryTests.cs
    ├── RabbitMqQueueTests.cs
    └── ApiEndpointTests.cs
```

### Coverage Requirements

- Minimum 80% code coverage overall
- 100% coverage for security-critical paths (PII masking, validation)
- All correctness properties implemented as property tests
