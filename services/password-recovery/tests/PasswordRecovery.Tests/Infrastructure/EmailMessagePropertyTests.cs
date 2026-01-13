using FsCheck;
using FsCheck.Xunit;
using Moq;
using PasswordRecovery.Application.Interfaces;
using PasswordRecovery.Application.Messages;
using PasswordRecovery.Application.Services;
using PasswordRecovery.Domain.Entities;

namespace PasswordRecovery.Tests.Infrastructure;

/// <summary>
/// Property 6: Email Message Completeness
/// Validates: Requirements 3.2, 3.3
/// </summary>
public class EmailMessagePropertyTests
{
    [Property(MaxTest = 10)]
    public bool EmailMessageContainsRecoveryLink(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        RecoveryEmailMessage? capturedMessage = null;
        var userId = Guid.NewGuid();

        mockRateLimiter.Setup(x => x.CheckAsync(It.IsAny<string>(), It.IsAny<int>(), It.IsAny<TimeSpan>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(true, 0, 5, null));
        mockTokenGen.Setup(x => x.GenerateToken(It.IsAny<int>())).Returns("test-recovery-token");
        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-token");
        mockUserRepo.Setup(x => x.GetByEmailAsync("user@test.com", It.IsAny<CancellationToken>()))
            .ReturnsAsync(new User { Id = userId, Email = "user@test.com", Name = "Test User" });
        mockEmail.Setup(x => x.PublishRecoveryEmailAsync(It.IsAny<RecoveryEmailMessage>(), It.IsAny<CancellationToken>()))
            .Callback<RecoveryEmailMessage, CancellationToken>((msg, _) => capturedMessage = msg)
            .Returns(Task.CompletedTask);

        var options = new RecoveryServiceOptions { BaseUrl = "https://test.com", TokenValidityMinutes = 15 };
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        service.RequestRecoveryAsync("user@test.com", "127.0.0.1", Guid.NewGuid()).Wait();

        return capturedMessage != null &&
               !string.IsNullOrEmpty(capturedMessage.RecoveryLink) &&
               capturedMessage.RecoveryLink.Contains("token=");
    }

    [Property(MaxTest = 10)]
    public bool EmailMessageContainsExpirationTime(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        RecoveryEmailMessage? capturedMessage = null;
        var userId = Guid.NewGuid();

        mockRateLimiter.Setup(x => x.CheckAsync(It.IsAny<string>(), It.IsAny<int>(), It.IsAny<TimeSpan>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(true, 0, 5, null));
        mockTokenGen.Setup(x => x.GenerateToken(It.IsAny<int>())).Returns("test-token");
        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-token");
        mockUserRepo.Setup(x => x.GetByEmailAsync("user@test.com", It.IsAny<CancellationToken>()))
            .ReturnsAsync(new User { Id = userId, Email = "user@test.com", Name = "Test User" });
        mockEmail.Setup(x => x.PublishRecoveryEmailAsync(It.IsAny<RecoveryEmailMessage>(), It.IsAny<CancellationToken>()))
            .Callback<RecoveryEmailMessage, CancellationToken>((msg, _) => capturedMessage = msg)
            .Returns(Task.CompletedTask);

        var options = new RecoveryServiceOptions { BaseUrl = "https://test.com", TokenValidityMinutes = 15 };
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        service.RequestRecoveryAsync("user@test.com", "127.0.0.1", Guid.NewGuid()).Wait();

        return capturedMessage != null &&
               capturedMessage.ExpiresAt > DateTime.UtcNow &&
               capturedMessage.ExpiresAt <= DateTime.UtcNow.AddMinutes(20);
    }

    [Property(MaxTest = 10)]
    public bool EmailMessageContainsRecipientEmail(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        RecoveryEmailMessage? capturedMessage = null;
        var userId = Guid.NewGuid();
        var testEmail = "user@test.com";

        mockRateLimiter.Setup(x => x.CheckAsync(It.IsAny<string>(), It.IsAny<int>(), It.IsAny<TimeSpan>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(true, 0, 5, null));
        mockTokenGen.Setup(x => x.GenerateToken(It.IsAny<int>())).Returns("test-token");
        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-token");
        mockUserRepo.Setup(x => x.GetByEmailAsync(testEmail, It.IsAny<CancellationToken>()))
            .ReturnsAsync(new User { Id = userId, Email = testEmail, Name = "Test User" });
        mockEmail.Setup(x => x.PublishRecoveryEmailAsync(It.IsAny<RecoveryEmailMessage>(), It.IsAny<CancellationToken>()))
            .Callback<RecoveryEmailMessage, CancellationToken>((msg, _) => capturedMessage = msg)
            .Returns(Task.CompletedTask);

        var options = new RecoveryServiceOptions { BaseUrl = "https://test.com", TokenValidityMinutes = 15 };
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        service.RequestRecoveryAsync(testEmail, "127.0.0.1", Guid.NewGuid()).Wait();

        return capturedMessage != null &&
               capturedMessage.RecipientEmail == testEmail;
    }
}
