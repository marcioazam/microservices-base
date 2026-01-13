using PasswordRecovery.Domain.Entities;

namespace PasswordRecovery.Application.Interfaces;

public interface ITokenRepository
{
    Task<RecoveryToken?> GetByHashAsync(string tokenHash, CancellationToken ct = default);
    Task<RecoveryToken?> GetByIdAsync(Guid id, CancellationToken ct = default);
    Task CreateAsync(RecoveryToken token, CancellationToken ct = default);
    Task UpdateAsync(RecoveryToken token, CancellationToken ct = default);
    Task InvalidateUserTokensAsync(Guid userId, CancellationToken ct = default);
    Task CleanupExpiredAsync(DateTime before, CancellationToken ct = default);
}
