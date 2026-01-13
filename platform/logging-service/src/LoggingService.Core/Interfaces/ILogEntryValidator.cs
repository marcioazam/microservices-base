using LoggingService.Core.Models;

namespace LoggingService.Core.Interfaces;

/// <summary>
/// Validator for log entries.
/// </summary>
public interface ILogEntryValidator
{
    /// <summary>
    /// Validates a log entry.
    /// </summary>
    /// <param name="entry">The log entry to validate.</param>
    /// <returns>Validation result with any errors.</returns>
    ValidationResult Validate(LogEntry entry);
}
