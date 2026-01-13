namespace PasswordRecovery.Application.Interfaces;

public interface IRateLimiter
{
    Task<RateLimitResult> CheckAsync(string key, int limit, TimeSpan window, CancellationToken ct = default);
    Task IncrementAsync(string key, TimeSpan window, CancellationToken ct = default);
}

public record RateLimitResult(bool IsAllowed, int CurrentCount, int Limit, TimeSpan? RetryAfter);
