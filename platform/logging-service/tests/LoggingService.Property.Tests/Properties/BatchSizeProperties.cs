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
/// Property tests for batch size enforcement.
/// Feature: logging-microservice
/// Property 2: Batch Size Enforcement
/// Validates: Requirements 1.5
/// </summary>
public class BatchSizeProperties
{
    private readonly LogIngestionService _service;
    private readonly InMemoryLogQueue _queue;

    public BatchSizeProperties()
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

        _service = new LogIngestionService(
            _queue,
            validator,
            enricher,
            masker,
            NullLogger<LogIngestionService>.Instance);
    }

    /// <summary>
    /// Property 2: Batch Size Enforcement
    /// For any batch submission request, if the batch contains more than 1000 entries,
    /// the Logging_Service SHALL reject the entire batch with HTTP 400.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public async Task BatchOver1000_IsRejected()
    {
        // Create batch of 1001 entries
        var entries = Enumerable.Range(0, 1001)
            .Select(i => CreateValidEntry($"entry-{i}"))
            .ToList();

        var result = await _service.IngestBatchAsync(entries);

        result.IsBatchTooLarge.ShouldBeTrue();
        result.SuccessCount.ShouldBe(0);
        result.FailedCount.ShouldBe(1001);
    }

    /// <summary>
    /// Property: Batch of exactly 1000 entries is accepted.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public async Task BatchOfExactly1000_IsAccepted()
    {
        var entries = Enumerable.Range(0, 1000)
            .Select(i => CreateValidEntry($"entry-{i}"))
            .ToList();

        var result = await _service.IngestBatchAsync(entries);

        result.IsBatchTooLarge.ShouldBeFalse();
        result.SuccessCount.ShouldBe(1000);
        result.FailedCount.ShouldBe(0);
    }

    /// <summary>
    /// Property: Batch under 1000 entries is accepted.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property BatchUnder1000_IsAccepted()
    {
        return Prop.ForAll(
            Gen.Choose(1, 999).ToArbitrary(),
            async batchSize =>
            {
                var entries = Enumerable.Range(0, batchSize)
                    .Select(i => CreateValidEntry($"entry-{i}"))
                    .ToList();

                var result = await _service.IngestBatchAsync(entries);

                result.IsBatchTooLarge.ShouldBeFalse();
                result.SuccessCount.ShouldBe(batchSize);

                return true;
            });
    }

    /// <summary>
    /// Property: Empty batch is accepted.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public async Task EmptyBatch_IsAccepted()
    {
        var entries = new List<LogEntry>();

        var result = await _service.IngestBatchAsync(entries);

        result.IsBatchTooLarge.ShouldBeFalse();
        result.SuccessCount.ShouldBe(0);
        result.FailedCount.ShouldBe(0);
    }

    /// <summary>
    /// Property: Batch with mixed valid/invalid entries processes correctly.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public async Task BatchWithMixedEntries_ProcessesCorrectly()
    {
        var entries = new List<LogEntry>
        {
            CreateValidEntry("valid-1"),
            CreateInvalidEntry("invalid-1"), // Missing message
            CreateValidEntry("valid-2"),
            CreateInvalidEntry("invalid-2"),
            CreateValidEntry("valid-3")
        };

        var result = await _service.IngestBatchAsync(entries);

        result.IsBatchTooLarge.ShouldBeFalse();
        result.SuccessCount.ShouldBe(3);
        result.FailedCount.ShouldBe(2);
        result.Results.Count.ShouldBe(5);
    }

    /// <summary>
    /// Property: All valid entries in batch are enqueued.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public async Task AllValidEntries_AreEnqueued()
    {
        var initialDepth = _queue.GetQueueDepth();

        var entries = Enumerable.Range(0, 50)
            .Select(i => CreateValidEntry($"entry-{i}"))
            .ToList();

        await _service.IngestBatchAsync(entries);

        var finalDepth = _queue.GetQueueDepth();
        (finalDepth - initialDepth).ShouldBe(50);
    }

    private static LogEntry CreateValidEntry(string id)
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

    private static LogEntry CreateInvalidEntry(string id)
    {
        return new LogEntry
        {
            Id = id,
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "test-service",
            Level = LogLevel.Info,
            Message = "" // Invalid: empty message
        };
    }
}
