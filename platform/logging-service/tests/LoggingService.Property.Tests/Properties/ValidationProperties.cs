using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Models;
using LoggingService.Core.Services;
using LoggingService.Property.Tests.Generators;
using Shouldly;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property tests for input validation.
/// Feature: logging-service-modernization, Property 4: Input Validation Rejects Invalid Entries
/// Validates: Requirements 8.2
/// </summary>
public class ValidationProperties
{
    private readonly LogEntryValidator _validator = new();

    /// <summary>
    /// Property 4: Input Validation Rejects Invalid Entries
    /// For any log entry that is missing required fields (timestamp, service_id, level, message),
    /// the validator SHALL reject it with field-specific validation messages.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public Property ValidLogEntry_PassesValidation()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var result = _validator.Validate(entry);
                return result.IsValid;
            });
    }

    /// <summary>
    /// Property: Entries with missing timestamp are rejected.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public Property MissingTimestamp_FailsValidation()
    {
        return Prop.ForAll(
            InvalidLogEntryGenerator.WithMissingTimestamp(),
            entry =>
            {
                var result = _validator.Validate(entry);

                result.IsValid.ShouldBeFalse();
                result.Errors.ShouldContain(e => e.Field == "timestamp" && e.Code == "REQUIRED");

                return true;
            });
    }

    /// <summary>
    /// Property: Entries with empty service ID are rejected.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public Property EmptyServiceId_FailsValidation()
    {
        return Prop.ForAll(
            InvalidLogEntryGenerator.WithEmptyServiceId(),
            entry =>
            {
                var result = _validator.Validate(entry);

                result.IsValid.ShouldBeFalse();
                result.Errors.ShouldContain(e => e.Field == "serviceId" && e.Code == "REQUIRED");

                return true;
            });
    }

    /// <summary>
    /// Property: Entries with whitespace-only service ID are rejected.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public Property WhitespaceServiceId_FailsValidation()
    {
        return Prop.ForAll(
            InvalidLogEntryGenerator.WithWhitespaceServiceId(),
            entry =>
            {
                var result = _validator.Validate(entry);

                result.IsValid.ShouldBeFalse();
                result.Errors.ShouldContain(e => e.Field == "serviceId" && e.Code == "REQUIRED");

                return true;
            });
    }

    /// <summary>
    /// Property: Entries with empty message are rejected.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public Property EmptyMessage_FailsValidation()
    {
        return Prop.ForAll(
            InvalidLogEntryGenerator.WithEmptyMessage(),
            entry =>
            {
                var result = _validator.Validate(entry);

                result.IsValid.ShouldBeFalse();
                result.Errors.ShouldContain(e => e.Field == "message" && e.Code == "REQUIRED");

                return true;
            });
    }

    /// <summary>
    /// Property: Entries with whitespace-only message are rejected.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public Property WhitespaceMessage_FailsValidation()
    {
        return Prop.ForAll(
            InvalidLogEntryGenerator.WithWhitespaceMessage(),
            entry =>
            {
                var result = _validator.Validate(entry);

                result.IsValid.ShouldBeFalse();
                result.Errors.ShouldContain(e => e.Field == "message" && e.Code == "REQUIRED");

                return true;
            });
    }

    /// <summary>
    /// Property: Validation errors contain field-specific information.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void ValidationErrors_ContainFieldSpecificInformation()
    {
        var invalidEntry = new LogEntry
        {
            Id = "test-id",
            Timestamp = default, // Invalid
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "", // Invalid
            Level = LogLevel.Info,
            Message = "   " // Invalid (whitespace only)
        };

        var result = _validator.Validate(invalidEntry);

        result.IsValid.ShouldBeFalse();
        result.Errors.Count.ShouldBe(3);
        result.Errors.ShouldContain(e => e.Field == "timestamp");
        result.Errors.ShouldContain(e => e.Field == "serviceId");
        result.Errors.ShouldContain(e => e.Field == "message");

        // Each error should have code and message
        foreach (var error in result.Errors)
        {
            error.Code.ShouldNotBeNullOrWhiteSpace();
            error.Message.ShouldNotBeNullOrWhiteSpace();
        }
    }
}
