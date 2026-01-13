namespace PasswordRecovery.Application.Interfaces;

public interface IAuditLogger
{
    Task LogRecoveryRequestAsync(AuditEvent auditEvent, CancellationToken ct = default);
    Task LogTokenValidationAsync(AuditEvent auditEvent, CancellationToken ct = default);
    Task LogPasswordChangeAsync(AuditEvent auditEvent, CancellationToken ct = default);
}

public record AuditEvent(
    Guid CorrelationId,
    string EventType,
    Guid? UserId,
    string? Email,
    string? IpAddress,
    DateTime Timestamp,
    Dictionary<string, object>? EventData = null
);
