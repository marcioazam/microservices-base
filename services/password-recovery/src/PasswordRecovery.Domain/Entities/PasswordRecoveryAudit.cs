namespace PasswordRecovery.Domain.Entities;

public class PasswordRecoveryAudit
{
    public Guid Id { get; set; }
    public string EventType { get; set; } = string.Empty;
    public Guid? UserId { get; set; }
    public string? Email { get; set; }
    public string? IpAddress { get; set; }
    public Guid CorrelationId { get; set; }
    public string? EventData { get; set; }
    public DateTime CreatedAt { get; set; }

    public static PasswordRecoveryAudit Create(
        string eventType,
        Guid correlationId,
        Guid? userId = null,
        string? email = null,
        string? ipAddress = null,
        string? eventData = null)
    {
        return new PasswordRecoveryAudit
        {
            Id = Guid.NewGuid(),
            EventType = eventType,
            UserId = userId,
            Email = email,
            IpAddress = ipAddress,
            CorrelationId = correlationId,
            EventData = eventData,
            CreatedAt = DateTime.UtcNow
        };
    }
}

public static class AuditEventTypes
{
    public const string RecoveryRequested = "RECOVERY_REQUESTED";
    public const string TokenValidated = "TOKEN_VALIDATED";
    public const string TokenValidationFailed = "TOKEN_VALIDATION_FAILED";
    public const string PasswordChanged = "PASSWORD_CHANGED";
    public const string RateLimitExceeded = "RATE_LIMIT_EXCEEDED";
}
