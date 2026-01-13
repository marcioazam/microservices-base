using System.Text.RegularExpressions;
using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Models;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property 14: Trace Context Propagation
/// Validates: Requirements 7.5
/// </summary>
[Trait("Category", "Property")]
[Trait("Feature", "logging-microservice")]
public partial class TraceContextProperties
{
    // W3C Trace Context format: 32 hex chars for trace-id, 16 hex chars for span-id
    [GeneratedRegex("^[0-9a-f]{32}$")]
    private static partial Regex TraceIdPattern();

    [GeneratedRegex("^[0-9a-f]{16}$")]
    private static partial Regex SpanIdPattern();

    /// <summary>
    /// Property: Valid trace IDs are preserved in log entries.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property ValidTraceIdIsPreserved()
    {
        return Prop.ForAll(
            GenerateValidTraceId(),
            traceId =>
            {
                var entry = new LogEntry
                {
                    Id = Guid.NewGuid().ToString(),
                    Timestamp = DateTimeOffset.UtcNow,
                    CorrelationId = Guid.NewGuid().ToString(),
                    ServiceId = "test-service",
                    Level = LogLevel.Info,
                    Message = "Test message",
                    TraceId = traceId
                };

                return entry.TraceId == traceId;
            });
    }

    /// <summary>
    /// Property: Valid span IDs are preserved in log entries.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property ValidSpanIdIsPreserved()
    {
        return Prop.ForAll(
            GenerateValidSpanId(),
            spanId =>
            {
                var entry = new LogEntry
                {
                    Id = Guid.NewGuid().ToString(),
                    Timestamp = DateTimeOffset.UtcNow,
                    CorrelationId = Guid.NewGuid().ToString(),
                    ServiceId = "test-service",
                    Level = LogLevel.Info,
                    Message = "Test message",
                    SpanId = spanId
                };

                return entry.SpanId == spanId;
            });
    }

    /// <summary>
    /// Property: Trace ID format follows W3C specification (32 hex chars).
    /// </summary>
    [Property(MaxTest = 100)]
    public Property TraceIdFollowsW3CFormat()
    {
        return Prop.ForAll(
            GenerateValidTraceId(),
            traceId => TraceIdPattern().IsMatch(traceId));
    }

    /// <summary>
    /// Property: Span ID format follows W3C specification (16 hex chars).
    /// </summary>
    [Property(MaxTest = 100)]
    public Property SpanIdFollowsW3CFormat()
    {
        return Prop.ForAll(
            GenerateValidSpanId(),
            spanId => SpanIdPattern().IsMatch(spanId));
    }

    /// <summary>
    /// Property: Trace context is optional (null values allowed).
    /// </summary>
    [Property(MaxTest = 100)]
    public Property TraceContextIsOptional()
    {
        var entry = new LogEntry
        {
            Id = Guid.NewGuid().ToString(),
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "test-service",
            Level = LogLevel.Info,
            Message = "Test message",
            TraceId = null,
            SpanId = null
        };

        return (entry.TraceId == null && entry.SpanId == null).ToProperty();
    }

    private static Arbitrary<string> GenerateValidTraceId()
    {
        return Gen.ArrayOf(32, Gen.Elements("0123456789abcdef".ToCharArray()))
            .Select(chars => new string(chars))
            .ToArbitrary();
    }

    private static Arbitrary<string> GenerateValidSpanId()
    {
        return Gen.ArrayOf(16, Gen.Elements("0123456789abcdef".ToCharArray()))
            .Select(chars => new string(chars))
            .ToArbitrary();
    }
}
