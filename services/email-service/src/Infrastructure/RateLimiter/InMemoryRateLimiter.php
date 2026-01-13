<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\RateLimiter;

use EmailService\Application\DTO\RateLimitResult;

/**
 * In-memory rate limiter for testing purposes
 */
class InMemoryRateLimiter implements RateLimiterInterface
{
    /** @var array<string, array{count: int, resetAt: int}> */
    private array $buckets = [];

    public function __construct(
        private readonly int $limit = 100,
        private readonly int $windowSeconds = 60
    ) {
    }

    public function check(string $identifier): RateLimitResult
    {
        $this->cleanExpired($identifier);
        
        $bucket = $this->buckets[$identifier] ?? null;
        $currentCount = $bucket['count'] ?? 0;
        $resetAt = $bucket['resetAt'] ?? time() + $this->windowSeconds;
        $remaining = max(0, $this->limit - $currentCount);

        if ($currentCount >= $this->limit) {
            return RateLimitResult::exceeded(
                limit: $this->limit,
                resetAt: $resetAt,
                retryAfter: max(0, $resetAt - time())
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
        $this->cleanExpired($identifier);
        
        if (!isset($this->buckets[$identifier])) {
            $this->buckets[$identifier] = [
                'count' => 0,
                'resetAt' => time() + $this->windowSeconds,
            ];
        }
        
        $this->buckets[$identifier]['count']++;
        
        return $this->check($identifier);
    }

    public function reset(string $identifier): void
    {
        unset($this->buckets[$identifier]);
    }

    public function getCount(string $identifier): int
    {
        $this->cleanExpired($identifier);
        return $this->buckets[$identifier]['count'] ?? 0;
    }

    private function cleanExpired(string $identifier): void
    {
        if (isset($this->buckets[$identifier]) && $this->buckets[$identifier]['resetAt'] <= time()) {
            unset($this->buckets[$identifier]);
        }
    }
}
