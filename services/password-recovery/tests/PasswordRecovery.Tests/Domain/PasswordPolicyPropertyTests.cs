using FsCheck;
using FsCheck.Xunit;
using FluentAssertions;
using PasswordRecovery.Domain.ValueObjects;

namespace PasswordRecovery.Tests.Domain;

/// <summary>
/// Property 8: Password Policy Validation
/// Validates: Requirements 5.1, 5.2
/// </summary>
public class PasswordPolicyPropertyTests
{
    private readonly PasswordPolicy _policy = PasswordPolicy.Default;

    [Property(MaxTest = 20)]
    public bool ValidPasswordsPassAllRules(PositiveInt length)
    {
        // Generate a valid password with all required character types
        var actualLength = Math.Max(12, Math.Min(length.Get % 30 + 12, 40));
        var password = GenerateValidPassword(actualLength);
        
        var result = _policy.Validate(password);
        return result.IsSuccess;
    }

    [Property(MaxTest = 20)]
    public bool ShortPasswordsFail(PositiveInt seed)
    {
        // Generate password with length 1-11
        var length = (seed.Get % 11) + 1;
        var password = GeneratePasswordWithLength(length);
        
        var result = _policy.Validate(password);
        return !result.IsSuccess && result.Errors.Any(e => e.Contains("at least 12 characters"));
    }

    [Property(MaxTest = 20)]
    public bool PasswordsWithoutUppercaseFail(PositiveInt seed)
    {
        // Generate 12+ char password without uppercase
        var length = (seed.Get % 10) + 12;
        var chars = "abcdefghij1234567890!@#$%";
        var password = GenerateFromChars(chars, length);
        
        var result = _policy.Validate(password);
        var hasNoUpper = !password.Any(char.IsUpper);
        return !hasNoUpper || (!result.IsSuccess && result.Errors.Any(e => e.Contains("uppercase")));
    }


    [Property(MaxTest = 20)]
    public bool PasswordsWithoutLowercaseFail(PositiveInt seed)
    {
        // Generate 12+ char password without lowercase
        var length = (seed.Get % 10) + 12;
        var chars = "ABCDEFGHIJ1234567890!@#$%";
        var password = GenerateFromChars(chars, length);
        
        var result = _policy.Validate(password);
        var hasNoLower = !password.Any(char.IsLower);
        return !hasNoLower || (!result.IsSuccess && result.Errors.Any(e => e.Contains("lowercase")));
    }

    [Property(MaxTest = 20)]
    public bool PasswordsWithoutDigitFail(PositiveInt seed)
    {
        // Generate 12+ char password without digits
        var length = (seed.Get % 10) + 12;
        var chars = "ABCDEFGHIJabcdefghij!@#$%";
        var password = GenerateFromChars(chars, length);
        
        var result = _policy.Validate(password);
        var hasNoDigit = !password.Any(char.IsDigit);
        return !hasNoDigit || (!result.IsSuccess && result.Errors.Any(e => e.Contains("digit")));
    }

    [Property(MaxTest = 20)]
    public bool PasswordsWithoutSpecialCharFail(PositiveInt seed)
    {
        // Generate 12+ char password without special chars
        var length = (seed.Get % 10) + 12;
        var chars = "ABCDEFGHIJabcdefghij1234567890";
        var password = GenerateFromChars(chars, length);
        
        var result = _policy.Validate(password);
        var hasNoSpecial = !password.Any(c => _policy.SpecialCharacters.Contains(c));
        return !hasNoSpecial || (!result.IsSuccess && result.Errors.Any(e => e.Contains("special")));
    }

    private static string GenerateValidPassword(int length)
    {
        // Ensure we have at least one of each required type
        var required = "Aa1!";
        var filler = new string('x', Math.Max(0, length - 4));
        return required + filler;
    }

    private static string GeneratePasswordWithLength(int length)
    {
        return new string('A', Math.Max(1, length));
    }

    private static string GenerateFromChars(string chars, int length)
    {
        var random = new Random();
        return new string(Enumerable.Range(0, length)
            .Select(_ => chars[random.Next(chars.Length)])
            .ToArray());
    }
}
