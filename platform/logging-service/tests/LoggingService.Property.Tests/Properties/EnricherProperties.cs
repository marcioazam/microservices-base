using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Models;
using LoggingService.Core.Services;
using LoggingService.Property.Tests.Generators;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property tests for log entry enrichment.
/// Feature: logging-microservice
/// Property 4: Correlation ID Generation
/// Validates: Requirements 2.2
/// </summary>
public class EnricherProperties
{
    private readonly LogEntryEnricher _enricher = new();

    /// <summary>
    /// Property 4: Correlation ID Generation
    /// For any log entry received without a correlation_id, after processing by the Logging_Service,
    /// the entry SHALL have a valid UUID v4 assigned as its correlation_id.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property MissingCorrelationId_GetsValidUuid()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                // Create entry without correlation ID
                var entryWithoutCorrelationId = entry with { CorrelationId = "" };

                var enrichedEntry = _enricher.Enrich(entryWithoutCorrelationId);

                // Should have a valid UUID
                enrichedEntry.CorrelationId.ShouldNotBeNullOrWhiteSpace();
                Guid.TryParse(enrichedEntry.CorrelationId, out var guid).ShouldBeTrue();
                guid.ShouldNotBe(Guid.Empty);

                return true;
            });
    }

    /// <summary>
    /// Property: Existing correlation ID is preserved.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property ExistingCorrelationId_IsPreserved()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var originalCorrelationId = entry.CorrelationId;

                var enrichedEntry = _enricher.Enrich(entry);

                // Original correlation ID should be preserved
                enrichedEntry.CorrelationId.ShouldBe(originalCorrelationId);

                return true;
            });
    }

    /// <summary>
    /// Property: Missing ID gets generated.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property MissingId_GetsGenerated()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var entryWithoutId = entry with { Id = "" };

                var enrichedEntry = _enricher.Enrich(entryWithoutId);

                enrichedEntry.Id.ShouldNotBeNullOrWhiteSpace();
                Guid.TryParse(enrichedEntry.Id, out var guid).ShouldBeTrue();
                guid.ShouldNotBe(Guid.Empty);

                return true;
            });
    }

    /// <summary>
    /// Property: Timestamp is normalized to UTC.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property Timestamp_IsNormalizedToUtc()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                // Create entry with non-UTC timestamp
                var nonUtcTimestamp = new DateTimeOffset(2024, 1, 15, 10, 30, 0, TimeSpan.FromHours(5));
                var entryWithNonUtc = entry with { Timestamp = nonUtcTimestamp };

                var enrichedEntry = _enricher.Enrich(entryWithNonUtc);

                // Timestamp should be in UTC
                enrichedEntry.Timestamp.Offset.ShouldBe(TimeSpan.Zero);

                return true;
            });
    }

    /// <summary>
    /// Property: UTC timestamp is preserved.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property UtcTimestamp_IsPreserved()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                // Ensure entry has UTC timestamp
                var utcTimestamp = DateTimeOffset.UtcNow;
                var entryWithUtc = entry with { Timestamp = utcTimestamp };

                var enrichedEntry = _enricher.Enrich(entryWithUtc);

                // Timestamp should be preserved
                enrichedEntry.Timestamp.ShouldBe(utcTimestamp);

                return true;
            });
    }

    /// <summary>
    /// Property: Generated correlation IDs are unique.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public void GeneratedCorrelationIds_AreUnique()
    {
        var baseEntry = new LogEntry
        {
            Id = "test",
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = "",
            ServiceId = "test-service",
            Level = LogLevel.Info,
            Message = "Test message"
        };

        var correlationIds = new HashSet<string>();

        for (int i = 0; i < 1000; i++)
        {
            var enrichedEntry = _enricher.Enrich(baseEntry);
            correlationIds.Add(enrichedEntry.CorrelationId);
        }

        // All generated IDs should be unique
        correlationIds.Count.ShouldBe(1000);
    }
}
