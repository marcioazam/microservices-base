using FluentValidation;
using PasswordRecovery.Application.DTOs;

namespace PasswordRecovery.Api.Validators;

public class RecoveryRequestValidator : AbstractValidator<RecoveryRequest>
{
    public RecoveryRequestValidator()
    {
        RuleFor(x => x.Email)
            .NotEmpty().WithMessage("Email is required.")
            .EmailAddress().WithMessage("Invalid email format.");
    }
}

public class TokenValidationRequestValidator : AbstractValidator<TokenValidationRequest>
{
    public TokenValidationRequestValidator()
    {
        RuleFor(x => x.Token)
            .NotEmpty().WithMessage("Token is required.")
            .MinimumLength(20).WithMessage("Invalid token format.");
    }
}

public class PasswordResetRequestValidator : AbstractValidator<PasswordResetRequest>
{
    public PasswordResetRequestValidator()
    {
        RuleFor(x => x.Token)
            .NotEmpty().WithMessage("Token is required.");

        RuleFor(x => x.NewPassword)
            .NotEmpty().WithMessage("New password is required.")
            .MinimumLength(12).WithMessage("Password must be at least 12 characters.");

        RuleFor(x => x.ConfirmPassword)
            .NotEmpty().WithMessage("Confirm password is required.")
            .Equal(x => x.NewPassword).WithMessage("Passwords do not match.");
    }
}
