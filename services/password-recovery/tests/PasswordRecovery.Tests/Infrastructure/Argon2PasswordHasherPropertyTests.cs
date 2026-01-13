using FsCheck;
using FsCheck.Xunit;
using FluentAssertions;
using PasswordRecovery.Infrastructure.Security;

namespace PasswordRecovery.Tests.Infrastructure;

/// <summary>
/// Property 9: Password Hashing with Argon2id
/// Validates: Requirements 5.3
/// </summary>
public class Argon2PasswordHasherPropertyTests
{
    private readonly Argon2PasswordHasher _hasher = new();

    [Property(MaxTest = 10)]
    public bool HashAndVerifyRoundTrip(NonEmptyString password)
    {
        var pwd = password.Get;
        if (string.IsNullOrWhiteSpace(pwd)) return true; // Skip empty
        
        var hash = _hasher.Hash(pwd);
        var verified = _hasher.Verify(pwd, hash);
        return verified;
    }

    [Property(MaxTest = 10)]
    public bool DifferentPasswordsProduceDifferentHashes(NonEmptyString p1, NonEmptyString p2)
    {
        var pwd1 = p1.Get;
        var pwd2 = p2.Get;
        
        if (string.IsNullOrWhiteSpace(pwd1) || string.IsNullOrWhiteSpace(pwd2)) 
            return true; // Skip empty
        if (pwd1 == pwd2) 
            return true; // Same passwords - skip
        
        var hash1 = _hasher.Hash(pwd1);
        var hash2 = _hasher.Hash(pwd2);
        return hash1 != hash2;
    }

    [Property(MaxTest = 10)]
    public bool HashContainsArgon2Identifier(NonEmptyString password)
    {
        var pwd = password.Get;
        if (string.IsNullOrWhiteSpace(pwd)) return true; // Skip empty
        
        var hash = _hasher.Hash(pwd);
        return hash.StartsWith("$argon2");
    }

    [Property(MaxTest = 10)]
    public bool WrongPasswordDoesNotVerify(NonEmptyString original, NonEmptyString wrong)
    {
        var pwd1 = original.Get;
        var pwd2 = wrong.Get;
        
        if (string.IsNullOrWhiteSpace(pwd1) || string.IsNullOrWhiteSpace(pwd2)) 
            return true; // Skip empty
        if (pwd1 == pwd2) 
            return true; // Same passwords - skip
        
        var hash = _hasher.Hash(pwd1);
        var verified = _hasher.Verify(pwd2, hash);
        return !verified;
    }
}
