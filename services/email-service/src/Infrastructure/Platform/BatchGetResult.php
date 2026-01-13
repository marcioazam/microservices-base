<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

/**
 * Result of a batch get operation from cache.
 */
final readonly class BatchGetResult
{
    /**
     * @param array<string, mixed> $values Map of key to value for found keys
     * @param array<string> $missingKeys List of keys that were not found
     */
    public function __construct(
        public array $values,
        public array $missingKeys,
    ) {}

    public static function empty(): self
    {
        return new self([], []);
    }

    public static function allMissing(array $keys): self
    {
        return new self([], $keys);
    }

    public function hasKey(string $key): bool
    {
        return array_key_exists($key, $this->values);
    }

    public function get(string $key): mixed
    {
        return $this->values[$key] ?? null;
    }
}
