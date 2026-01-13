using System.Text.Json;
using System.Text.Json.Serialization;
using LoggingService.Core.Models;

namespace LoggingService.Core.Serialization;

/// <summary>
/// JSON serialization context for LogEntry and related types.
/// Uses source generation for better performance.
/// </summary>
[JsonSourceGenerationOptions(
    PropertyNamingPolicy = JsonKnownNamingPolicy.CamelCase,
    WriteIndented = false,
    DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull)]
[JsonSerializable(typeof(LogEntry))]
[JsonSerializable(typeof(LogEntry[]))]
[JsonSerializable(typeof(List<LogEntry>))]
[JsonSerializable(typeof(ExceptionInfo))]
[JsonSerializable(typeof(ValidationResult))]
[JsonSerializable(typeof(FieldError))]
[JsonSerializable(typeof(IngestResult))]
[JsonSerializable(typeof(BatchIngestResult))]
[JsonSerializable(typeof(ErrorResponse))]
[JsonSerializable(typeof(LogQuery))]
[JsonSerializable(typeof(PagedResult<LogEntry>))]
[JsonSerializable(typeof(Dictionary<string, object>))]
public partial class LogEntryJsonContext : JsonSerializerContext
{
}

/// <summary>
/// JSON serializer configuration for the logging service.
/// </summary>
public static class LogEntrySerializer
{
    /// <summary>
    /// Default JSON serializer options for log entries.
    /// </summary>
    public static JsonSerializerOptions Options { get; } = new()
    {
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
        WriteIndented = false,
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull,
        Converters =
        {
            new JsonStringEnumConverter(JsonNamingPolicy.CamelCase)
        },
        TypeInfoResolver = LogEntryJsonContext.Default
    };

    /// <summary>
    /// Serializes a log entry to JSON.
    /// </summary>
    public static string Serialize(LogEntry entry)
    {
        return JsonSerializer.Serialize(entry, Options);
    }

    /// <summary>
    /// Serializes a log entry to UTF-8 bytes.
    /// </summary>
    public static byte[] SerializeToUtf8Bytes(LogEntry entry)
    {
        return JsonSerializer.SerializeToUtf8Bytes(entry, Options);
    }

    /// <summary>
    /// Deserializes a log entry from JSON.
    /// </summary>
    public static LogEntry? Deserialize(string json)
    {
        return JsonSerializer.Deserialize<LogEntry>(json, Options);
    }

    /// <summary>
    /// Deserializes a log entry from UTF-8 bytes.
    /// </summary>
    public static LogEntry? Deserialize(ReadOnlySpan<byte> utf8Json)
    {
        return JsonSerializer.Deserialize<LogEntry>(utf8Json, Options);
    }
}
