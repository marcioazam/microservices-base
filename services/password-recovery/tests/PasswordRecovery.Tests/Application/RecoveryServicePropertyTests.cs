using FsCheck;
using FsCheck.Xunit;
using Moq;
using PasswordRecovery.Application.Interfaces;
using PasswordRecovery.Application.Services;
using PasswordRecovery.Domain.Entities;

namespace PasswordRecovery.Tests.Application;

/// <summary>
/// Property 5: Response Uniformity (Email Enumeration Prevention)
/// Validates: Requirements 1.5, 1.6
/// </summary>
public class ResponseUniformityPropertyTests
{
    [Property(MaxTest = 10)]
    public bool ResponseMessageIsIdenticalForExistingAndNonExistingEmails(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        mockRateLimiter.Setup(x => x.CheckAsync(It.IsAny<string>(), It.IsAny<int>(), It.IsAny<TimeSpan>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(true, 0, 5, null));
        mockTokenGen.Setup(x => x.GenerateToken(It.IsAny<int>())).Returns("test-token");
        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-token");

        var options = new RecoveryServiceOptions { BaseUrl = "https://test.com", TokenValidityMinutes = 15 };
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object, 
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        // Test with existing user
        mockUserRepo.Setup(x => x.GetByEmailAsync("existing@test.com", It.IsAny<CancellationToken>()))
            .ReturnsAsync(new User { Id = Guid.NewGuid(), Email = "existing@test.com", Name = "Test" });
        var existingResult = service.RequestRecoveryAsync("existing@test.com", "127.0.0.1", Guid.NewGuid()).Result;

        // Test with non-existing user
        mockUserRepo.Setup(x => x.GetByEmailAsync("nonexisting@test.com", It.IsAny<CancellationToken>()))
            .ReturnsAsync((User?)null);
        var nonExistingResult = service.RequestRecoveryAsync("nonexisting@test.com", "127.0.0.1", Guid.NewGuid()).Result;

        return existingResult.IsSuccess == nonExistingResult.IsSuccess &&
               existingResult.Value?.Message == nonExistingResult.Value?.Message;
    }
}

/// <summary>
/// Property 7: Token Validation Correctness
/// Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5
/// </summary>
public class TokenValidationCorrectnessPropertyTests
{
    [Property(MaxTest = 10)]
    public bool ValidTokenSucceeds(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        mockRateLimiter.Setup(x => x.CheckAsync(It.IsAny<string>(), It.IsAny<int>(), It.IsAny<TimeSpan>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(true, 0, 5, null));
        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-token");

        var validToken = RecoveryToken.Create(Guid.NewGuid(), "hashed-token", TimeSpan.FromMinutes(15), "127.0.0.1");
        mockTokenRepo.Setup(x => x.GetByHashAsync("hashed-token", It.IsAny<CancellationToken>()))
            .ReturnsAsync(validToken);

        var options = new RecoveryServiceOptions();
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        var result = service.ValidateTokenAsync("test-token", Guid.NewGuid()).Result;
        return result.IsSuccess && result.Value?.IsValid == true;
    }

    [Property(MaxTest = 10)]
    public bool ExpiredTokenFails(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        mockRateLimiter.Setup(x => x.CheckAsync(It.IsAny<string>(), It.IsAny<int>(), It.IsAny<TimeSpan>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(true, 0, 5, null));
        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-token");

        // Create a valid token first, then simulate it being expired via mock
        var expiredToken = CreateExpiredToken(Guid.NewGuid(), "hashed-token");
        mockTokenRepo.Setup(x => x.GetByHashAsync("hashed-token", It.IsAny<CancellationToken>()))
            .ReturnsAsync(expiredToken);

        var options = new RecoveryServiceOptions();
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        var result = service.ValidateTokenAsync("test-token", Guid.NewGuid()).Result;
        return !result.IsSuccess;
    }

    private static RecoveryToken CreateExpiredToken(Guid userId, string tokenHash)
    {
        // Use reflection to create an expired token for testing
        var token = RecoveryToken.Create(userId, tokenHash, TimeSpan.FromMinutes(1), "127.0.0.1");
        var expiresAtField = typeof(RecoveryToken).GetProperty("ExpiresAt", 
            System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Public);
        
        // Use reflection to set ExpiresAt to past
        var backingField = typeof(RecoveryToken).GetField("<ExpiresAt>k__BackingField", 
            System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.NonPublic);
        backingField?.SetValue(token, DateTime.UtcNow.AddMinutes(-10));
        
        return token;
    }

    [Property(MaxTest = 10)]
    public bool NonExistentTokenFails(PositiveInt seed)
    {
        var mockTokenRepo = new Mock<ITokenRepository>();
        var mockUserRepo = new Mock<IUserRepository>();
        var mockTokenGen = new Mock<ITokenGenerator>();
        var mockHasher = new Mock<IPasswordHasher>();
        var mockEmail = new Mock<IEmailPublisher>();
        var mockRateLimiter = new Mock<IRateLimiter>();

        mockRateLimiter.Setup(x => x.CheckAsync(It.IsAny<string>(), It.IsAny<int>(), It.IsAny<TimeSpan>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RateLimitResult(true, 0, 5, null));
        mockTokenGen.Setup(x => x.HashToken(It.IsAny<string>())).Returns("hashed-token");
        mockTokenRepo.Setup(x => x.GetByHashAsync("hashed-token", It.IsAny<CancellationToken>()))
            .ReturnsAsync((RecoveryToken?)null);

        var options = new RecoveryServiceOptions();
        var service = new RecoveryService(mockTokenRepo.Object, mockUserRepo.Object, mockTokenGen.Object,
            mockHasher.Object, mockEmail.Object, mockRateLimiter.Object, options);

        var result = service.ValidateTokenAsync("test-token", Guid.NewGuid()).Result;
        return !result.IsSuccess;
    }
}
