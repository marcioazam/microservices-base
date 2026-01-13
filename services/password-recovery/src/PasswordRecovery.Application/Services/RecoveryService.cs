using PasswordRecovery.Application.DTOs;
using PasswordRecovery.Application.Interfaces;
using PasswordRecovery.Application.Messages;
using PasswordRecovery.Domain.Common;
using PasswordRecovery.Domain.Entities;
using PasswordRecovery.Domain.ValueObjects;

namespace PasswordRecovery.Application.Services;

public class RecoveryService : IRecoveryService
{
    private readonly ITokenRepository _tokenRepository;
    private readonly IUserRepository _userRepository;
    private readonly ITokenGenerator _tokenGenerator;
    private readonly IPasswordHasher _passwordHasher;
    private readonly IEmailPublisher _emailPublisher;
    private readonly IRateLimiter _rateLimiter;
    private readonly RecoveryServiceOptions _options;

    public RecoveryService(
        ITokenRepository tokenRepository,
        IUserRepository userRepository,
        ITokenGenerator tokenGenerator,
        IPasswordHasher passwordHasher,
        IEmailPublisher emailPublisher,
        IRateLimiter rateLimiter,
        RecoveryServiceOptions options)
    {
        _tokenRepository = tokenRepository;
        _userRepository = userRepository;
        _tokenGenerator = tokenGenerator;
        _passwordHasher = passwordHasher;
        _emailPublisher = emailPublisher;
        _rateLimiter = rateLimiter;
        _options = options;
    }

    public async Task<Result<RecoveryRequestResponse>> RequestRecoveryAsync(
        string email,
        string ipAddress,
        Guid correlationId,
        CancellationToken ct = default)
    {
        var emailRateLimit = await _rateLimiter.CheckAsync($"email:{email}", 5, TimeSpan.FromHours(1), ct);
        if (!emailRateLimit.IsAllowed)
            return Result<RecoveryRequestResponse>.Failure("Too many requests. Please try again later.");

        var ipRateLimit = await _rateLimiter.CheckAsync($"ip:{ipAddress}", 10, TimeSpan.FromHours(1), ct);
        if (!ipRateLimit.IsAllowed)
            return Result<RecoveryRequestResponse>.Failure("Too many requests. Please try again later.");

        await _rateLimiter.IncrementAsync($"email:{email}", TimeSpan.FromHours(1), ct);
        await _rateLimiter.IncrementAsync($"ip:{ipAddress}", TimeSpan.FromHours(1), ct);

        var user = await _userRepository.GetByEmailAsync(email, ct);
        
        if (user != null)
        {
            await _tokenRepository.InvalidateUserTokensAsync(user.Id, ct);

            var token = _tokenGenerator.GenerateToken();
            var tokenHash = _tokenGenerator.HashToken(token);
            var recoveryToken = RecoveryToken.Create(
                user.Id,
                tokenHash,
                TimeSpan.FromMinutes(_options.TokenValidityMinutes),
                ipAddress);

            await _tokenRepository.CreateAsync(recoveryToken, ct);

            var recoveryLink = $"{_options.BaseUrl}/reset-password?token={Uri.EscapeDataString(token)}";
            var emailMessage = new RecoveryEmailMessage(
                correlationId,
                email,
                recoveryLink,
                recoveryToken.ExpiresAt,
                user.Name);

            await _emailPublisher.PublishRecoveryEmailAsync(emailMessage, ct);
        }

        return Result<RecoveryRequestResponse>.Success(new RecoveryRequestResponse(
            "If an account exists with this email, you will receive a password recovery link.",
            correlationId.ToString()));
    }

    public async Task<Result<TokenValidationResponse>> ValidateTokenAsync(
        string token,
        Guid correlationId,
        CancellationToken ct = default)
    {
        var tokenHash = _tokenGenerator.HashToken(token);
        
        var tokenRateLimit = await _rateLimiter.CheckAsync($"token:{tokenHash}", 5, TimeSpan.FromHours(1), ct);
        if (!tokenRateLimit.IsAllowed)
            return Result<TokenValidationResponse>.Failure("Too many attempts. Please request a new recovery link.");

        await _rateLimiter.IncrementAsync($"token:{tokenHash}", TimeSpan.FromHours(1), ct);

        var recoveryToken = await _tokenRepository.GetByHashAsync(tokenHash, ct);

        if (recoveryToken == null || !recoveryToken.IsValid)
            return Result<TokenValidationResponse>.Failure("Invalid or expired recovery link.");

        return Result<TokenValidationResponse>.Success(new TokenValidationResponse(
            true,
            token,
            correlationId.ToString()));
    }

    public async Task<Result<PasswordResetResponse>> ResetPasswordAsync(
        string token,
        string newPassword,
        Guid correlationId,
        CancellationToken ct = default)
    {
        var policy = PasswordPolicy.Default;
        var validationResult = policy.Validate(newPassword);
        if (!validationResult.IsSuccess)
            return Result<PasswordResetResponse>.ValidationFailure(validationResult.Errors);

        var tokenHash = _tokenGenerator.HashToken(token);
        var recoveryToken = await _tokenRepository.GetByHashAsync(tokenHash, ct);

        if (recoveryToken == null || !recoveryToken.IsValid)
            return Result<PasswordResetResponse>.Failure("Invalid or expired recovery link.");

        var user = await _userRepository.GetByIdAsync(recoveryToken.UserId, ct);
        if (user == null)
            return Result<PasswordResetResponse>.Failure("Invalid or expired recovery link.");

        var passwordHash = _passwordHasher.Hash(newPassword);
        await _userRepository.UpdatePasswordAsync(user.Id, passwordHash, ct);

        recoveryToken.MarkAsUsed();
        await _tokenRepository.UpdateAsync(recoveryToken, ct);

        var emailMessage = new PasswordChangedEmailMessage(
            correlationId,
            user.Email,
            user.Name,
            DateTime.UtcNow);

        await _emailPublisher.PublishPasswordChangedEmailAsync(emailMessage, ct);

        return Result<PasswordResetResponse>.Success(new PasswordResetResponse(
            true,
            "Your password has been successfully reset.",
            correlationId.ToString()));
    }
}

public class RecoveryServiceOptions
{
    public string BaseUrl { get; set; } = "https://example.com";
    public int TokenValidityMinutes { get; set; } = 15;
}
