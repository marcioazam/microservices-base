using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using PasswordRecovery.Domain.Entities;

namespace PasswordRecovery.Infrastructure.Data;

public class PasswordRecoveryAuditConfiguration : IEntityTypeConfiguration<PasswordRecoveryAudit>
{
    public void Configure(EntityTypeBuilder<PasswordRecoveryAudit> builder)
    {
        builder.ToTable("password_recovery_audit");

        builder.HasKey(a => a.Id);
        builder.Property(a => a.Id).HasColumnName("id");
        builder.Property(a => a.EventType).HasColumnName("event_type").HasMaxLength(50).IsRequired();
        builder.Property(a => a.UserId).HasColumnName("user_id");
        builder.Property(a => a.Email).HasColumnName("email").HasMaxLength(255);
        builder.Property(a => a.IpAddress).HasColumnName("ip_address").HasMaxLength(45);
        builder.Property(a => a.CorrelationId).HasColumnName("correlation_id").IsRequired();
        builder.Property(a => a.EventData).HasColumnName("event_data").HasColumnType("jsonb");
        builder.Property(a => a.CreatedAt).HasColumnName("created_at").IsRequired();

        builder.HasIndex(a => a.UserId);
        builder.HasIndex(a => a.CreatedAt);
        builder.HasIndex(a => a.CorrelationId);
    }
}
