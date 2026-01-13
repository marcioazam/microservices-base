using FsCheck;
using FsCheck.Xunit;
using Moq;
using PasswordRecovery.Application.Interfaces;
using PasswordRecovery.Application.Services;
using PasswordRecovery.Domain.Entities;

namespace PasswordRecovery.Tests.Infrastructure;

/// <summary>
/// Property 4: Token Invalidation on New Request
/// Validates: Requirements 2.5
/// </summary>
public class TokenInvalidationPropertyTests
{
    [Property(MaxTest = 10)]
    public bool NewTokenInvalidatesPreviousTokens(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        var userId = Guid.NewGuid();
        var invalidateCalled = false;

        mockRateLimiter.Setup(x => x.CheckAsync(It.IsAny<string>(), It.IsAny<int>(), It.IsAny<TimeSpan>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(true, 0, 5, null));
        mockTokenGen.Setup(x => x.GenerateToken(It.IsAny<int>())).Returns("new-token");
        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-new-token");
        mockUserRepo.Setup(x => x.GetByEmailAsync("user@test.com", It.IsAny<CancellationToken>()))
            .ReturnsAsync(new User { Id = userId, Email = "user@test.com", Name = "Test" });
        mockTokenRepo.Setup(x => x.InvalidateUserTokensAsync(userId, It.IsAny<CancellationToken>()))
            .Callback(() => invalidateCalled = true)
            .Returns(Task.CompletedTask);

        var options = new RecoveryServiceOptions { BaseUrl = "https://test.com", TokenValidityMinutes = 15 };
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        var result = service.RequestRecoveryAsync("user@test.com", "127.0.0.1", Guid.NewGuid()).Result;

        return result.IsSuccess && invalidateCalled;
    }

    [Property(MaxTest = 10)]
    public bool InvalidateIsCalledBeforeNewTokenCreation(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        var userId = Guid.NewGuid();
        var callOrder = new List<string>();

        mockRateLimiter.Setup(x => x.CheckAsync(It.IsAny<string>(), It.IsAny<int>(), It.IsAny<TimeSpan>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(true, 0, 5, null));
        mockTokenGen.Setup(x => x.GenerateToken(It.IsAny<int>())).Returns("new-token");
        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-new-token");
        mockUserRepo.Setup(x => x.GetByEmailAsync("user@test.com", It.IsAny<CancellationToken>()))
            .ReturnsAsync(new User { Id = userId, Email = "user@test.com", Name = "Test" });
        mockTokenRepo.Setup(x => x.InvalidateUserTokensAsync(userId, It.IsAny<CancellationToken>()))
            .Callback(() => callOrder.Add("invalidate"))
            .Returns(Task.CompletedTask);
        mockTokenRepo.Setup(x => x.CreateAsync(It.IsAny<RecoveryToken>(), It.IsAny<CancellationToken>()))
            .Callback(() => callOrder.Add("create"))
            .Returns(Task.CompletedTask);

        var options = new RecoveryServiceOptions { BaseUrl = "https://test.com", TokenValidityMinutes = 15 };
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        service.RequestRecoveryAsync("user@test.com", "127.0.0.1", Guid.NewGuid()).Wait();

        return callOrder.Count == 2 && 
               callOrder[0] == "invalidate" && 
               callOrder[1] == "create";
    }
}
