<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

/**
 * gRPC client for platform logging-service.
 * Integrates with platform/logging-service via gRPC protocol.
 */
final readonly class LoggingClient implements LoggingClientInterface
{
    private const DEFAULT_TIMEOUT_MS = 5000;

    public function __construct(
        private string $host,
        private int $port,
        private int $timeoutMs = self::DEFAULT_TIMEOUT_MS,
        private ?FallbackLogger $fallback = null,
    ) {}

    public function log(LogEntry $entry): void
    {
        try {
            $this->callGrpc('IngestLog', [
                'entry' => $this->entryToProto($entry),
            ]);
        } catch (\Throwable $e) {
            $this->fallback?->log($entry);
        }
    }

    public function logBatch(array $entries): BatchLogResult
    {
        if (empty($entries)) {
            return BatchLogResult::success(0);
        }

        try {
            $protoEntries = array_map(
                fn(LogEntry $entry) => $this->entryToProto($entry),
                $entries
            );

            $response = $this->callGrpc('IngestLogBatch', [
                'entries' => $protoEntries,
            ]);

            return new BatchLogResult(
                acceptedCount: $response['accepted_count'] ?? 0,
                rejectedCount: $response['rejected_count'] ?? 0,
                errors: $response['errors'] ?? [],
            );
        } catch (\Throwable $e) {
            // Fallback: log each entry individually
            foreach ($entries as $entry) {
                $this->fallback?->log($entry);
            }

            return BatchLogResult::failure(count($entries), [$e->getMessage()]);
        }
    }

    public function isHealthy(): bool
    {
        try {
            $this->callGrpc('Health', []);
            return true;
        } catch (\Throwable) {
            return false;
        }
    }

    private function entryToProto(LogEntry $entry): array
    {
        return $entry->toArray();
    }

    /**
     * @param array<string, mixed> $params
     * @return array<string, mixed>
     */
    private function callGrpc(string $method, array $params): array
    {
        $endpoint = sprintf('%s:%d', $this->host, $this->port);

        // This is a placeholder for actual gRPC implementation
        // Real implementation would use: new \Logging\V1\LoggingServiceClient($endpoint, [...])
        throw new \RuntimeException("gRPC call to {$method} at {$endpoint} - implement with grpc extension");
    }
}
