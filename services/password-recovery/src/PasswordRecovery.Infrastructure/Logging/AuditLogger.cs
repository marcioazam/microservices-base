using System.Text.Json;
using Microsoft.Extensions.Logging;
using PasswordRecovery.Application.Interfaces;
using PasswordRecovery.Domain.Entities;
using PasswordRecovery.Infrastructure.Data;

namespace PasswordRecovery.Infrastructure.Logging;

public class AuditLogger : IAuditLogger
{
    private readonly PasswordRecoveryDbContext _context;
    private readonly ILogger<AuditLogger> _logger;

    public AuditLogger(PasswordRecoveryDbContext context, ILogger<AuditLogger> logger)
    {
        _context = context;
        _logger = logger;
    }

    public async Task LogRecoveryRequestAsync(AuditEvent auditEvent, CancellationToken ct = default)
    {
        await LogEventAsync(auditEvent with { EventType = AuditEventTypes.RecoveryRequested }, ct);
    }

    public async Task LogTokenValidationAsync(AuditEvent auditEvent, CancellationToken ct = default)
    {
        await LogEventAsync(auditEvent with { EventType = AuditEventTypes.TokenValidated }, ct);
    }

    public async Task LogPasswordChangeAsync(AuditEvent auditEvent, CancellationToken ct = default)
    {
        await LogEventAsync(auditEvent with { EventType = AuditEventTypes.PasswordChanged }, ct);
    }

    private async Task LogEventAsync(AuditEvent auditEvent, CancellationToken ct)
    {
        var eventDataJson = auditEvent.EventData != null 
            ? JsonSerializer.Serialize(auditEvent.EventData) 
            : null;

        var audit = PasswordRecoveryAudit.Create(
            auditEvent.EventType,
            auditEvent.CorrelationId,
            auditEvent.UserId,
            SanitizeEmail(auditEvent.Email),
            auditEvent.IpAddress,
            eventDataJson);

        await _context.AuditLogs.AddAsync(audit, ct);
        await _context.SaveChangesAsync(ct);

        _logger.LogInformation(
            "Audit: {EventType} | CorrelationId: {CorrelationId} | UserId: {UserId}",
            auditEvent.EventType,
            auditEvent.CorrelationId,
            auditEvent.UserId);
    }

    private static string? SanitizeEmail(string? email)
    {
        if (string.IsNullOrEmpty(email)) return null;
        var parts = email.Split('@');
        if (parts.Length != 2) return "***";
        var localPart = parts[0].Length > 2 ? parts[0][..2] + "***" : "***";
        return $"{localPart}@{parts[1]}";
    }
}
