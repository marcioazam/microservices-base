namespace LoggingService.Core.Models;

/// <summary>
/// Standard error response model for API errors.
/// </summary>
public sealed record ErrorResponse
{
    /// <summary>
    /// Error code for programmatic handling.
    /// </summary>
    public required string Code { get; init; }

    /// <summary>
    /// Human-readable error message.
    /// </summary>
    public required string Message { get; init; }

    /// <summary>
    /// Optional correlation ID for tracing.
    /// </summary>
    public string? CorrelationId { get; init; }

    /// <summary>
    /// Optional field-level errors for validation failures.
    /// </summary>
    public IReadOnlyList<FieldError>? FieldErrors { get; init; }

    /// <summary>
    /// Creates an error response from a validation result.
    /// </summary>
    public static ErrorResponse FromValidation(ValidationResult result, string? correlationId = null)
    {
        return new ErrorResponse
        {
            Code = "VALIDATION_ERROR",
            Message = "One or more validation errors occurred",
            CorrelationId = correlationId,
            FieldErrors = result.Errors
        };
    }

    /// <summary>
    /// Creates a batch too large error response.
    /// </summary>
    public static ErrorResponse BatchTooLarge(int count, int max = 1000)
    {
        return new ErrorResponse
        {
            Code = "BATCH_TOO_LARGE",
            Message = $"Batch size {count} exceeds maximum of {max}"
        };
    }

    /// <summary>
    /// Creates a service unavailable error response.
    /// </summary>
    public static ErrorResponse ServiceUnavailable(string service)
    {
        return new ErrorResponse
        {
            Code = "SERVICE_UNAVAILABLE",
            Message = $"{service} is currently unavailable"
        };
    }
}
