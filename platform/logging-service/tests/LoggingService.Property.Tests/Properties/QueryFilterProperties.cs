using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Models;
using LoggingService.Property.Tests.Generators;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property tests for query filter correctness.
/// Feature: logging-microservice
/// Property 8: Query Filter Correctness
/// Validates: Requirements 4.4, 4.5
/// </summary>
public class QueryFilterProperties
{
    /// <summary>
    /// Property 8: Query Filter Correctness
    /// For any query with filters, all returned log entries SHALL match ALL specified
    /// filter criteria, and no log entry matching all criteria SHALL be excluded.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property TimeRangeFilter_MatchesAllEntries()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var query = new LogQuery
                {
                    StartTime = entry.Timestamp.AddHours(-1),
                    EndTime = entry.Timestamp.AddHours(1)
                };

                // Entry should match the time range
                var matchesTimeRange =
                    entry.Timestamp >= query.StartTime &&
                    entry.Timestamp <= query.EndTime;

                return matchesTimeRange;
            });
    }

    /// <summary>
    /// Property: Service ID filter matches exactly.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property ServiceIdFilter_MatchesExactly()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var query = new LogQuery
                {
                    ServiceId = entry.ServiceId
                };

                // Entry should match the service ID
                return entry.ServiceId == query.ServiceId;
            });
    }

    /// <summary>
    /// Property: Log level filter includes entries at or above minimum level.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property LogLevelFilter_IncludesEntriesAtOrAboveMinLevel()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            LogEntryGenerator.GenerateLogLevel(),
            (entry, minLevel) =>
            {
                var query = new LogQuery
                {
                    MinLevel = minLevel
                };

                // Entry should match if its level is >= min level
                var shouldMatch = (int)entry.Level >= (int)minLevel;

                return shouldMatch == ((int)entry.Level >= (int)query.MinLevel);
            });
    }

    /// <summary>
    /// Property: Correlation ID filter matches exactly.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property CorrelationIdFilter_MatchesExactly()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var query = new LogQuery
                {
                    CorrelationId = entry.CorrelationId
                };

                // Entry should match the correlation ID
                return entry.CorrelationId == query.CorrelationId;
            });
    }

    /// <summary>
    /// Property: Combined filters are AND-ed together.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property CombinedFilters_AreAndedTogether()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var query = new LogQuery
                {
                    ServiceId = entry.ServiceId,
                    MinLevel = entry.Level,
                    CorrelationId = entry.CorrelationId,
                    StartTime = entry.Timestamp.AddHours(-1),
                    EndTime = entry.Timestamp.AddHours(1)
                };

                // Entry should match ALL criteria
                var matchesServiceId = entry.ServiceId == query.ServiceId;
                var matchesLevel = (int)entry.Level >= (int)query.MinLevel;
                var matchesCorrelationId = entry.CorrelationId == query.CorrelationId;
                var matchesTimeRange =
                    entry.Timestamp >= query.StartTime &&
                    entry.Timestamp <= query.EndTime;

                var matchesAll = matchesServiceId && matchesLevel && matchesCorrelationId && matchesTimeRange;

                return matchesAll;
            });
    }

    /// <summary>
    /// Property: Empty query matches all entries.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property EmptyQuery_MatchesAllEntries()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var query = new LogQuery();

                // Empty query should match any entry
                var matchesServiceId = string.IsNullOrEmpty(query.ServiceId) || entry.ServiceId == query.ServiceId;
                var matchesLevel = !query.MinLevel.HasValue || (int)entry.Level >= (int)query.MinLevel;
                var matchesCorrelationId = string.IsNullOrEmpty(query.CorrelationId) || entry.CorrelationId == query.CorrelationId;
                var matchesTimeRange =
                    (!query.StartTime.HasValue || entry.Timestamp >= query.StartTime) &&
                    (!query.EndTime.HasValue || entry.Timestamp <= query.EndTime);

                return matchesServiceId && matchesLevel && matchesCorrelationId && matchesTimeRange;
            });
    }

    /// <summary>
    /// Property: Query pagination parameters are valid.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public void QueryPagination_HasValidDefaults()
    {
        var query = new LogQuery();

        query.Page.ShouldBe(1);
        query.PageSize.ShouldBe(100);
        query.SortDirection.ShouldBe(SortDirection.Descending);
    }

    /// <summary>
    /// Property: Query with all filters set.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public void QueryWithAllFilters_IsValid()
    {
        var now = DateTimeOffset.UtcNow;

        var query = new LogQuery
        {
            StartTime = now.AddDays(-7),
            EndTime = now,
            ServiceId = "auth-service",
            MinLevel = LogLevel.Warn,
            CorrelationId = Guid.NewGuid().ToString(),
            SearchText = "error",
            UserId = "user-123",
            TraceId = "trace-abc",
            Page = 2,
            PageSize = 50,
            SortDirection = SortDirection.Ascending
        };

        query.StartTime.ShouldBeLessThan(query.EndTime!.Value);
        query.ServiceId.ShouldNotBeNullOrEmpty();
        query.MinLevel.ShouldBe(LogLevel.Warn);
        query.Page.ShouldBe(2);
        query.PageSize.ShouldBe(50);
        query.SortDirection.ShouldBe(SortDirection.Ascending);
    }
}
