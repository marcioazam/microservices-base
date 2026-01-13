<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

use Psr\Log\LoggerInterface;

/**
 * gRPC client for platform cache-service.
 * Integrates with platform/cache-service via gRPC protocol.
 */
final readonly class CacheClient implements CacheClientInterface
{
    private const DEFAULT_TIMEOUT_MS = 5000;

    public function __construct(
        private string $host,
        private int $port,
        private LoggerInterface $logger,
        private int $timeoutMs = self::DEFAULT_TIMEOUT_MS,
        private ?InMemoryCacheClient $fallback = null,
    ) {}

    public function get(string $key, string $namespace = 'email'): ?CacheValue
    {
        try {
            $response = $this->callGrpc('Get', [
                'key' => $key,
                'namespace' => $namespace,
            ]);

            if (!$response['found']) {
                return null;
            }

            $value = $this->deserialize($response['value']);
            $source = match ($response['source'] ?? 1) {
                2 => CacheSource::LOCAL,
                default => CacheSource::REDIS,
            };

            return new CacheValue($value, $source);
        } catch (\Throwable $e) {
            $this->logger->warning('Cache get failed, using fallback', [
                'key' => $key,
                'error' => $e->getMessage(),
            ]);

            return $this->fallback?->get($key, $namespace);
        }
    }

    public function set(string $key, mixed $value, int $ttlSeconds = 0, string $namespace = 'email'): bool
    {
        try {
            $response = $this->callGrpc('Set', [
                'key' => $key,
                'value' => $this->serialize($value),
                'ttl_seconds' => $ttlSeconds,
                'namespace' => $namespace,
                'encrypt' => false,
            ]);

            $success = $response['success'] ?? false;

            if ($success) {
                $this->fallback?->set($key, $value, $ttlSeconds, $namespace);
            }

            return $success;
        } catch (\Throwable $e) {
            $this->logger->warning('Cache set failed, using fallback', [
                'key' => $key,
                'error' => $e->getMessage(),
            ]);

            return $this->fallback?->set($key, $value, $ttlSeconds, $namespace) ?? false;
        }
    }

    public function delete(string $key, string $namespace = 'email'): bool
    {
        try {
            $response = $this->callGrpc('Delete', [
                'key' => $key,
                'namespace' => $namespace,
            ]);

            $deleted = $response['deleted'] ?? false;

            if ($deleted) {
                $this->fallback?->delete($key, $namespace);
            }

            return $deleted;
        } catch (\Throwable $e) {
            $this->logger->warning('Cache delete failed, using fallback', [
                'key' => $key,
                'error' => $e->getMessage(),
            ]);

            return $this->fallback?->delete($key, $namespace) ?? false;
        }
    }

    public function batchGet(array $keys, string $namespace = 'email'): BatchGetResult
    {
        if (empty($keys)) {
            return BatchGetResult::empty();
        }

        try {
            $response = $this->callGrpc('BatchGet', [
                'keys' => $keys,
                'namespace' => $namespace,
            ]);

            $values = [];
            foreach ($response['values'] ?? [] as $key => $serialized) {
                $values[$key] = $this->deserialize($serialized);
            }

            return new BatchGetResult($values, $response['missing_keys'] ?? []);
        } catch (\Throwable $e) {
            $this->logger->warning('Cache batch get failed, using fallback', [
                'keys' => $keys,
                'error' => $e->getMessage(),
            ]);

            return $this->fallback?->batchGet($keys, $namespace) ?? BatchGetResult::allMissing($keys);
        }
    }

    public function batchSet(array $entries, int $ttlSeconds = 0, string $namespace = 'email'): bool
    {
        if (empty($entries)) {
            return true;
        }

        try {
            $serializedEntries = [];
            foreach ($entries as $key => $value) {
                $serializedEntries[$key] = $this->serialize($value);
            }

            $response = $this->callGrpc('BatchSet', [
                'entries' => $serializedEntries,
                'ttl_seconds' => $ttlSeconds,
                'namespace' => $namespace,
            ]);

            $success = $response['success'] ?? false;

            if ($success) {
                $this->fallback?->batchSet($entries, $ttlSeconds, $namespace);
            }

            return $success;
        } catch (\Throwable $e) {
            $this->logger->warning('Cache batch set failed, using fallback', [
                'count' => count($entries),
                'error' => $e->getMessage(),
            ]);

            return $this->fallback?->batchSet($entries, $ttlSeconds, $namespace) ?? false;
        }
    }

    public function isHealthy(): bool
    {
        try {
            $response = $this->callGrpc('Health', []);
            return $response['healthy'] ?? false;
        } catch (\Throwable) {
            return false;
        }
    }

    /**
     * @param array<string, mixed> $params
     * @return array<string, mixed>
     */
    private function callGrpc(string $method, array $params): array
    {
        // In production, this would use the gRPC PHP extension
        // For now, we simulate the call structure
        $endpoint = sprintf('%s:%d', $this->host, $this->port);

        // This is a placeholder for actual gRPC implementation
        // Real implementation would use: new \Cache\V1\CacheServiceClient($endpoint, [...])
        throw new \RuntimeException("gRPC call to {$method} at {$endpoint} - implement with grpc extension");
    }

    private function serialize(mixed $value): string
    {
        return json_encode($value, JSON_THROW_ON_ERROR);
    }

    private function deserialize(string $data): mixed
    {
        return json_decode($data, true, 512, JSON_THROW_ON_ERROR);
    }
}
