<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

/**
 * Interface for cache operations via platform cache-service gRPC.
 * Integrates with platform/cache-service for distributed caching.
 */
interface CacheClientInterface
{
    /**
     * Get a value from cache.
     */
    public function get(string $key, string $namespace = 'email'): ?CacheValue;

    /**
     * Set a value in cache.
     */
    public function set(string $key, mixed $value, int $ttlSeconds = 0, string $namespace = 'email'): bool;

    /**
     * Delete a value from cache.
     */
    public function delete(string $key, string $namespace = 'email'): bool;

    /**
     * Get multiple values from cache.
     *
     * @param array<string> $keys
     */
    public function batchGet(array $keys, string $namespace = 'email'): BatchGetResult;

    /**
     * Set multiple values in cache.
     *
     * @param array<string, mixed> $entries
     */
    public function batchSet(array $entries, int $ttlSeconds = 0, string $namespace = 'email'): bool;

    /**
     * Check if the cache service is healthy.
     */
    public function isHealthy(): bool;
}
