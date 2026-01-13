<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

/**
 * In-memory cache client fallback when platform cache-service is unavailable.
 * Provides degraded functionality with limited capacity and no persistence.
 */
final class InMemoryCacheClient implements CacheClientInterface
{
    private const MAX_ENTRIES = 1000;

    /** @var array<string, array{value: mixed, expires: ?int}> */
    private array $cache = [];

    public function __construct(
        private readonly int $maxEntries = self::MAX_ENTRIES,
    ) {}

    public function get(string $key, string $namespace = 'email'): ?CacheValue
    {
        $fullKey = $this->buildKey($key, $namespace);

        if (!isset($this->cache[$fullKey])) {
            return null;
        }

        $entry = $this->cache[$fullKey];

        if ($entry['expires'] !== null && $entry['expires'] < time()) {
            unset($this->cache[$fullKey]);
            return null;
        }

        return CacheValue::fromMemory($entry['value']);
    }

    public function set(string $key, mixed $value, int $ttlSeconds = 0, string $namespace = 'email'): bool
    {
        $this->evictIfNeeded();

        $fullKey = $this->buildKey($key, $namespace);
        $expires = $ttlSeconds > 0 ? time() + $ttlSeconds : null;

        $this->cache[$fullKey] = [
            'value' => $value,
            'expires' => $expires,
        ];

        return true;
    }

    public function delete(string $key, string $namespace = 'email'): bool
    {
        $fullKey = $this->buildKey($key, $namespace);

        if (!isset($this->cache[$fullKey])) {
            return false;
        }

        unset($this->cache[$fullKey]);
        return true;
    }

    public function batchGet(array $keys, string $namespace = 'email'): BatchGetResult
    {
        $values = [];
        $missingKeys = [];

        foreach ($keys as $key) {
            $result = $this->get($key, $namespace);
            if ($result !== null) {
                $values[$key] = $result->value;
            } else {
                $missingKeys[] = $key;
            }
        }

        return new BatchGetResult($values, $missingKeys);
    }

    public function batchSet(array $entries, int $ttlSeconds = 0, string $namespace = 'email'): bool
    {
        foreach ($entries as $key => $value) {
            $this->set($key, $value, $ttlSeconds, $namespace);
        }

        return true;
    }

    public function isHealthy(): bool
    {
        return true;
    }

    public function clear(): void
    {
        $this->cache = [];
    }

    public function count(): int
    {
        return count($this->cache);
    }

    private function buildKey(string $key, string $namespace): string
    {
        return "{$namespace}:{$key}";
    }

    private function evictIfNeeded(): void
    {
        if (count($this->cache) < $this->maxEntries) {
            return;
        }

        // Remove expired entries first
        $now = time();
        foreach ($this->cache as $key => $entry) {
            if ($entry['expires'] !== null && $entry['expires'] < $now) {
                unset($this->cache[$key]);
            }
        }

        // If still over limit, remove oldest entries (FIFO)
        while (count($this->cache) >= $this->maxEntries) {
            array_shift($this->cache);
        }
    }
}
