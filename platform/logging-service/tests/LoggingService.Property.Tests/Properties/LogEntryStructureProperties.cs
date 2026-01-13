using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Models;
using LoggingService.Property.Tests.Generators;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property tests for LogEntry structure completeness.
/// Feature: logging-microservice
/// Property 3: Log Entry Structure Completeness
/// Validates: Requirements 2.1, 2.3
/// </summary>
public class LogEntryStructureProperties
{
    /// <summary>
    /// Property 3: Log Entry Structure Completeness
    /// For any valid LogEntry object created by the system, it SHALL contain all required fields
    /// (id, timestamp, correlation_id, service_id, level, message) with correct types,
    /// and the log level SHALL be one of: DEBUG, INFO, WARN, ERROR, FATAL.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property ValidLogEntry_ContainsAllRequiredFields()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                // Required fields must not be null or empty
                entry.Id.ShouldNotBeNullOrWhiteSpace("Id is required");
                entry.CorrelationId.ShouldNotBeNullOrWhiteSpace("CorrelationId is required");
                entry.ServiceId.ShouldNotBeNullOrWhiteSpace("ServiceId is required");
                entry.Message.ShouldNotBeNullOrWhiteSpace("Message is required");

                // Timestamp must be valid (not default)
                entry.Timestamp.ShouldNotBe(default(DateTimeOffset), "Timestamp is required");

                // Log level must be one of the valid values
                var validLevels = new[] { LogLevel.Debug, LogLevel.Info, LogLevel.Warn, LogLevel.Error, LogLevel.Fatal };
                validLevels.ShouldContain(entry.Level, "Level must be a valid LogLevel");

                return true;
            });
    }

    /// <summary>
    /// Property: LogLevel enum contains exactly the expected values.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property LogLevel_ContainsOnlyValidValues()
    {
        return Prop.ForAll(
            LogEntryGenerator.GenerateLogLevel(),
            level =>
            {
                var validLevels = new[] { LogLevel.Debug, LogLevel.Info, LogLevel.Warn, LogLevel.Error, LogLevel.Fatal };
                return validLevels.Contains(level);
            });
    }

    /// <summary>
    /// Property: LogLevel values have correct ordering (Debug < Info < Warn < Error < Fatal).
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public void LogLevel_HasCorrectOrdering()
    {
        ((int)LogLevel.Debug).ShouldBeLessThan((int)LogLevel.Info);
        ((int)LogLevel.Info).ShouldBeLessThan((int)LogLevel.Warn);
        ((int)LogLevel.Warn).ShouldBeLessThan((int)LogLevel.Error);
        ((int)LogLevel.Error).ShouldBeLessThan((int)LogLevel.Fatal);
    }

    /// <summary>
    /// Property: Optional fields can be null without affecting validity.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-microservice")]
    public Property OptionalFields_CanBeNull()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                // Create entry with all optional fields as null
                var entryWithNulls = entry with
                {
                    TraceId = null,
                    SpanId = null,
                    UserId = null,
                    RequestId = null,
                    Method = null,
                    Path = null,
                    StatusCode = null,
                    DurationMs = null,
                    Metadata = null,
                    Exception = null
                };

                // Entry should still be valid (required fields present)
                entryWithNulls.Id.ShouldNotBeNullOrWhiteSpace();
                entryWithNulls.CorrelationId.ShouldNotBeNullOrWhiteSpace();
                entryWithNulls.ServiceId.ShouldNotBeNullOrWhiteSpace();
                entryWithNulls.Message.ShouldNotBeNullOrWhiteSpace();
                entryWithNulls.Timestamp.ShouldNotBe(default(DateTimeOffset));

                return true;
            });
    }
}
