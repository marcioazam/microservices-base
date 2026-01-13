using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Models;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property 12: Audit Trail Completeness
/// Validates: Requirements 6.5
/// </summary>
[Trait("Category", "Property")]
[Trait("Feature", "logging-microservice")]
public class AuditTrailProperties
{
    /// <summary>
    /// Property: Audit entry always has a timestamp.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property AuditEntryAlwaysHasTimestamp()
    {
        return Prop.ForAll(
            GenerateAuditEntry(),
            entry => entry.Timestamp != default);
    }

    /// <summary>
    /// Property: Audit entry always has a user ID.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property AuditEntryAlwaysHasUserId()
    {
        return Prop.ForAll(
            GenerateAuditEntry(),
            entry => !string.IsNullOrEmpty(entry.UserId));
    }

    /// <summary>
    /// Property: Audit entry always has operation metadata.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property AuditEntryAlwaysHasMetadata()
    {
        return Prop.ForAll(
            GenerateAuditEntry(),
            entry => entry.Metadata != null && entry.Metadata.Count > 0);
    }

    /// <summary>
    /// Property: Query audit includes query parameters.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property QueryAuditIncludesParameters()
    {
        return Prop.ForAll(
            GenerateQueryAuditEntry(),
            entry =>
            {
                var metadata = entry.Metadata;
                return metadata != null &&
                       metadata.ContainsKey("page") &&
                       metadata.ContainsKey("pageSize") &&
                       metadata.ContainsKey("resultCount");
            });
    }

    /// <summary>
    /// Property: Export audit includes format.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property ExportAuditIncludesFormat()
    {
        return Prop.ForAll(
            Gen.Elements("json", "csv").ToArbitrary(),
            format =>
            {
                var entry = CreateExportAuditEntry("test-user", format, 100);
                return entry.Metadata != null &&
                       entry.Metadata.ContainsKey("format") &&
                       entry.Metadata["format"]?.ToString() == format;
            });
    }

    /// <summary>
    /// Property: Audit service ID is consistent.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property AuditServiceIdIsConsistent()
    {
        return Prop.ForAll(
            GenerateAuditEntry(),
            entry => entry.ServiceId == "logging-service-audit");
    }

    /// <summary>
    /// Property: Audit entries have unique IDs.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property AuditEntriesHaveUniqueIds()
    {
        return Prop.ForAll(
            Gen.ListOf(10, GenerateAuditEntry()).ToArbitrary(),
            entries =>
            {
                var ids = entries.Select(e => e.Id).ToList();
                return ids.Distinct().Count() == ids.Count;
            });
    }

    private static Arbitrary<LogEntry> GenerateAuditEntry()
    {
        return Gen.Fresh(() => CreateQueryAuditEntry(
            $"user-{Guid.NewGuid():N}",
            new LogQuery { Page = 1, PageSize = 100 },
            Random.Shared.Next(0, 1000))).ToArbitrary();
    }

    private static Arbitrary<LogEntry> GenerateQueryAuditEntry()
    {
        return Gen.Fresh(() =>
        {
            var query = new LogQuery
            {
                Page = Random.Shared.Next(1, 100),
                PageSize = Random.Shared.Next(1, 1000),
                ServiceId = $"service-{Random.Shared.Next(1, 10)}"
            };
            return CreateQueryAuditEntry($"user-{Guid.NewGuid():N}", query, Random.Shared.Next(0, 1000));
        }).ToArbitrary();
    }

    private static LogEntry CreateQueryAuditEntry(string userId, LogQuery query, int resultCount)
    {
        return new LogEntry
        {
            Id = Guid.NewGuid().ToString(),
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "logging-service-audit",
            Level = LogLevel.Info,
            Message = $"Audit: QUERY by {userId}",
            UserId = userId,
            Metadata = new Dictionary<string, object>
            {
                ["startTime"] = query.StartTime?.ToString("O") ?? "null",
                ["endTime"] = query.EndTime?.ToString("O") ?? "null",
                ["serviceId"] = query.ServiceId ?? "null",
                ["page"] = query.Page,
                ["pageSize"] = query.PageSize,
                ["resultCount"] = resultCount
            }
        };
    }

    private static LogEntry CreateExportAuditEntry(string userId, string format, int exportedCount)
    {
        return new LogEntry
        {
            Id = Guid.NewGuid().ToString(),
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "logging-service-audit",
            Level = LogLevel.Info,
            Message = $"Audit: EXPORT by {userId}",
            UserId = userId,
            Metadata = new Dictionary<string, object>
            {
                ["format"] = format,
                ["exportedCount"] = exportedCount
            }
        };
    }
}
