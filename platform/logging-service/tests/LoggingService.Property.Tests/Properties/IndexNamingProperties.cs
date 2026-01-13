using Elastic.Clients.Elasticsearch;
using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Configuration;
using LoggingService.Core.Models;
using LoggingService.Infrastructure.Storage;
using LoggingService.Property.Tests.Generators;
using Microsoft.Extensions.Logging.Abstractions;
using Microsoft.Extensions.Options;
using NSubstitute;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property tests for ElasticSearch index naming.
/// Feature: logging-microservice
/// Property 7: Index Naming Convention
/// Validates: Requirements 4.1, 4.2
/// </summary>
public class IndexNamingProperties
{
    private readonly ElasticsearchLogRepository _repository;

    public IndexNamingProperties()
    {
        var client = Substitute.For<ElasticsearchClient>();
        var options = Options.Create(new ElasticSearchOptions
        {
            IndexPrefix = "logs",
            Nodes = ["http://localhost:9200"]
        });
        var logger = NullLogger<ElasticsearchLogRepository>.Instance;

        _repository = new ElasticsearchLogRepository(client, options, logger);
    }

    /// <summary>
    /// Property 7: Index Naming Convention
    /// For any log entry persisted to ElasticSearch, the index name SHALL follow
    /// the format `logs-{service_id}-{yyyy.MM.dd}` where the date corresponds
    /// to the log entry's timestamp.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property IndexName_FollowsConvention()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var indexName = _repository.GetIndexName(entry);

                // Should start with prefix
                indexName.ShouldStartWith("logs-");

                // Should contain service ID (lowercase)
                var expectedServiceId = entry.ServiceId.ToLowerInvariant().Replace(" ", "-");
                indexName.ShouldContain(expectedServiceId);

                // Should end with date in correct format
                var expectedDate = entry.Timestamp.ToString("yyyy.MM.dd");
                indexName.ShouldEndWith(expectedDate);

                // Full format check
                var expectedIndexName = $"logs-{expectedServiceId}-{expectedDate}";
                indexName.ShouldBe(expectedIndexName);

                return true;
            });
    }

    /// <summary>
    /// Property: Index name contains valid date from timestamp.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property IndexName_ContainsValidDateFromTimestamp()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var indexName = _repository.GetIndexName(entry);

                // Extract date part from index name
                var parts = indexName.Split('-');
                var datePart = parts[^1]; // Last part should be the date

                // Should be valid date format
                datePart.ShouldMatch(@"^\d{4}\.\d{2}\.\d{2}$");

                // Date should match entry timestamp
                var expectedDate = entry.Timestamp.ToString("yyyy.MM.dd");
                datePart.ShouldBe(expectedDate);

                return true;
            });
    }

    /// <summary>
    /// Property: Index name is lowercase.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property IndexName_IsLowercase()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var indexName = _repository.GetIndexName(entry);

                // Index name should be lowercase (except for date separators)
                var withoutDate = indexName[..^11]; // Remove date part
                withoutDate.ShouldBe(withoutDate.ToLowerInvariant());

                return true;
            });
    }

    /// <summary>
    /// Property: Same service and date produce same index name.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public void SameServiceAndDate_ProduceSameIndexName()
    {
        var timestamp = new DateTimeOffset(2024, 6, 15, 10, 30, 0, TimeSpan.Zero);

        var entry1 = new LogEntry
        {
            Id = "id-1",
            Timestamp = timestamp,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "auth-service",
            Level = LogLevel.Info,
            Message = "Message 1"
        };

        var entry2 = new LogEntry
        {
            Id = "id-2",
            Timestamp = timestamp.AddHours(5), // Same day, different time
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "auth-service",
            Level = LogLevel.Error,
            Message = "Message 2"
        };

        var indexName1 = _repository.GetIndexName(entry1);
        var indexName2 = _repository.GetIndexName(entry2);

        indexName1.ShouldBe(indexName2);
        indexName1.ShouldBe("logs-auth-service-2024.06.15");
    }

    /// <summary>
    /// Property: Different dates produce different index names.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public void DifferentDates_ProduceDifferentIndexNames()
    {
        var entry1 = new LogEntry
        {
            Id = "id-1",
            Timestamp = new DateTimeOffset(2024, 6, 15, 10, 30, 0, TimeSpan.Zero),
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "auth-service",
            Level = LogLevel.Info,
            Message = "Message 1"
        };

        var entry2 = new LogEntry
        {
            Id = "id-2",
            Timestamp = new DateTimeOffset(2024, 6, 16, 10, 30, 0, TimeSpan.Zero),
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "auth-service",
            Level = LogLevel.Info,
            Message = "Message 2"
        };

        var indexName1 = _repository.GetIndexName(entry1);
        var indexName2 = _repository.GetIndexName(entry2);

        indexName1.ShouldNotBe(indexName2);
        indexName1.ShouldBe("logs-auth-service-2024.06.15");
        indexName2.ShouldBe("logs-auth-service-2024.06.16");
    }

    /// <summary>
    /// Property: Different services produce different index names.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public void DifferentServices_ProduceDifferentIndexNames()
    {
        var timestamp = new DateTimeOffset(2024, 6, 15, 10, 30, 0, TimeSpan.Zero);

        var entry1 = new LogEntry
        {
            Id = "id-1",
            Timestamp = timestamp,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "auth-service",
            Level = LogLevel.Info,
            Message = "Message 1"
        };

        var entry2 = new LogEntry
        {
            Id = "id-2",
            Timestamp = timestamp,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "token-service",
            Level = LogLevel.Info,
            Message = "Message 2"
        };

        var indexName1 = _repository.GetIndexName(entry1);
        var indexName2 = _repository.GetIndexName(entry2);

        indexName1.ShouldNotBe(indexName2);
        indexName1.ShouldBe("logs-auth-service-2024.06.15");
        indexName2.ShouldBe("logs-token-service-2024.06.15");
    }
}
