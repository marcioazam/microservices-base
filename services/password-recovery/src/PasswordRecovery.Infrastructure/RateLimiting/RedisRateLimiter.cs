using PasswordRecovery.Application.Interfaces;
using StackExchange.Redis;

namespace PasswordRecovery.Infrastructure.RateLimiting;

public class RedisRateLimiter : IRateLimiter
{
    private readonly IConnectionMultiplexer _redis;
    private const string KeyPrefix = "ratelimit:";

    public RedisRateLimiter(IConnectionMultiplexer redis)
    {
        _redis = redis;
    }

    public async Task<RateLimitResult> CheckAsync(string key, int limit, TimeSpan window, CancellationToken ct = default)
    {
        var db = _redis.GetDatabase();
        var fullKey = $"{KeyPrefix}{key}";
        
        var count = await db.StringGetAsync(fullKey);
        var currentCount = count.HasValue ? (int)count : 0;

        if (currentCount >= limit)
        {
            var ttl = await db.KeyTimeToLiveAsync(fullKey);
            return new RateLimitResult(false, currentCount, limit, ttl);
        }

        return new RateLimitResult(true, currentCount, limit, null);
    }

    public async Task IncrementAsync(string key, TimeSpan window, CancellationToken ct = default)
    {
        var db = _redis.GetDatabase();
        var fullKey = $"{KeyPrefix}{key}";

        var transaction = db.CreateTransaction();
        _ = transaction.StringIncrementAsync(fullKey);
        _ = transaction.KeyExpireAsync(fullKey, window, ExpireWhen.HasNoExpiry);
        await transaction.ExecuteAsync();
    }
}
