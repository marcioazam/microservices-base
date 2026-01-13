namespace PasswordRecovery.Domain.Entities;

public class RecoveryToken
{
    public Guid Id { get; private set; }
    public Guid UserId { get; private set; }
    public string TokenHash { get; private set; } = string.Empty;
    public DateTime CreatedAt { get; private set; }
    public DateTime ExpiresAt { get; private set; }
    public bool IsUsed { get; private set; }
    public DateTime? UsedAt { get; private set; }
    public string? IpAddress { get; private set; }

    private RecoveryToken() { }

    public bool IsValid => !IsUsed && DateTime.UtcNow < ExpiresAt;
    public bool IsExpired => DateTime.UtcNow >= ExpiresAt;

    public void MarkAsUsed()
    {
        if (IsUsed)
            throw new InvalidOperationException("Token has already been used.");
        
        IsUsed = true;
        UsedAt = DateTime.UtcNow;
    }

    public static RecoveryToken Create(
        Guid userId,
        string tokenHash,
        TimeSpan validity,
        string? ipAddress = null)
    {
        if (userId == Guid.Empty)
            throw new ArgumentException("UserId cannot be empty.", nameof(userId));
        
        if (string.IsNullOrWhiteSpace(tokenHash))
            throw new ArgumentException("TokenHash cannot be empty.", nameof(tokenHash));
        
        if (validity <= TimeSpan.Zero)
            throw new ArgumentException("Validity must be positive.", nameof(validity));

        var now = DateTime.UtcNow;
        return new RecoveryToken
        {
            Id = Guid.NewGuid(),
            UserId = userId,
            TokenHash = tokenHash,
            CreatedAt = now,
            ExpiresAt = now.Add(validity),
            IsUsed = false,
            IpAddress = ipAddress
        };
    }
}
