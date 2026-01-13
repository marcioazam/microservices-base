using PasswordRecovery.Application.Messages;

namespace PasswordRecovery.Application.Interfaces;

public interface IEmailPublisher
{
    Task PublishRecoveryEmailAsync(RecoveryEmailMessage message, CancellationToken ct = default);
    Task PublishPasswordChangedEmailAsync(PasswordChangedEmailMessage message, CancellationToken ct = default);
}
