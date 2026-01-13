<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

/**
 * Represents a cached value with its source information.
 */
final readonly class CacheValue
{
    public function __construct(
        public mixed $value,
        public CacheSource $source,
    ) {}

    public static function fromRedis(mixed $value): self
    {
        return new self($value, CacheSource::REDIS);
    }

    public static function fromLocal(mixed $value): self
    {
        return new self($value, CacheSource::LOCAL);
    }

    public static function fromMemory(mixed $value): self
    {
        return new self($value, CacheSource::MEMORY);
    }
}
