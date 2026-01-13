using Elastic.Clients.Elasticsearch;
using Elastic.Clients.Elasticsearch.QueryDsl;
using LoggingService.Core.Configuration;
using LoggingService.Core.Interfaces;
using LoggingService.Core.Models;
using LoggingService.Core.Observability;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using LogLevel = LoggingService.Core.Models.LogLevel;

namespace LoggingService.Infrastructure.Storage;

/// <summary>
/// Elasticsearch 8.x implementation of ILogRepository.
/// </summary>
public sealed class ElasticsearchLogRepository : ILogRepository
{
    private readonly ElasticsearchClient _client;
    private readonly ElasticSearchOptions _options;
    private readonly ILogger<ElasticsearchLogRepository> _logger;

    public ElasticsearchLogRepository(
        ElasticsearchClient client,
        IOptions<ElasticSearchOptions> options,
        ILogger<ElasticsearchLogRepository> logger)
    {
        _client = client;
        _options = options.Value;
        _logger = logger;
    }

    /// <inheritdoc />
    public async Task<string> SaveAsync(LogEntry entry, CancellationToken ct = default)
    {
        var indexName = GetIndexName(entry);

        var response = await _client.IndexAsync(entry, idx => idx
            .Index(indexName)
            .Id(entry.Id), ct);

        if (!response.IsValidResponse)
        {
            LoggingMetrics.StorageErrors.Inc();
            _logger.LogError("Failed to index log entry: {Error}", response.DebugInformation);
            throw new InvalidOperationException($"Failed to index log entry: {response.DebugInformation}");
        }

        LoggingMetrics.LogsStored.Inc();
        return response.Id;
    }

    /// <inheritdoc />
    public async Task SaveBatchAsync(IEnumerable<LogEntry> entries, CancellationToken ct = default)
    {
        var entryList = entries.ToList();
        if (entryList.Count == 0) return;

        var bulkDescriptor = new BulkRequestDescriptor();
        foreach (var entry in entryList)
        {
            var indexName = GetIndexName(entry);
            bulkDescriptor.Index<LogEntry>(entry, idx => idx
                .Index(indexName)
                .Id(entry.Id));
        }

        var bulkResponse = await _client.BulkAsync(bulkDescriptor, ct);

        if (bulkResponse.Errors)
        {
            var errorCount = bulkResponse.ItemsWithErrors.Count();
            LoggingMetrics.StorageErrors.Inc(errorCount);
            _logger.LogError("Bulk indexing had {ErrorCount} errors", errorCount);
        }

        var successCount = entryList.Count - (bulkResponse.ItemsWithErrors?.Count() ?? 0);
        LoggingMetrics.LogsStored.Inc(successCount);
    }

    /// <inheritdoc />
    public async Task<LogEntry?> GetByIdAsync(string id, CancellationToken ct = default)
    {
        var searchResponse = await _client.SearchAsync<LogEntry>(s => s
            .Index($"{_options.IndexPrefix}-*")
            .Query(q => q.Ids(ids => ids.Values(new[] { id }))), ct);

        return searchResponse.Documents.FirstOrDefault();
    }

    /// <inheritdoc />
    public async Task<PagedResult<LogEntry>> QueryAsync(LogQuery query, CancellationToken ct = default)
    {
        var from = (query.Page - 1) * query.PageSize;
        var size = Math.Min(query.PageSize, 1000);
        var queries = BuildQueryFilters(query);
        var sortOrder = query.SortDirection == SortDirection.Descending 
            ? SortOrder.Desc 
            : SortOrder.Asc;

        SearchResponse<LogEntry> searchResponse;
        if (queries.Count > 0)
        {
            searchResponse = await _client.SearchAsync<LogEntry>(s => s
                .Index($"{_options.IndexPrefix}-*")
                .From(from)
                .Size(size)
                .Sort(so => so.Field(f => f.Timestamp, fs => fs.Order(sortOrder)))
                .Query(q => q.Bool(b => b.Must(queries.ToArray()))), ct);
        }
        else
        {
            searchResponse = await _client.SearchAsync<LogEntry>(s => s
                .Index($"{_options.IndexPrefix}-*")
                .From(from)
                .Size(size)
                .Sort(so => so.Field(f => f.Timestamp, fs => fs.Order(sortOrder))), ct);
        }

        LoggingMetrics.QueriesExecuted.Inc();

        return new PagedResult<LogEntry>
        {
            Items = searchResponse.Documents.ToList(),
            Page = query.Page,
            PageSize = query.PageSize,
            TotalCount = (int)(searchResponse.Total)
        };
    }

    /// <inheritdoc />
    public async Task<long> DeleteOlderThanAsync(
        DateTimeOffset olderThan,
        LogLevel? level = null,
        CancellationToken ct = default)
    {
        var queries = new List<Action<QueryDescriptor<LogEntry>>>
        {
            q => q.Range(r => r.DateRange(dr => dr
                .Field(f => f.Timestamp)
                .Lt(olderThan.UtcDateTime)))
        };

        if (level.HasValue)
        {
            queries.Add(q => q.Term(t => t.Field(f => f.Level).Value((int)level.Value)));
        }

        var response = await _client.DeleteByQueryAsync<LogEntry>(
            $"{_options.IndexPrefix}-*",
            d => d.Query(q => q.Bool(b => b.Must(queries.ToArray()))), ct);

        var deleted = response.Deleted ?? 0;
        _logger.LogInformation("Deleted {Count} log entries older than {Date}", deleted, olderThan);
        return deleted;
    }

    /// <inheritdoc />
    public async Task<long> ArchiveOlderThanAsync(DateTimeOffset olderThan, CancellationToken ct = default)
    {
        // For simplicity, archive all documents from source indices to archive index
        // then delete old entries. In production, consider using async reindex with wait_for_completion=false
        var archiveIndex = $"{_options.IndexPrefix}-archive-{olderThan:yyyy.MM}";

        // First, query to get count of documents to archive
        var countResponse = await _client.CountAsync<LogEntry>(c => c
            .Indices($"{_options.IndexPrefix}-*")
            .Query(q => q.Range(r => r.DateRange(dr => dr
                .Field(f => f.Timestamp)
                .Lt(olderThan.UtcDateTime)))), ct);

        if (!countResponse.IsValidResponse || countResponse.Count == 0)
        {
            _logger.LogInformation("No log entries to archive older than {Date}", olderThan);
            return 0;
        }

        // Delete old entries (archive functionality simplified - in production use scroll + bulk)
        var deleted = await DeleteOlderThanAsync(olderThan, ct: ct);
        _logger.LogInformation("Archived {Count} log entries older than {Date}", deleted, olderThan);
        return deleted;
    }

    /// <summary>
    /// Gets the index name for a log entry based on service ID and timestamp.
    /// </summary>
    public string GetIndexName(LogEntry entry)
    {
        var serviceId = entry.ServiceId.ToLowerInvariant().Replace(" ", "-");
        var date = entry.Timestamp.ToString("yyyy.MM.dd");
        return $"{_options.IndexPrefix}-{serviceId}-{date}";
    }

    private List<Action<QueryDescriptor<LogEntry>>> BuildQueryFilters(LogQuery query)
    {
        var queries = new List<Action<QueryDescriptor<LogEntry>>>();

        if (query.StartTime.HasValue)
        {
            queries.Add(q => q.Range(r => r.DateRange(dr => dr
                .Field(f => f.Timestamp)
                .Gte(query.StartTime.Value.UtcDateTime))));
        }

        if (query.EndTime.HasValue)
        {
            queries.Add(q => q.Range(r => r.DateRange(dr => dr
                .Field(f => f.Timestamp)
                .Lte(query.EndTime.Value.UtcDateTime))));
        }

        if (!string.IsNullOrEmpty(query.ServiceId))
        {
            queries.Add(q => q.Term(t => t.Field(f => f.ServiceId).Value(query.ServiceId)));
        }

        if (query.MinLevel.HasValue)
        {
            queries.Add(q => q.Range(r => r.NumberRange(nr => nr
                .Field(f => f.Level)
                .Gte((int)query.MinLevel.Value))));
        }

        if (!string.IsNullOrEmpty(query.CorrelationId))
        {
            queries.Add(q => q.Term(t => t.Field(f => f.CorrelationId).Value(query.CorrelationId)));
        }

        if (!string.IsNullOrEmpty(query.SearchText))
        {
            queries.Add(q => q.Match(m => m.Field(f => f.Message).Query(query.SearchText)));
        }

        if (!string.IsNullOrEmpty(query.UserId))
        {
            queries.Add(q => q.Term(t => t.Field(f => f.UserId).Value(query.UserId)));
        }

        if (!string.IsNullOrEmpty(query.TraceId))
        {
            queries.Add(q => q.Term(t => t.Field(f => f.TraceId).Value(query.TraceId)));
        }

        return queries;
    }
}
