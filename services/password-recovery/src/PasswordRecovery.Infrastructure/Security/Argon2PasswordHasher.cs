using System.Security.Cryptography;
using System.Text;
using Isopoh.Cryptography.Argon2;
using Isopoh.Cryptography.SecureArray;
using PasswordRecovery.Application.Interfaces;

namespace PasswordRecovery.Infrastructure.Security;

public class Argon2PasswordHasher : IPasswordHasher
{
    private const int SaltSize = 16;
    private const int HashLength = 32;
    private const int MemoryCost = 65536; // 64 MB
    private const int TimeCost = 3;
    private const int Lanes = 4;

    public string Hash(string password)
    {
        if (string.IsNullOrEmpty(password))
            throw new ArgumentException("Password cannot be empty.", nameof(password));

        byte[] passwordBytes = Encoding.UTF8.GetBytes(password);
        byte[] salt = new byte[SaltSize];
        RandomNumberGenerator.Fill(salt);

        var config = new Argon2Config
        {
            Type = Argon2Type.HybridAddressing, // Argon2id
            Version = Argon2Version.Nineteen,
            MemoryCost = MemoryCost,
            TimeCost = TimeCost,
            Lanes = Lanes,
            Threads = Lanes,
            Password = passwordBytes,
            Salt = salt,
            HashLength = HashLength
        };

        using var argon2 = new Argon2(config);
        using SecureArray<byte> hashBytes = argon2.Hash();
        return config.EncodeString(hashBytes.Buffer);
    }

    public bool Verify(string password, string hash)
    {
        if (string.IsNullOrEmpty(password) || string.IsNullOrEmpty(hash))
            return false;

        try
        {
            byte[] passwordBytes = Encoding.UTF8.GetBytes(password);
            return Argon2.Verify(hash, passwordBytes, Lanes);
        }
        catch
        {
            return false;
        }
    }
}
