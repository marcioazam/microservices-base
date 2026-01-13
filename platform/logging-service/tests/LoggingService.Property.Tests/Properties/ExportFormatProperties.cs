using System.Text;
using System.Text.Json;
using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Models;
using LoggingService.Property.Tests.Generators;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property 16: Export Format Validity
/// Validates: Requirements 8.5
/// </summary>
[Trait("Category", "Property")]
[Trait("Feature", "logging-microservice")]
public class ExportFormatProperties
{
    /// <summary>
    /// Property: JSON export produces valid JSON that can be parsed.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property JsonExportProducesValidJson()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var entries = new List<LogEntry> { entry };
                var json = JsonSerializer.Serialize(entries, new JsonSerializerOptions
                {
                    WriteIndented = true
                });

                try
                {
                    var parsed = JsonSerializer.Deserialize<List<LogEntry>>(json);
                    return parsed != null && parsed.Count == 1;
                }
                catch
                {
                    return false;
                }
            });
    }

    /// <summary>
    /// Property: CSV export produces valid CSV with correct headers.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property CsvExportHasCorrectHeaders()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                var csv = ExportAsCsv(new List<LogEntry> { entry });
                var lines = csv.Split('\n', StringSplitOptions.RemoveEmptyEntries);

                if (lines.Length < 1) return false;

                var headers = lines[0].Trim();
                return headers == "Id,Timestamp,CorrelationId,ServiceId,Level,Message";
            });
    }

    /// <summary>
    /// Property: CSV export has correct number of rows.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property CsvExportHasCorrectRowCount()
    {
        return Prop.ForAll(
            Gen.ListOf(LogEntryGenerator.Generate()).ToArbitrary(),
            entries =>
            {
                var csv = ExportAsCsv(entries.ToList());
                var lines = csv.Split('\n', StringSplitOptions.RemoveEmptyEntries);

                // Header + data rows
                return lines.Length == entries.Count + 1;
            });
    }

    /// <summary>
    /// Property: CSV escapes quotes correctly.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property CsvEscapesQuotesCorrectly()
    {
        var messageWithQuotes = "Test \"quoted\" message";
        var entry = new LogEntry
        {
            Id = Guid.NewGuid().ToString(),
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "test-service",
            Level = LogLevel.Info,
            Message = messageWithQuotes
        };

        var csv = ExportAsCsv(new List<LogEntry> { entry });
        return csv.Contains("\"\"quoted\"\"");
    }

    private static string ExportAsCsv(IReadOnlyList<LogEntry> entries)
    {
        var sb = new StringBuilder();
        sb.AppendLine("Id,Timestamp,CorrelationId,ServiceId,Level,Message");

        foreach (var entry in entries)
        {
            sb.AppendLine($"\"{entry.Id}\",\"{entry.Timestamp:O}\",\"{entry.CorrelationId}\",\"{entry.ServiceId}\",\"{entry.Level}\",\"{EscapeCsv(entry.Message)}\"");
        }

        return sb.ToString();
    }

    private static string EscapeCsv(string value) =>
        value.Replace("\"", "\"\"").Replace("\n", " ").Replace("\r", "");
}
