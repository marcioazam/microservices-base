<?php

declare(strict_types=1);

namespace EmailService\Application\DTO;

final readonly class RateLimitResult
{
    public function __construct(
        public bool $isAllowed,
        public int $remaining,
        public int $limit,
        public int $resetAt,
        public ?int $retryAfter = null
    ) {
    }

    public static function allowed(int $remaining, int $limit, int $resetAt): self
    {
        return new self(
            isAllowed: true,
            remaining: $remaining,
            limit: $limit,
            resetAt: $resetAt
        );
    }

    public static function exceeded(int $limit, int $resetAt, int $retryAfter): self
    {
        return new self(
            isAllowed: false,
            remaining: 0,
            limit: $limit,
            resetAt: $resetAt,
            retryAfter: $retryAfter
        );
    }

    public function isApproachingLimit(float $threshold = 0.8): bool
    {
        return $this->isAllowed && ($this->remaining / $this->limit) <= (1 - $threshold);
    }
}
