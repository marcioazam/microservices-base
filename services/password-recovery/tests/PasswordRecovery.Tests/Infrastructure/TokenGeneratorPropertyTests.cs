using FsCheck;
using FsCheck.Xunit;
using FluentAssertions;
using PasswordRecovery.Infrastructure.Security;

namespace PasswordRecovery.Tests.Infrastructure;

/// <summary>
/// Property 2: Token Generation Security
/// Validates: Requirements 2.1, 2.2, 2.3
/// </summary>
public class TokenGeneratorPropertyTests
{
    private readonly TokenGenerator _generator = new();

    [Property(MaxTest = 20)]
    public bool GeneratedTokensHaveMinimum32BytesEntropy(PositiveInt seed)
    {
        var byteLength = (seed.Get % 33) + 32; // 32-64 bytes
        var token = _generator.GenerateToken(byteLength);
        var tokenBytes = Convert.FromBase64String(token);
        return tokenBytes.Length >= 32;
    }

    [Property(MaxTest = 20)]
    public bool GeneratedTokensAreUnique(PositiveInt seed)
    {
        var tokens = Enumerable.Range(0, 10).Select(_ => _generator.GenerateToken()).ToList();
        var uniqueTokens = tokens.Distinct().Count();
        return uniqueTokens == 10;
    }

    [Property(MaxTest = 20)]
    public bool TokensHaveSufficientEntropy(PositiveInt seed)
    {
        var byteLength = (seed.Get % 33) + 32; // 32-64 bytes
        var token = _generator.GenerateToken(byteLength);
        var tokenBytes = Convert.FromBase64String(token);
        var uniqueBytes = tokenBytes.Distinct().Count();
        // At least 25% unique bytes indicates good entropy
        return uniqueBytes > byteLength / 4;
    }

    [Property(MaxTest = 20)]
    public bool HashedTokensAreConsistent(PositiveInt seed)
    {
        var token = _generator.GenerateToken();
        var hash1 = _generator.HashToken(token);
        var hash2 = _generator.HashToken(token);
        return hash1 == hash2;
    }

    [Property(MaxTest = 20)]
    public bool HashedTokensAre64HexCharacters(PositiveInt seed)
    {
        var token = _generator.GenerateToken();
        var hash = _generator.HashToken(token);
        return hash.Length == 64 && hash.All(c => "0123456789abcdef".Contains(c));
    }
}

/// <summary>
/// Property 3: Token Storage Security (Hashing)
/// Validates: Requirements 2.4
/// </summary>
public class TokenStorageSecurityPropertyTests
{
    private readonly TokenGenerator _generator = new();

    [Property(MaxTest = 20)]
    public bool DifferentTokensProduceDifferentHashes(PositiveInt seed)
    {
        var token1 = _generator.GenerateToken();
        var token2 = _generator.GenerateToken();
        var hash1 = _generator.HashToken(token1);
        var hash2 = _generator.HashToken(token2);
        return hash1 != hash2;
    }

    [Property(MaxTest = 20)]
    public bool HashCannotBeReversedToOriginalToken(PositiveInt seed)
    {
        var token = _generator.GenerateToken();
        var hash = _generator.HashToken(token);
        // Hash should not contain the original token and vice versa
        return !hash.Contains(token) && !token.Contains(hash);
    }
}
