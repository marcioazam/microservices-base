using Microsoft.EntityFrameworkCore;
using PasswordRecovery.Application.Interfaces;
using PasswordRecovery.Domain.Entities;
using PasswordRecovery.Infrastructure.Data;

namespace PasswordRecovery.Infrastructure.Repositories;

public class TokenRepository : ITokenRepository
{
    private readonly PasswordRecoveryDbContext _context;

    public TokenRepository(PasswordRecoveryDbContext context)
    {
        _context = context;
    }

    public async Task<RecoveryToken?> GetByHashAsync(string tokenHash, CancellationToken ct = default)
    {
        return await _context.RecoveryTokens
            .FirstOrDefaultAsync(t => t.TokenHash == tokenHash, ct);
    }

    public async Task<RecoveryToken?> GetByIdAsync(Guid id, CancellationToken ct = default)
    {
        return await _context.RecoveryTokens.FindAsync([id], ct);
    }

    public async Task CreateAsync(RecoveryToken token, CancellationToken ct = default)
    {
        await _context.RecoveryTokens.AddAsync(token, ct);
        await _context.SaveChangesAsync(ct);
    }

    public async Task UpdateAsync(RecoveryToken token, CancellationToken ct = default)
    {
        _context.RecoveryTokens.Update(token);
        await _context.SaveChangesAsync(ct);
    }

    public async Task InvalidateUserTokensAsync(Guid userId, CancellationToken ct = default)
    {
        var tokens = await _context.RecoveryTokens
            .Where(t => t.UserId == userId && !t.IsUsed)
            .ToListAsync(ct);

        foreach (var token in tokens)
        {
            token.MarkAsUsed();
        }

        await _context.SaveChangesAsync(ct);
    }

    public async Task CleanupExpiredAsync(DateTime before, CancellationToken ct = default)
    {
        await _context.RecoveryTokens
            .Where(t => t.ExpiresAt < before && t.IsUsed)
            .ExecuteDeleteAsync(ct);
    }
}
