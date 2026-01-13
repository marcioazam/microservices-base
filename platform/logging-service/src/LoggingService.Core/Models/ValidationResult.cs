namespace LoggingService.Core.Models;

/// <summary>
/// Result of a validation operation.
/// </summary>
public sealed record ValidationResult
{
    /// <summary>
    /// Indicates whether the validation passed.
    /// </summary>
    public bool IsValid { get; init; }

    /// <summary>
    /// List of validation errors if validation failed.
    /// </summary>
    public IReadOnlyList<FieldError> Errors { get; init; } = [];

    /// <summary>
    /// Creates a successful validation result.
    /// </summary>
    public static ValidationResult Valid() => new() { IsValid = true };

    /// <summary>
    /// Creates a failed validation result with errors.
    /// </summary>
    public static ValidationResult Invalid(IEnumerable<FieldError> errors) =>
        new() { IsValid = false, Errors = errors.ToList() };
}

/// <summary>
/// Represents a validation error for a specific field.
/// </summary>
public sealed record FieldError
{
    /// <summary>
    /// Name of the field that failed validation.
    /// </summary>
    public required string Field { get; init; }

    /// <summary>
    /// Error code for programmatic handling.
    /// </summary>
    public required string Code { get; init; }

    /// <summary>
    /// Human-readable error message.
    /// </summary>
    public required string Message { get; init; }

    public FieldError() { }

    public FieldError(string field, string code, string message)
    {
        Field = field;
        Code = code;
        Message = message;
    }
}
