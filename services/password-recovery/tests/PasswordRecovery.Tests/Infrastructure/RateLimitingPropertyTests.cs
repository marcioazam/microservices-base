using FsCheck;
using FsCheck.Xunit;
using Moq;
using PasswordRecovery.Application.Interfaces;
using PasswordRecovery.Application.Services;
using PasswordRecovery.Domain.Entities;

namespace PasswordRecovery.Tests.Infrastructure;

/// <summary>
/// Property 11: Rate Limiting Enforcement
/// Validates: Requirements 6.1, 6.2, 6.3, 6.4
/// </summary>
public class RateLimitingPropertyTests
{
    [Property(MaxTest = 10)]
    public bool EmailRateLimitRejectsAfterFiveRequests(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        // Simulate rate limit exceeded for email
        mockRateLimiter.Setup(x => x.CheckAsync(
            It.Is<string>(k => k.StartsWith("email:")), 
            It.IsAny<int>(), 
            It.IsAny<TimeSpan>(), 
            It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(false, 5, 5, TimeSpan.FromMinutes(30)));

        var options = new RecoveryServiceOptions { BaseUrl = "https://test.com", TokenValidityMinutes = 15 };
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        var result = service.RequestRecoveryAsync("user@test.com", "127.0.0.1", Guid.NewGuid()).Result;

        return !result.IsSuccess && result.Error!.Contains("Too many requests");
    }

    [Property(MaxTest = 10)]
    public bool IpRateLimitRejectsAfterTenRequests(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        // Email check passes
        mockRateLimiter.Setup(x => x.CheckAsync(
            It.Is<string>(k => k.StartsWith("email:")), 
            It.IsAny<int>(), 
            It.IsAny<TimeSpan>(), 
            It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(true, 1, 5, null));

        // IP check fails
        mockRateLimiter.Setup(x => x.CheckAsync(
            It.Is<string>(k => k.StartsWith("ip:")), 
            It.IsAny<int>(), 
            It.IsAny<TimeSpan>(), 
            It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(false, 10, 10, TimeSpan.FromMinutes(30)));

        var options = new RecoveryServiceOptions { BaseUrl = "https://test.com", TokenValidityMinutes = 15 };
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        var result = service.RequestRecoveryAsync("user@test.com", "127.0.0.1", Guid.NewGuid()).Result;

        return !result.IsSuccess && result.Error!.Contains("Too many requests");
    }

    [Property(MaxTest = 10)]
    public bool TokenValidationRateLimitRejectsAfterFiveAttempts(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-token");
        mockRateLimiter.Setup(x => x.CheckAsync(
            It.Is<string>(k => k.StartsWith("token:")), 
            It.IsAny<int>(), 
            It.IsAny<TimeSpan>(), 
            It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(false, 5, 5, TimeSpan.FromMinutes(30)));

        var options = new RecoveryServiceOptions();
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        var result = service.ValidateTokenAsync("test-token", Guid.NewGuid()).Result;

        return !result.IsSuccess && result.Error!.Contains("Too many attempts");
    }

    [Property(MaxTest = 10)]
    public bool RateLimitChecksCorrectLimits(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        int? emailLimit = null;
        int? ipLimit = null;

        mockRateLimiter.Setup(x => x.CheckAsync(
            It.Is<string>(k => k.StartsWith("email:")), 
            It.IsAny<int>(), 
            It.IsAny<TimeSpan>(), 
            It.IsAny<CancellationToken>()))
            .Callback<string, int, TimeSpan, CancellationToken>((_, limit, _, _) => emailLimit = limit)
            .ReturnsAsync(new RateLimitResult(true, 0, 5, null));

        mockRateLimiter.Setup(x => x.CheckAsync(
            It.Is<string>(k => k.StartsWith("ip:")), 
            It.IsAny<int>(), 
            It.IsAny<TimeSpan>(), 
            It.IsAny<CancellationToken>()))
            .Callback<string, int, TimeSpan, CancellationToken>((_, limit, _, _) => ipLimit = limit)
            .ReturnsAsync(new RateLimitResult(true, 0, 10, null));

        mockTokenGen.Setup(x => x.GenerateToken(It.IsAny<int>())).Returns("token");
        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed");
        mockUserRepo.Setup(x => x.GetByEmailAsync(It.IsAny<string>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((User?)null);

        var options = new RecoveryServiceOptions { BaseUrl = "https://test.com", TokenValidityMinutes = 15 };
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        service.RequestRecoveryAsync("user@test.com", "127.0.0.1", Guid.NewGuid()).Wait();

        return emailLimit == 5 && ipLimit == 10;
    }
}
