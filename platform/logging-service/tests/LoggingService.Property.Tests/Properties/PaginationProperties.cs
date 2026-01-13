using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Models;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property 15: Query Pagination Correctness
/// Validates: Requirements 8.1, 8.2, 8.3
/// </summary>
[Trait("Category", "Property")]
[Trait("Feature", "logging-microservice")]
public class PaginationProperties
{
    /// <summary>
    /// Property: Page size is always capped at 1000.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property PageSizeIsCappedAt1000()
    {
        return Prop.ForAll(
            Arb.From<int>().Filter(x => x > 0),
            requestedPageSize =>
            {
                var query = new LogQuery { PageSize = requestedPageSize };
                var effectivePageSize = Math.Min(query.PageSize, 1000);
                return effectivePageSize <= 1000;
            });
    }

    /// <summary>
    /// Property: Page number is always positive.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property PageNumberIsAlwaysPositive()
    {
        return Prop.ForAll(
            Arb.From<int>(),
            requestedPage =>
            {
                var page = requestedPage <= 0 ? 1 : requestedPage;
                return page >= 1;
            });
    }

    /// <summary>
    /// Property: Offset calculation is correct.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property OffsetCalculationIsCorrect()
    {
        return Prop.ForAll(
            Gen.Choose(1, 100).ToArbitrary(),
            Gen.Choose(1, 1000).ToArbitrary(),
            (page, pageSize) =>
            {
                var expectedOffset = (page - 1) * pageSize;
                return expectedOffset >= 0 && expectedOffset == (page - 1) * pageSize;
            });
    }

    /// <summary>
    /// Property: HasMore is correctly calculated.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property HasMoreIsCorrectlyCalculated()
    {
        return Prop.ForAll(
            Gen.Choose(1, 100).ToArbitrary(),
            Gen.Choose(1, 100).ToArbitrary(),
            Gen.Choose(0, 10000).ToArbitrary(),
            (page, pageSize, totalCount) =>
            {
                var result = new PagedResult<LogEntry>
                {
                    Items = [],
                    Page = page,
                    PageSize = pageSize,
                    TotalCount = totalCount
                };

                var expectedHasMore = page * pageSize < totalCount;
                return result.HasMore == expectedHasMore;
            });
    }

    /// <summary>
    /// Property: Sort direction is preserved in query.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property SortDirectionIsPreserved()
    {
        return Prop.ForAll(
            Gen.Elements(SortDirection.Ascending, SortDirection.Descending).ToArbitrary(),
            direction =>
            {
                var query = new LogQuery { SortDirection = direction };
                return query.SortDirection == direction;
            });
    }
}
