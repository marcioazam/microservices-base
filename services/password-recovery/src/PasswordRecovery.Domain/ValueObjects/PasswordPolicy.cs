namespace PasswordRecovery.Domain.ValueObjects;

public class PasswordPolicy
{
    public int MinLength { get; init; } = 12;
    public bool RequireUppercase { get; init; } = true;
    public bool RequireLowercase { get; init; } = true;
    public bool RequireDigit { get; init; } = true;
    public bool RequireSpecialChar { get; init; } = true;
    public string SpecialCharacters { get; init; } = "!@#$%^&*()_+-=[]{}|;:,.<>?";

    public static PasswordPolicy Default => new();

    public ValidationResult Validate(string password)
    {
        if (string.IsNullOrEmpty(password))
            return ValidationResult.Failure(["Password cannot be empty"]);

        var errors = new List<string>();

        if (password.Length < MinLength)
            errors.Add($"Password must be at least {MinLength} characters");

        if (RequireUppercase && !password.Any(char.IsUpper))
            errors.Add("Password must contain at least one uppercase letter");

        if (RequireLowercase && !password.Any(char.IsLower))
            errors.Add("Password must contain at least one lowercase letter");

        if (RequireDigit && !password.Any(char.IsDigit))
            errors.Add("Password must contain at least one digit");

        if (RequireSpecialChar && !password.Any(c => SpecialCharacters.Contains(c)))
            errors.Add("Password must contain at least one special character");

        return errors.Count == 0
            ? ValidationResult.Success()
            : ValidationResult.Failure(errors);
    }

    public bool IsValid(string password) => Validate(password).IsSuccess;
}

public class ValidationResult
{
    public bool IsSuccess { get; }
    public IReadOnlyList<string> Errors { get; }

    private ValidationResult(bool isSuccess, IReadOnlyList<string> errors)
    {
        IsSuccess = isSuccess;
        Errors = errors;
    }

    public static ValidationResult Success() => new(true, []);
    public static ValidationResult Failure(IEnumerable<string> errors) => new(false, errors.ToList());
    public static ValidationResult Failure(string error) => new(false, [error]);
}
