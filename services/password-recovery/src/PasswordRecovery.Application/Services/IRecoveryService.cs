using PasswordRecovery.Application.DTOs;
using PasswordRecovery.Domain.Common;

namespace PasswordRecovery.Application.Services;

public interface IRecoveryService
{
    Task<Result<RecoveryRequestResponse>> RequestRecoveryAsync(
        string email,
        string ipAddress,
        Guid correlationId,
        CancellationToken ct = default);

    Task<Result<TokenValidationResponse>> ValidateTokenAsync(
        string token,
        Guid correlationId,
        CancellationToken ct = default);

    Task<Result<PasswordResetResponse>> ResetPasswordAsync(
        string token,
        string newPassword,
        Guid correlationId,
        CancellationToken ct = default);
}
