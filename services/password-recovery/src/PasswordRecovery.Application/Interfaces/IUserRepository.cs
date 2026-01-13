using PasswordRecovery.Domain.Entities;

namespace PasswordRecovery.Application.Interfaces;

public interface IUserRepository
{
    Task<User?> GetByEmailAsync(string email, CancellationToken ct = default);
    Task<User?> GetByIdAsync(Guid id, CancellationToken ct = default);
    Task UpdatePasswordAsync(Guid userId, string passwordHash, CancellationToken ct = default);
}
