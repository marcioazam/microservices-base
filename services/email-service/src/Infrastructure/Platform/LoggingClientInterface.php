<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

/**
 * Interface for logging operations via platform logging-service gRPC.
 * Integrates with platform/logging-service for centralized structured logging.
 */
interface LoggingClientInterface
{
    /**
     * Send a single log entry to the logging service.
     */
    public function log(LogEntry $entry): void;

    /**
     * Send multiple log entries in a batch.
     *
     * @param LogEntry[] $entries
     */
    public function logBatch(array $entries): BatchLogResult;

    /**
     * Check if the logging service is healthy.
     */
    public function isHealthy(): bool;
}
