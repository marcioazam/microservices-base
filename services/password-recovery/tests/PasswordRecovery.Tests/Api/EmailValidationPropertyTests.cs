using FsCheck;
using FsCheck.Xunit;
using FluentValidation;
using PasswordRecovery.Api.Validators;
using PasswordRecovery.Application.DTOs;

namespace PasswordRecovery.Tests.Api;

/// <summary>
/// Property 1: Email Format Validation
/// Validates: Requirements 1.1
/// </summary>
public class EmailValidationPropertyTests
{
    private readonly RecoveryRequestValidator _validator = new();

    [Property(MaxTest = 10)]
    public bool ValidEmailsAreAccepted(PositiveInt seed)
    {
        var validEmails = new[]
        {
            "test@example.com",
            "user.name@domain.org",
            "user+tag@example.co.uk",
            "firstname.lastname@company.com"
        };
        var email = validEmails[seed.Get % validEmails.Length];
        var request = new RecoveryRequest(email);
        var result = _validator.Validate(request);
        return result.IsValid;
    }

    [Property(MaxTest = 10)]
    public bool InvalidEmailsAreRejected(PositiveInt seed)
    {
        var invalidEmails = new[]
        {
            "notanemail",
            "@nodomain.com",
            "missing@"
        };
        var email = invalidEmails[seed.Get % invalidEmails.Length];
        var request = new RecoveryRequest(email);
        var result = _validator.Validate(request);
        return !result.IsValid;
    }

    [Property(MaxTest = 10)]
    public bool EmptyEmailIsRejected(PositiveInt seed)
    {
        var request = new RecoveryRequest("");
        var result = _validator.Validate(request);
        return !result.IsValid && result.Errors.Any(e => e.PropertyName == "Email");
    }
}
