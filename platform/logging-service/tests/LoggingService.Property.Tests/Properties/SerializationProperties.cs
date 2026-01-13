using System.Text.Json;
using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Models;
using LoggingService.Core.Serialization;
using LoggingService.Property.Tests.Generators;
using Shouldly;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property tests for JSON serialization.
/// Feature: logging-service-modernization, Property 1: Serialization Round-Trip
/// Validates: Requirements 2.5, 9.4
/// </summary>
public class SerializationProperties
{
    /// <summary>
    /// Property 1: Serialization Round-Trip
    /// For any valid LogEntry object, serializing it to JSON and then deserializing back
    /// SHALL produce an object that is equivalent to the original.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public Property SerializationRoundTrip_ProducesEquivalentObject()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                // Serialize to JSON
                var json = LogEntrySerializer.Serialize(entry);

                // Deserialize back
                var deserialized = LogEntrySerializer.Deserialize(json);

                // Should be equivalent
                deserialized.ShouldNotBeNull();
                deserialized!.Id.ShouldBe(entry.Id);
                deserialized.Timestamp.ShouldBe(entry.Timestamp);
                deserialized.CorrelationId.ShouldBe(entry.CorrelationId);
                deserialized.ServiceId.ShouldBe(entry.ServiceId);
                deserialized.Level.ShouldBe(entry.Level);
                deserialized.Message.ShouldBe(entry.Message);
                deserialized.TraceId.ShouldBe(entry.TraceId);
                deserialized.SpanId.ShouldBe(entry.SpanId);
                deserialized.UserId.ShouldBe(entry.UserId);
                deserialized.RequestId.ShouldBe(entry.RequestId);
                deserialized.Method.ShouldBe(entry.Method);
                deserialized.Path.ShouldBe(entry.Path);
                deserialized.StatusCode.ShouldBe(entry.StatusCode);
                deserialized.DurationMs.ShouldBe(entry.DurationMs);

                return true;
            });
    }

    /// <summary>
    /// Property: UTF-8 bytes round-trip produces equivalent object.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public Property Utf8BytesRoundTrip_ProducesEquivalentObject()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                // Serialize to UTF-8 bytes
                var bytes = LogEntrySerializer.SerializeToUtf8Bytes(entry);

                // Deserialize back
                var deserialized = LogEntrySerializer.Deserialize(bytes);

                // Should be equivalent
                deserialized.ShouldNotBeNull();
                deserialized!.Id.ShouldBe(entry.Id);
                deserialized.CorrelationId.ShouldBe(entry.CorrelationId);
                deserialized.ServiceId.ShouldBe(entry.ServiceId);
                deserialized.Level.ShouldBe(entry.Level);
                deserialized.Message.ShouldBe(entry.Message);

                return true;
            });
    }

    /// <summary>
    /// Property: Serialized JSON is valid JSON.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public Property SerializedJson_IsValidJson()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var json = LogEntrySerializer.Serialize(entry);

                // Should be parseable as JSON
                var parseAction = () => JsonDocument.Parse(json);
                Should.NotThrow(parseAction);

                return true;
            });
    }

    /// <summary>
    /// Property: Serialized JSON uses camelCase property names.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public Property SerializedJson_UsesCamelCase()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var json = LogEntrySerializer.Serialize(entry);

                // Should contain camelCase property names
                json.ShouldContain("\"id\":");
                json.ShouldContain("\"timestamp\":");
                json.ShouldContain("\"correlationId\":");
                json.ShouldContain("\"serviceId\":");
                json.ShouldContain("\"level\":");
                json.ShouldContain("\"message\":");

                // Should NOT contain PascalCase
                json.ShouldNotContain("\"Id\":");
                json.ShouldNotContain("\"Timestamp\":");
                json.ShouldNotContain("\"CorrelationId\":");

                return true;
            });
    }

    /// <summary>
    /// Property: Null optional fields are not included in JSON.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void NullOptionalFields_AreNotIncludedInJson()
    {
        var entry = new LogEntry
        {
            Id = "test-id",
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "test-service",
            Level = LogLevel.Info,
            Message = "Test message",
            TraceId = null,
            SpanId = null,
            UserId = null,
            Metadata = null,
            Exception = null
        };

        var json = LogEntrySerializer.Serialize(entry);

        // Null fields should not be present
        json.ShouldNotContain("\"traceId\":");
        json.ShouldNotContain("\"spanId\":");
        json.ShouldNotContain("\"userId\":");
        json.ShouldNotContain("\"metadata\":");
        json.ShouldNotContain("\"exception\":");
    }

    /// <summary>
    /// Property: LogLevel is serialized as string.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void LogLevel_IsSerializedAsString()
    {
        var entry = new LogEntry
        {
            Id = "test-id",
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "test-service",
            Level = LogLevel.Error,
            Message = "Test message"
        };

        var json = LogEntrySerializer.Serialize(entry);

        // Level should be serialized as string, not number
        (json.Contains("\"level\":\"error\"") || json.Contains("\"level\":\"Error\"")).ShouldBeTrue();
    }
}
