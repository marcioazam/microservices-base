using LoggingService.Core.Models;

namespace LoggingService.Core.Interfaces;

/// <summary>
/// Masks personally identifiable information (PII) in log entries.
/// </summary>
public interface IPiiMasker
{
    /// <summary>
    /// Masks sensitive data in a log entry.
    /// </summary>
    /// <param name="entry">The log entry to mask.</param>
    /// <returns>The log entry with masked PII.</returns>
    LogEntry MaskSensitiveData(LogEntry entry);
}
