using System.Text;
using System.Text.Json;
using LoggingService.Api.Models;
using LoggingService.Core.Interfaces;
using LoggingService.Core.Models;
using Microsoft.AspNetCore.Mvc;

namespace LoggingService.Api.Controllers;

/// <summary>
/// REST API controller for log ingestion and querying.
/// </summary>
[ApiController]
[Route("api/v1/logs")]
[Produces("application/json")]
public sealed class LogsController : ControllerBase
{
    private readonly ILogIngestionService _ingestionService;
    private readonly ILogRepository _repository;
    private readonly IAuditLogService _auditService;
    private readonly ILogger<LogsController> _logger;

    public LogsController(
        ILogIngestionService ingestionService,
        ILogRepository repository,
        IAuditLogService auditService,
        ILogger<LogsController> logger)
    {
        _ingestionService = ingestionService;
        _repository = repository;
        _auditService = auditService;
        _logger = logger;
    }

    /// <summary>
    /// Ingest a single log entry.
    /// </summary>
    [HttpPost]
    [ProducesResponseType(typeof(IngestResult), StatusCodes.Status202Accepted)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status400BadRequest)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status503ServiceUnavailable)]
    public async Task<IActionResult> IngestLog(
        [FromBody] LogEntryRequest request,
        CancellationToken ct)
    {
        var entry = request.ToLogEntry();
        var result = await _ingestionService.IngestAsync(entry, ct);

        return result.IsSuccess
            ? Accepted(result)
            : BadRequest(result.ToErrorResponse());
    }

    /// <summary>
    /// Ingest a batch of log entries (max 1000).
    /// </summary>
    [HttpPost("batch")]
    [ProducesResponseType(typeof(BatchIngestResult), StatusCodes.Status202Accepted)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status400BadRequest)]
    public async Task<IActionResult> IngestBatch(
        [FromBody] BatchLogEntryRequest request,
        CancellationToken ct)
    {
        if (request.Entries.Count > 1000)
        {
            return BadRequest(new ErrorResponse
            {
                Code = "BATCH_TOO_LARGE",
                Message = $"Batch size {request.Entries.Count} exceeds maximum of 1000"
            });
        }

        var entries = request.Entries.Select(e => e.ToLogEntry());
        var result = await _ingestionService.IngestBatchAsync(entries, ct);

        return Accepted(result);
    }

    /// <summary>
    /// Query logs with filters.
    /// </summary>
    [HttpGet]
    [ProducesResponseType(typeof(PagedResult<LogEntry>), StatusCodes.Status200OK)]
    public async Task<IActionResult> QueryLogs(
        [FromQuery] LogQueryRequest request,
        CancellationToken ct)
    {
        var query = request.ToLogQuery();
        var result = await _repository.QueryAsync(query, ct);

        await _auditService.LogQueryAsync(
            User.Identity?.Name ?? "anonymous",
            query,
            result.TotalCount,
            ct);

        if (result.TotalCount > 10000)
        {
            Response.Headers.Append("X-Warning", "Large result set. Consider narrowing your search.");
        }

        return Ok(result);
    }

    /// <summary>
    /// Get a specific log entry by ID.
    /// </summary>
    [HttpGet("{id}")]
    [ProducesResponseType(typeof(LogEntry), StatusCodes.Status200OK)]
    [ProducesResponseType(StatusCodes.Status404NotFound)]
    public async Task<IActionResult> GetLog(string id, CancellationToken ct)
    {
        var entry = await _repository.GetByIdAsync(id, ct);
        return entry != null ? Ok(entry) : NotFound();
    }

    /// <summary>
    /// Export logs as JSON or CSV.
    /// </summary>
    [HttpGet("export")]
    [ProducesResponseType(typeof(FileContentResult), StatusCodes.Status200OK)]
    public async Task<IActionResult> ExportLogs(
        [FromQuery] LogQueryRequest request,
        [FromQuery] string format = "json",
        CancellationToken ct = default)
    {
        var query = request.ToLogQuery() with { PageSize = 1000 };
        var result = await _repository.QueryAsync(query, ct);

        await _auditService.LogExportAsync(
            User.Identity?.Name ?? "anonymous",
            query,
            format,
            result.TotalCount,
            ct);

        return format.ToLowerInvariant() switch
        {
            "csv" => ExportAsCsv(result.Items),
            _ => ExportAsJson(result.Items)
        };
    }

    private FileContentResult ExportAsJson(IReadOnlyList<LogEntry> entries)
    {
        var json = JsonSerializer.SerializeToUtf8Bytes(entries, new JsonSerializerOptions
        {
            WriteIndented = true
        });
        return File(json, "application/json", "logs-export.json");
    }

    private FileContentResult ExportAsCsv(IReadOnlyList<LogEntry> entries)
    {
        var sb = new StringBuilder();
        sb.AppendLine("Id,Timestamp,CorrelationId,ServiceId,Level,Message");

        foreach (var entry in entries)
        {
            sb.AppendLine($"\"{entry.Id}\",\"{entry.Timestamp:O}\",\"{entry.CorrelationId}\",\"{entry.ServiceId}\",\"{entry.Level}\",\"{EscapeCsv(entry.Message)}\"");
        }

        return File(Encoding.UTF8.GetBytes(sb.ToString()), "text/csv", "logs-export.csv");
    }

    private static string EscapeCsv(string value) =>
        value.Replace("\"", "\"\"").Replace("\n", " ").Replace("\r", "");
}
