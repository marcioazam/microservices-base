using System.Security.Cryptography;
using PasswordRecovery.Application.Interfaces;

namespace PasswordRecovery.Infrastructure.Security;

public class TokenGenerator : ITokenGenerator
{
    private const int DefaultByteLength = 32;

    public string GenerateToken(int byteLength = DefaultByteLength)
    {
        if (byteLength < 32)
            throw new ArgumentException("Token must be at least 32 bytes for security.", nameof(byteLength));

        var tokenBytes = new byte[byteLength];
        RandomNumberGenerator.Fill(tokenBytes);
        return Convert.ToBase64String(tokenBytes);
    }

    public string HashToken(string token)
    {
        if (string.IsNullOrEmpty(token))
            throw new ArgumentException("Token cannot be empty.", nameof(token));

        var tokenBytes = Convert.FromBase64String(token);
        var hashBytes = SHA256.HashData(tokenBytes);
        return Convert.ToHexString(hashBytes).ToLowerInvariant();
    }
}
