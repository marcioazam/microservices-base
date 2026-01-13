using Microsoft.EntityFrameworkCore;
using PasswordRecovery.Application.Interfaces;
using PasswordRecovery.Domain.Entities;
using PasswordRecovery.Infrastructure.Data;

namespace PasswordRecovery.Infrastructure.Repositories;

public class UserRepository : IUserRepository
{
    private readonly PasswordRecoveryDbContext _context;

    public UserRepository(PasswordRecoveryDbContext context)
    {
        _context = context;
    }

    public async Task<User?> GetByEmailAsync(string email, CancellationToken ct = default)
    {
        return await _context.Set<User>()
            .FirstOrDefaultAsync(u => u.Email.ToLower() == email.ToLower() && u.IsActive, ct);
    }

    public async Task<User?> GetByIdAsync(Guid id, CancellationToken ct = default)
    {
        return await _context.Set<User>()
            .FirstOrDefaultAsync(u => u.Id == id && u.IsActive, ct);
    }

    public async Task UpdatePasswordAsync(Guid userId, string passwordHash, CancellationToken ct = default)
    {
        await _context.Set<User>()
            .Where(u => u.Id == userId)
            .ExecuteUpdateAsync(s => s
                .SetProperty(u => u.PasswordHash, passwordHash)
                .SetProperty(u => u.UpdatedAt, DateTime.UtcNow), ct);
    }
}
