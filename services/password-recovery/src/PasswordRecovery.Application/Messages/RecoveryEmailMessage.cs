namespace PasswordRecovery.Application.Messages;

public record RecoveryEmailMessage(
    Guid CorrelationId,
    string RecipientEmail,
    string RecoveryLink,
    DateTime ExpiresAt,
    string? UserName
);

public record PasswordChangedEmailMessage(
    Guid CorrelationId,
    string RecipientEmail,
    string? UserName,
    DateTime ChangedAt
);
