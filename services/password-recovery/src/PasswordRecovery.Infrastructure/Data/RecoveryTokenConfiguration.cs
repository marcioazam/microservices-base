using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using PasswordRecovery.Domain.Entities;

namespace PasswordRecovery.Infrastructure.Data;

public class RecoveryTokenConfiguration : IEntityTypeConfiguration<RecoveryToken>
{
    public void Configure(EntityTypeBuilder<RecoveryToken> builder)
    {
        builder.ToTable("recovery_tokens");

        builder.HasKey(t => t.Id);
        builder.Property(t => t.Id).HasColumnName("id");
        builder.Property(t => t.UserId).HasColumnName("user_id").IsRequired();
        builder.Property(t => t.TokenHash).HasColumnName("token_hash").HasMaxLength(64).IsRequired();
        builder.Property(t => t.CreatedAt).HasColumnName("created_at").IsRequired();
        builder.Property(t => t.ExpiresAt).HasColumnName("expires_at").IsRequired();
        builder.Property(t => t.IsUsed).HasColumnName("is_used").IsRequired();
        builder.Property(t => t.UsedAt).HasColumnName("used_at");
        builder.Property(t => t.IpAddress).HasColumnName("ip_address").HasMaxLength(45);

        builder.HasIndex(t => t.TokenHash).IsUnique();
        builder.HasIndex(t => t.UserId);
        builder.HasIndex(t => t.ExpiresAt);
    }
}
