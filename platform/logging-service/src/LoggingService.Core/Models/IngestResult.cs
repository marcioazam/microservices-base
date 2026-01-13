namespace LoggingService.Core.Models;

/// <summary>
/// Result of a log ingestion operation.
/// </summary>
public sealed record IngestResult
{
    /// <summary>
    /// Indicates whether the ingestion was successful.
    /// </summary>
    public bool IsSuccess { get; init; }

    /// <summary>
    /// ID of the ingested log entry (if successful).
    /// </summary>
    public string? LogId { get; init; }

    /// <summary>
    /// Error code if ingestion failed.
    /// </summary>
    public string? ErrorCode { get; init; }

    /// <summary>
    /// Error message if ingestion failed.
    /// </summary>
    public string? ErrorMessage { get; init; }

    /// <summary>
    /// Field-specific validation errors.
    /// </summary>
    public IReadOnlyList<FieldError>? FieldErrors { get; init; }

    /// <summary>
    /// Creates a successful ingestion result.
    /// </summary>
    public static IngestResult Success(string logId) =>
        new() { IsSuccess = true, LogId = logId };

    /// <summary>
    /// Creates a failed ingestion result due to validation errors.
    /// </summary>
    public static IngestResult ValidationFailed(IEnumerable<FieldError> errors) =>
        new()
        {
            IsSuccess = false,
            ErrorCode = "VALIDATION_FAILED",
            ErrorMessage = "Log entry validation failed",
            FieldErrors = errors.ToList()
        };

    /// <summary>
    /// Creates a failed ingestion result due to queue being full.
    /// </summary>
    public static IngestResult QueueFull() =>
        new()
        {
            IsSuccess = false,
            ErrorCode = "QUEUE_FULL",
            ErrorMessage = "Log queue is full. Please retry later."
        };

    /// <summary>
    /// Converts to an error response for API responses.
    /// </summary>
    public ErrorResponse ToErrorResponse() =>
        new()
        {
            Code = ErrorCode ?? "UNKNOWN_ERROR",
            Message = ErrorMessage ?? "An unknown error occurred",
            FieldErrors = FieldErrors
        };
}

/// <summary>
/// Result of a batch log ingestion operation.
/// </summary>
public sealed record BatchIngestResult
{
    /// <summary>
    /// Individual results for each log entry.
    /// </summary>
    public IReadOnlyList<IngestResult> Results { get; init; }

    /// <summary>
    /// Number of successfully ingested entries.
    /// </summary>
    public int SuccessCount { get; init; }

    /// <summary>
    /// Number of failed entries.
    /// </summary>
    public int FailedCount { get; init; }

    /// <summary>
    /// Indicates if the batch was too large.
    /// </summary>
    public bool IsBatchTooLarge { get; init; }

    public BatchIngestResult(IReadOnlyList<IngestResult> results, int successCount, int failedCount)
    {
        Results = results;
        SuccessCount = successCount;
        FailedCount = failedCount;
    }

    /// <summary>
    /// Creates a result indicating the batch was too large.
    /// </summary>
    public static BatchIngestResult BatchTooLarge(int actualSize, int maxSize) =>
        new([], 0, actualSize)
        {
            IsBatchTooLarge = true
        };
}
