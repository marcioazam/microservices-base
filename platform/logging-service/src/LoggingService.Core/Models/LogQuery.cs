namespace LoggingService.Core.Models;

/// <summary>
/// Query parameters for searching log entries.
/// </summary>
public sealed record LogQuery
{
    /// <summary>
    /// Start of the time range filter.
    /// </summary>
    public DateTimeOffset? StartTime { get; init; }

    /// <summary>
    /// End of the time range filter.
    /// </summary>
    public DateTimeOffset? EndTime { get; init; }

    /// <summary>
    /// Filter by service ID.
    /// </summary>
    public string? ServiceId { get; init; }

    /// <summary>
    /// Minimum log level to include.
    /// </summary>
    public LogLevel? MinLevel { get; init; }

    /// <summary>
    /// Filter by correlation ID.
    /// </summary>
    public string? CorrelationId { get; init; }

    /// <summary>
    /// Full-text search on message field.
    /// </summary>
    public string? SearchText { get; init; }

    /// <summary>
    /// Filter by user ID.
    /// </summary>
    public string? UserId { get; init; }

    /// <summary>
    /// Filter by trace ID.
    /// </summary>
    public string? TraceId { get; init; }

    /// <summary>
    /// Page number (1-based).
    /// </summary>
    public int Page { get; init; } = 1;

    /// <summary>
    /// Number of items per page (max 1000).
    /// </summary>
    public int PageSize { get; init; } = 100;

    /// <summary>
    /// Sort direction for results.
    /// </summary>
    public SortDirection SortDirection { get; init; } = SortDirection.Descending;
}

/// <summary>
/// Sort direction for query results.
/// </summary>
public enum SortDirection
{
    Ascending,
    Descending
}

/// <summary>
/// Paged result container.
/// </summary>
/// <typeparam name="T">Type of items in the result.</typeparam>
public sealed record PagedResult<T>
{
    /// <summary>
    /// Items in the current page.
    /// </summary>
    public required IReadOnlyList<T> Items { get; init; }

    /// <summary>
    /// Total count of matching items across all pages.
    /// </summary>
    public required int TotalCount { get; init; }

    /// <summary>
    /// Current page number (1-based).
    /// </summary>
    public required int Page { get; init; }

    /// <summary>
    /// Number of items per page.
    /// </summary>
    public required int PageSize { get; init; }

    /// <summary>
    /// Indicates if there are more pages available.
    /// </summary>
    public bool HasMore => Page * PageSize < TotalCount;

    /// <summary>
    /// Total number of pages.
    /// </summary>
    public int TotalPages => (int)Math.Ceiling((double)TotalCount / PageSize);

    /// <summary>
    /// Creates an empty paged result.
    /// </summary>
    public static PagedResult<T> Empty(int page = 1, int pageSize = 100) =>
        new()
        {
            Items = [],
            TotalCount = 0,
            Page = page,
            PageSize = pageSize
        };
}
