using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Configuration;
using LoggingService.Core.Interfaces;
using LoggingService.Core.Models;
using LoggingService.Core.Services;
using LoggingService.Infrastructure.Queue;
using LoggingService.Property.Tests.Generators;
using Microsoft.Extensions.Logging.Abstractions;
using Microsoft.Extensions.Options;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property tests for queue delivery guarantee.
/// Feature: logging-microservice
/// Property 6: Queue Delivery Guarantee
/// Validates: Requirements 3.1, 3.5
/// </summary>
public class DeliveryGuaranteeProperties
{
    private readonly InMemoryLogQueue _queue;
    private readonly LogIngestionService _ingestionService;

    public DeliveryGuaranteeProperties()
    {
        var queueOptions = Options.Create(new QueueOptions
        {
            MaxQueueSize = 100_000,
            WarningThresholdPercent = 80
        });

        _queue = new InMemoryLogQueue(queueOptions, NullLogger<InMemoryLogQueue>.Instance);

        var validator = new LogEntryValidator();
        var enricher = new LogEntryEnricher();
        var masker = new PiiMasker(new SecurityOptions { MaskingMode = PiiMaskingMode.None });

        _ingestionService = new LogIngestionService(
            _queue,
            validator,
            enricher,
            masker,
            NullLogger<LogIngestionService>.Instance);
    }

    /// <summary>
    /// Property 6: Queue Delivery Guarantee
    /// For any valid log entry that is successfully ingested, it SHALL be enqueued
    /// for processing, and eventually it SHALL appear in the storage layer.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property ValidEntry_IsEnqueued()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            async entry =>
            {
                var initialDepth = _queue.GetQueueDepth();

                var result = await _ingestionService.IngestAsync(entry);

                if (result.IsSuccess)
                {
                    var finalDepth = _queue.GetQueueDepth();
                    (finalDepth - initialDepth).ShouldBe(1);
                }

                return true;
            });
    }

    /// <summary>
    /// Property: Enqueued entries can be dequeued.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property EnqueuedEntries_CanBeDequeued()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            async entry =>
            {
                // Enqueue
                await _queue.EnqueueAsync(entry);

                // Dequeue
                var dequeued = await _queue.DequeueAsync();

                dequeued.ShouldNotBeNull();
                dequeued!.Id.ShouldBe(entry.Id);
                dequeued.Message.ShouldBe(entry.Message);

                return true;
            });
    }

    /// <summary>
    /// Property: Queue maintains FIFO order.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public async Task Queue_MaintainsFifoOrder()
    {
        var entries = Enumerable.Range(0, 10)
            .Select(i => CreateEntry($"entry-{i}"))
            .ToList();

        // Enqueue all
        foreach (var entry in entries)
        {
            await _queue.EnqueueAsync(entry);
        }

        // Dequeue and verify order
        for (int i = 0; i < 10; i++)
        {
            var dequeued = await _queue.DequeueAsync();
            dequeued.ShouldNotBeNull();
            dequeued!.Id.ShouldBe($"entry-{i}");
        }
    }

    /// <summary>
    /// Property: Batch enqueue maintains order.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public async Task BatchEnqueue_MaintainsOrder()
    {
        var entries = Enumerable.Range(0, 50)
            .Select(i => CreateEntry($"batch-entry-{i}"))
            .ToList();

        // Enqueue batch
        await _queue.EnqueueBatchAsync(entries);

        // Verify queue depth
        _queue.GetQueueDepth().ShouldBe(50);

        // Dequeue and verify order
        for (int i = 0; i < 50; i++)
        {
            var dequeued = await _queue.DequeueAsync();
            dequeued.ShouldNotBeNull();
            dequeued!.Id.ShouldBe($"batch-entry-{i}");
        }
    }

    /// <summary>
    /// Property: No entries are lost during ingestion.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public async Task NoEntries_AreLostDuringIngestion()
    {
        var entries = Enumerable.Range(0, 100)
            .Select(i => CreateEntry($"entry-{i}"))
            .ToList();

        var initialDepth = _queue.GetQueueDepth();

        // Ingest all entries
        foreach (var entry in entries)
        {
            var result = await _ingestionService.IngestAsync(entry);
            result.IsSuccess.ShouldBeTrue();
        }

        // All entries should be in queue
        var finalDepth = _queue.GetQueueDepth();
        (finalDepth - initialDepth).ShouldBe(100);

        // Dequeue all and verify
        var dequeuedIds = new HashSet<string>();
        for (int i = 0; i < 100; i++)
        {
            var dequeued = await _queue.DequeueAsync();
            dequeued.ShouldNotBeNull();
            dequeuedIds.Add(dequeued!.Id);
        }

        // All original IDs should be present (after enrichment, IDs are preserved)
        dequeuedIds.Count.ShouldBe(100);
    }

    /// <summary>
    /// Property: Queue depth is accurate.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public async Task QueueDepth_IsAccurate()
    {
        _queue.GetQueueDepth().ShouldBe(0);

        // Enqueue 10
        for (int i = 0; i < 10; i++)
        {
            await _queue.EnqueueAsync(CreateEntry($"entry-{i}"));
        }
        _queue.GetQueueDepth().ShouldBe(10);

        // Dequeue 5
        for (int i = 0; i < 5; i++)
        {
            await _queue.DequeueAsync();
        }
        _queue.GetQueueDepth().ShouldBe(5);

        // Enqueue 3 more
        for (int i = 0; i < 3; i++)
        {
            await _queue.EnqueueAsync(CreateEntry($"new-entry-{i}"));
        }
        _queue.GetQueueDepth().ShouldBe(8);
    }

    private static LogEntry CreateEntry(string id)
    {
        return new LogEntry
        {
            Id = id,
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "test-service",
            Level = LogLevel.Info,
            Message = $"Test message for {id}"
        };
    }
}
