namespace PasswordRecovery.Application.Interfaces;

public interface ITokenGenerator
{
    string GenerateToken(int byteLength = 32);
    string HashToken(string token);
}
