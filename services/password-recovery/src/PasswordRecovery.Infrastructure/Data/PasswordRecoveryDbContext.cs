using Microsoft.EntityFrameworkCore;
using PasswordRecovery.Domain.Entities;

namespace PasswordRecovery.Infrastructure.Data;

public class PasswordRecoveryDbContext : DbContext
{
    public PasswordRecoveryDbContext(DbContextOptions<PasswordRecoveryDbContext> options)
        : base(options)
    {
    }

    public DbSet<RecoveryToken> RecoveryTokens => Set<RecoveryToken>();
    public DbSet<PasswordRecoveryAudit> AuditLogs => Set<PasswordRecoveryAudit>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        modelBuilder.ApplyConfiguration(new RecoveryTokenConfiguration());
        modelBuilder.ApplyConfiguration(new PasswordRecoveryAuditConfiguration());
        base.OnModelCreating(modelBuilder);
    }
}
