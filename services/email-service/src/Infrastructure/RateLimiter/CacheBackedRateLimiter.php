<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\RateLimiter;

use EmailService\Application\DTO\RateLimitResult;
use EmailService\Infrastructure\Platform\CacheClientInterface;
use Psr\Log\LoggerInterface;
use Psr\Log\NullLogger;

/**
 * Rate limiter backed by platform cache-service.
 * Uses distributed cache for rate limiting state management.
 */
final class CacheBackedRateLimiter implements RateLimiterInterface
{
    private const KEY_PREFIX = 'rate_limit';
    private const NAMESPACE = 'email';

    public function __construct(
        private readonly CacheClientInterface $cacheClient,
        private readonly int $limit = 100,
        private readonly int $windowSeconds = 60,
        private readonly ?InMemoryRateLimiter $fallback = null,
        private readonly LoggerInterface $logger = new NullLogger(),
    ) {}

    public function check(string $identifier): RateLimitResult
    {
        try {
            $key = $this->buildKey($identifier);
            $cached = $this->cacheClient->get($key, self::NAMESPACE);

            if ($cached === null) {
                return RateLimitResult::allowed(
                    remaining: $this->limit,
                    limit: $this->limit,
                    resetAt: $this->getWindowEnd()
                );
            }

            $data = $cached->value;
            $count = $data['count'] ?? 0;
            $windowEnd = $data['window_end'] ?? $this->getWindowEnd();

            // Check if window has expired
            if ($windowEnd < time()) {
                return RateLimitResult::allowed(
                    remaining: $this->limit,
                    limit: $this->limit,
                    resetAt: $this->getWindowEnd()
                );
            }

            $remaining = max(0, $this->limit - $count);

            if ($remaining === 0) {
                $retryAfter = $windowEnd - time();
                return RateLimitResult::exceeded(
                    limit: $this->limit,
                    resetAt: $windowEnd,
                    retryAfter: $retryAfter
                );
            }

            return RateLimitResult::allowed(
                remaining: $remaining,
                limit: $this->limit,
                resetAt: $windowEnd
            );
        } catch (\Throwable $e) {
            $this->logger->warning('Rate limit check failed, using fallback', [
                'identifier' => $identifier,
                'error' => $e->getMessage(),
            ]);

            return $this->fallback?->check($identifier)
                ?? RateLimitResult::allowed(
                    remaining: $this->limit,
                    limit: $this->limit,
                    resetAt: $this->getWindowEnd()
                );
        }
    }

    public function hit(string $identifier): RateLimitResult
    {
        try {
            $key = $this->buildKey($identifier);
            $cached = $this->cacheClient->get($key, self::NAMESPACE);

            $now = time();
            $windowEnd = $this->getWindowEnd();

            if ($cached === null) {
                // First hit in window
                $data = [
                    'count' => 1,
                    'window_end' => $windowEnd,
                ];
                $this->cacheClient->set($key, $data, $this->windowSeconds, self::NAMESPACE);

                return RateLimitResult::allowed(
                    remaining: $this->limit - 1,
                    limit: $this->limit,
                    resetAt: $windowEnd
                );
            }

            $data = $cached->value;
            $count = $data['count'] ?? 0;
            $storedWindowEnd = $data['window_end'] ?? $windowEnd;

            // Check if window has expired
            if ($storedWindowEnd < $now) {
                // Start new window
                $data = [
                    'count' => 1,
                    'window_end' => $windowEnd,
                ];
                $this->cacheClient->set($key, $data, $this->windowSeconds, self::NAMESPACE);

                return RateLimitResult::allowed(
                    remaining: $this->limit - 1,
                    limit: $this->limit,
                    resetAt: $windowEnd
                );
            }

            // Increment count
            $newCount = $count + 1;
            $data['count'] = $newCount;

            $ttl = $storedWindowEnd - $now;
            $this->cacheClient->set($key, $data, max(1, $ttl), self::NAMESPACE);

            $remaining = max(0, $this->limit - $newCount);

            if ($remaining === 0) {
                $retryAfter = $storedWindowEnd - $now;
                return RateLimitResult::exceeded(
                    limit: $this->limit,
                    resetAt: $storedWindowEnd,
                    retryAfter: $retryAfter
                );
            }

            return RateLimitResult::allowed(
                remaining: $remaining,
                limit: $this->limit,
                resetAt: $storedWindowEnd
            );
        } catch (\Throwable $e) {
            $this->logger->warning('Rate limit hit failed, using fallback', [
                'identifier' => $identifier,
                'error' => $e->getMessage(),
            ]);

            return $this->fallback?->hit($identifier)
                ?? RateLimitResult::allowed(
                    remaining: $this->limit - 1,
                    limit: $this->limit,
                    resetAt: $this->getWindowEnd()
                );
        }
    }

    public function reset(string $identifier): void
    {
        try {
            $key = $this->buildKey($identifier);
            $this->cacheClient->delete($key, self::NAMESPACE);
            $this->fallback?->reset($identifier);
        } catch (\Throwable $e) {
            $this->logger->warning('Rate limit reset failed', [
                'identifier' => $identifier,
                'error' => $e->getMessage(),
            ]);

            $this->fallback?->reset($identifier);
        }
    }

    public function getCount(string $identifier): int
    {
        try {
            $key = $this->buildKey($identifier);
            $cached = $this->cacheClient->get($key, self::NAMESPACE);

            if ($cached === null) {
                return 0;
            }

            $data = $cached->value;
            $windowEnd = $data['window_end'] ?? 0;

            // Check if window has expired
            if ($windowEnd < time()) {
                return 0;
            }

            return $data['count'] ?? 0;
        } catch (\Throwable $e) {
            $this->logger->warning('Rate limit get count failed, using fallback', [
                'identifier' => $identifier,
                'error' => $e->getMessage(),
            ]);

            return $this->fallback?->getCount($identifier) ?? 0;
        }
    }

    private function buildKey(string $identifier): string
    {
        return self::KEY_PREFIX . ':' . $identifier;
    }

    private function getWindowEnd(): int
    {
        return time() + $this->windowSeconds;
    }
}
