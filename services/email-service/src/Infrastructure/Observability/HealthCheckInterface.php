<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Observability;

/**
 * Interface for health check operations.
 */
interface HealthCheckInterface
{
    /**
     * Perform full health check with dependency status.
     */
    public function check(): HealthCheckResult;

    /**
     * Check if service is alive (Kubernetes liveness probe).
     */
    public function checkLiveness(): bool;

    /**
     * Check if service is ready to accept traffic (Kubernetes readiness probe).
     */
    public function checkReadiness(): bool;
}
