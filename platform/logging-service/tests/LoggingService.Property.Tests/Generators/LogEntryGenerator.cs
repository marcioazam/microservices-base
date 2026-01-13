using FsCheck;
using LoggingService.Core.Models;

namespace LoggingService.Property.Tests.Generators;

/// <summary>
/// FsCheck 3.x generators for LogEntry and related types.
/// </summary>
public static class LogEntryGenerator
{
    /// <summary>
    /// Generates valid LogEntry instances.
    /// </summary>
    public static Arbitrary<LogEntry> Generate() =>
        Arb.From(
            from id in Arb.Generate<NonEmptyString>()
            from timestamp in GenerateTimestamp()
            from correlationId in GenerateUuid()
            from serviceId in GenerateServiceId()
            from level in GenerateLogLevelGen()
            from message in GenerateMessage()
            from traceId in Gen.OneOf(Gen.Constant<string?>(null), GenerateHexString(32))
            from spanId in Gen.OneOf(Gen.Constant<string?>(null), GenerateHexString(16))
            from userId in Gen.OneOf(Gen.Constant<string?>(null), GenerateUuid())
            from requestId in Gen.OneOf(Gen.Constant<string?>(null), GenerateUuid())
            from method in Gen.OneOf(Gen.Constant<string?>(null), GenerateHttpMethod())
            from path in Gen.OneOf(Gen.Constant<string?>(null), GeneratePath())
            from statusCode in Gen.OneOf(Gen.Constant<int?>(null), GenerateStatusCode())
            from durationMs in Gen.OneOf(Gen.Constant<long?>(null), Gen.Choose(0, 60000).Select(x => (long?)x))
            select new LogEntry
            {
                Id = id.Get,
                Timestamp = timestamp,
                CorrelationId = correlationId,
                ServiceId = serviceId,
                Level = level,
                Message = message,
                TraceId = traceId,
                SpanId = spanId,
                UserId = userId,
                RequestId = requestId,
                Method = method,
                Path = path,
                StatusCode = statusCode,
                DurationMs = durationMs,
                Metadata = null,
                Exception = null
            });

    /// <summary>
    /// Generates valid LogLevel values.
    /// </summary>
    public static Arbitrary<LogLevel> GenerateLogLevel() =>
        Arb.From(GenerateLogLevelGen());

    private static Gen<LogLevel> GenerateLogLevelGen() =>
        Gen.Elements(LogLevel.Debug, LogLevel.Info, LogLevel.Warn, LogLevel.Error, LogLevel.Fatal);

    private static Gen<DateTimeOffset> GenerateTimestamp() =>
        Gen.Choose(0, 365 * 5)
            .Select(days => DateTimeOffset.UtcNow.AddDays(-days).AddSeconds(Random.Shared.Next(0, 86400)));

    private static Gen<string> GenerateUuid() =>
        Gen.Fresh(() => Guid.NewGuid().ToString());

    private static Gen<string> GenerateServiceId() =>
        Gen.Elements(
            "auth-edge-service",
            "token-service",
            "iam-policy-service",
            "user-service",
            "notification-service",
            "audit-service");

    private static Gen<string> GenerateMessage() =>
        Gen.OneOf(
            Gen.Constant("Request processed successfully"),
            Gen.Constant("User authentication completed"),
            Gen.Constant("Token validation failed"),
            Gen.Constant("Database connection established"),
            Gen.Constant("Cache miss for key"),
            from text in Arb.Generate<NonEmptyString>()
            select $"Log message: {text.Get}");

    private static Gen<string?> GenerateHexString(int length) =>
        Gen.ArrayOf(length / 2, Gen.Choose(0, 255))
            .Select(bytes => string.Concat(bytes.Select(b => b.ToString("x2"))));

    private static Gen<string?> GenerateHttpMethod() =>
        Gen.Elements<string?>("GET", "POST", "PUT", "DELETE", "PATCH");

    private static Gen<string?> GeneratePath() =>
        Gen.Elements<string?>(
            "/api/v1/users",
            "/api/v1/auth/login",
            "/api/v1/tokens/refresh",
            "/api/v1/policies",
            "/health/live");

    private static Gen<int?> GenerateStatusCode() =>
        Gen.Elements<int?>(200, 201, 204, 400, 401, 403, 404, 500, 502, 503);

    /// <summary>
    /// Generates LogEntry with PII data for testing masking.
    /// </summary>
    public static Gen<LogEntry> WithPiiData() =>
        from entry in Generate().Generator
        from email in GenerateEmail()
        from phone in GeneratePhone()
        from ip in GenerateIpAddress()
        select entry with
        {
            Message = $"User {email} logged in from {ip}, contact: {phone}",
            UserId = email,
            Path = $"/api/users/{email}/profile"
        };

    private static Gen<string> GenerateEmail() =>
        from name in Gen.Elements("john", "jane", "bob", "alice", "test")
        from domain in Gen.Elements("example.com", "test.org", "mail.net")
        select $"{name}@{domain}";

    private static Gen<string> GeneratePhone() =>
        from areaCode in Gen.Choose(100, 999)
        from prefix in Gen.Choose(100, 999)
        from line in Gen.Choose(1000, 9999)
        select $"+1-{areaCode}-{prefix}-{line}";

    private static Gen<string> GenerateIpAddress() =>
        from a in Gen.Choose(1, 255)
        from b in Gen.Choose(0, 255)
        from c in Gen.Choose(0, 255)
        from d in Gen.Choose(1, 254)
        select $"{a}.{b}.{c}.{d}";
}

/// <summary>
/// Generators for invalid LogEntry instances (for testing validation).
/// </summary>
public static class InvalidLogEntryGenerator
{
    /// <summary>
    /// Generates LogEntry with missing timestamp.
    /// </summary>
    public static Gen<LogEntry> WithMissingTimestamp() =>
        from entry in LogEntryGenerator.Generate().Generator
        select entry with { Timestamp = default };

    /// <summary>
    /// Generates LogEntry with empty service ID.
    /// </summary>
    public static Gen<LogEntry> WithEmptyServiceId() =>
        from entry in LogEntryGenerator.Generate().Generator
        select entry with { ServiceId = "" };

    /// <summary>
    /// Generates LogEntry with whitespace-only service ID.
    /// </summary>
    public static Gen<LogEntry> WithWhitespaceServiceId() =>
        from entry in LogEntryGenerator.Generate().Generator
        select entry with { ServiceId = "   " };

    /// <summary>
    /// Generates LogEntry with empty message.
    /// </summary>
    public static Gen<LogEntry> WithEmptyMessage() =>
        from entry in LogEntryGenerator.Generate().Generator
        select entry with { Message = "" };

    /// <summary>
    /// Generates LogEntry with whitespace-only message.
    /// </summary>
    public static Gen<LogEntry> WithWhitespaceMessage() =>
        from entry in LogEntryGenerator.Generate().Generator
        select entry with { Message = "   " };

    /// <summary>
    /// Generates LogEntry with empty ID.
    /// </summary>
    public static Gen<LogEntry> WithEmptyId() =>
        from entry in LogEntryGenerator.Generate().Generator
        select entry with { Id = "" };

    /// <summary>
    /// Generates LogEntry with empty correlation ID.
    /// </summary>
    public static Gen<LogEntry> WithEmptyCorrelationId() =>
        from entry in LogEntryGenerator.Generate().Generator
        select entry with { CorrelationId = "" };
}
