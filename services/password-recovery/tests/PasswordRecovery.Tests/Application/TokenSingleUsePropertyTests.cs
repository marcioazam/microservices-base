using FsCheck;
using FsCheck.Xunit;
using Moq;
using PasswordRecovery.Application.Interfaces;
using PasswordRecovery.Application.Services;
using PasswordRecovery.Domain.Entities;

namespace PasswordRecovery.Tests.Application;

/// <summary>
/// Property 10: Token Single-Use Enforcement
/// Validates: Requirements 5.5
/// </summary>
public class TokenSingleUsePropertyTests
{
    [Property(MaxTest = 10)]
    public bool TokenIsMarkedAsUsedAfterSuccessfulReset(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        var userId = Guid.NewGuid();
        var validToken = RecoveryToken.Create(userId, "hashed-token", TimeSpan.FromMinutes(15), "127.0.0.1");
        RecoveryToken? updatedToken = null;

        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-token");
        mockTokenRepo.Setup(x => x.GetByHashAsync("hashed-token", It.IsAny<CancellationToken>()))
            .ReturnsAsync(validToken);
        mockTokenRepo.Setup(x => x.UpdateAsync(It.IsAny<RecoveryToken>(), It.IsAny<CancellationToken>()))
            .Callback<RecoveryToken, CancellationToken>((t, _) => updatedToken = t)
            .Returns(Task.CompletedTask);
        mockUserRepo.Setup(x => x.GetByIdAsync(userId, It.IsAny<CancellationToken>()))
            .ReturnsAsync(new User { Id = userId, Email = "user@test.com", Name = "Test" });
        mockHasher.Setup(x => x.Hash(It.IsAny<string>())).Returns("hashed-password");

        var options = new RecoveryServiceOptions();
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        var result = service.ResetPasswordAsync("test-token", "ValidP@ssw0rd123!", Guid.NewGuid()).Result;

        return result.IsSuccess && 
               updatedToken != null && 
               updatedToken.IsUsed;
    }

    [Property(MaxTest = 10)]
    public bool UsedTokenCannotBeReused(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        var userId = Guid.NewGuid();
        var usedToken = RecoveryToken.Create(userId, "hashed-token", TimeSpan.FromMinutes(15), "127.0.0.1");
        usedToken.MarkAsUsed();

        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-token");
        mockTokenRepo.Setup(x => x.GetByHashAsync("hashed-token", It.IsAny<CancellationToken>()))
            .ReturnsAsync(usedToken);

        var options = new RecoveryServiceOptions();
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        var result = service.ResetPasswordAsync("test-token", "ValidP@ssw0rd123!", Guid.NewGuid()).Result;

        return !result.IsSuccess && result.Error!.Contains("Invalid or expired");
    }

    [Property(MaxTest = 10)]
    public bool TokenUsedAtTimestampIsSet(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        var userId = Guid.NewGuid();
        var validToken = RecoveryToken.Create(userId, "hashed-token", TimeSpan.FromMinutes(15), "127.0.0.1");
        RecoveryToken? updatedToken = null;
        var beforeReset = DateTime.UtcNow;

        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-token");
        mockTokenRepo.Setup(x => x.GetByHashAsync("hashed-token", It.IsAny<CancellationToken>()))
            .ReturnsAsync(validToken);
        mockTokenRepo.Setup(x => x.UpdateAsync(It.IsAny<RecoveryToken>(), It.IsAny<CancellationToken>()))
            .Callback<RecoveryToken, CancellationToken>((t, _) => updatedToken = t)
            .Returns(Task.CompletedTask);
        mockUserRepo.Setup(x => x.GetByIdAsync(userId, It.IsAny<CancellationToken>()))
            .ReturnsAsync(new User { Id = userId, Email = "user@test.com", Name = "Test" });
        mockHasher.Setup(x => x.Hash(It.IsAny<string>())).Returns("hashed-password");

        var options = new RecoveryServiceOptions();
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        service.ResetPasswordAsync("test-token", "ValidP@ssw0rd123!", Guid.NewGuid()).Wait();
        var afterReset = DateTime.UtcNow;

        return updatedToken != null && 
               updatedToken.UsedAt.HasValue &&
               updatedToken.UsedAt.Value >= beforeReset &&
               updatedToken.UsedAt.Value <= afterReset;
    }
}
