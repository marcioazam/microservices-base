<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\RateLimiter;

use EmailService\Application\DTO\RateLimitResult;

interface RateLimiterInterface
{
    /**
     * Check if request is allowed for given identifier
     */
    public function check(string $identifier): RateLimitResult;

    /**
     * Record a request for given identifier
     */
    public function hit(string $identifier): RateLimitResult;

    /**
     * Reset rate limit for given identifier
     */
    public function reset(string $identifier): void;

    /**
     * Get current count for identifier
     */
    public function getCount(string $identifier): int;
}
