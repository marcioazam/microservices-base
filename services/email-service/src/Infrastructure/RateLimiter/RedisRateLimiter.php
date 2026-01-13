<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\RateLimiter;

use EmailService\Application\DTO\RateLimitResult;
use Predis\ClientInterface;

class RedisRateLimiter implements RateLimiterInterface
{
    private const DEFAULT_LIMIT = 100;
    private const DEFAULT_WINDOW_SECONDS = 60;
    private const KEY_PREFIX = 'email_rate_limit:';

    public function __construct(
        private readonly ClientInterface $redis,
        private readonly int $limit = self::DEFAULT_LIMIT,
        private readonly int $windowSeconds = self::DEFAULT_WINDOW_SECONDS
    ) {
    }

    public function check(string $identifier): RateLimitResult
    {
        $key = $this->getKey($identifier);
        $currentCount = (int) $this->redis->get($key);
        $ttl = (int) $this->redis->ttl($key);
        
        if ($ttl < 0) {
            $ttl = $this->windowSeconds;
        }
        
        $resetAt = time() + $ttl;
        $remaining = max(0, $this->limit - $currentCount);

        if ($currentCount >= $this->limit) {
            return RateLimitResult::exceeded(
                limit: $this->limit,
                resetAt: $resetAt,
                retryAfter: $ttl
            );
        }

        return RateLimitResult::allowed(
            remaining: $remaining,
            limit: $this->limit,
            resetAt: $resetAt
        );
    }

    public function hit(string $identifier): RateLimitResult
    {
        $key = $this->getKey($identifier);
        
        // Use MULTI/EXEC for atomic operation
        $this->redis->multi();
        $this->redis->incr($key);
        $this->redis->ttl($key);
        $results = $this->redis->exec();
        
        $currentCount = (int) $results[0];
        $ttl = (int) $results[1];
        
        // Set expiry if this is a new key
        if ($ttl < 0) {
            $this->redis->expire($key, $this->windowSeconds);
            $ttl = $this->windowSeconds;
        }
        
        $resetAt = time() + $ttl;
        $remaining = max(0, $this->limit - $currentCount);

        if ($currentCount > $this->limit) {
            return RateLimitResult::exceeded(
                limit: $this->limit,
                resetAt: $resetAt,
                retryAfter: $ttl
            );
        }

        return RateLimitResult::allowed(
            remaining: $remaining,
            limit: $this->limit,
            resetAt: $resetAt
        );
    }

    public function reset(string $identifier): void
    {
        $key = $this->getKey($identifier);
        $this->redis->del([$key]);
    }

    public function getCount(string $identifier): int
    {
        $key = $this->getKey($identifier);
        return (int) $this->redis->get($key);
    }

    private function getKey(string $identifier): string
    {
        return self::KEY_PREFIX . $identifier;
    }
}
