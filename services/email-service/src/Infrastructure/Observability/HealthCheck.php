<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Observability;

use EmailService\Infrastructure\Platform\CacheClientInterface;
use EmailService\Infrastructure\Platform\LoggingClientInterface;

/**
 * Health check service with dependency status reporting.
 * Includes CacheClient, LoggingClient, and provider connectivity status.
 */
final readonly class HealthCheck implements HealthCheckInterface
{
    private const STATUS_HEALTHY = 'healthy';
    private const STATUS_DEGRADED = 'degraded';
    private const STATUS_UNHEALTHY = 'unhealthy';

    public function __construct(
        private CacheClientInterface $cacheClient,
        private LoggingClientInterface $loggingClient,
        private ?ProviderHealthCheckerInterface $providerChecker = null,
    ) {
    }

    public function check(): HealthCheckResult
    {
        $dependencies = $this->checkDependencies();
        $overallStatus = $this->determineOverallStatus($dependencies);

        return new HealthCheckResult(
            status: $overallStatus,
            timestamp: new \DateTimeImmutable(),
            dependencies: $dependencies,
            version: $this->getServiceVersion(),
        );
    }

    public function checkLiveness(): bool
    {
        return true; // Service is running
    }

    public function checkReadiness(): bool
    {
        $result = $this->check();
        return $result->status !== self::STATUS_UNHEALTHY;
    }

    /**
     * @return array<string, DependencyStatus>
     */
    private function checkDependencies(): array
    {
        return [
            'cache' => $this->checkCacheHealth(),
            'logging' => $this->checkLoggingHealth(),
            'providers' => $this->checkProvidersHealth(),
        ];
    }

    private function checkCacheHealth(): DependencyStatus
    {
        $startTime = hrtime(true);

        try {
            $healthy = $this->cacheClient->isHealthy();
            $latencyMs = (hrtime(true) - $startTime) / 1_000_000;

            return new DependencyStatus(
                name: 'cache-service',
                status: $healthy ? self::STATUS_HEALTHY : self::STATUS_DEGRADED,
                latencyMs: $latencyMs,
                message: $healthy ? 'Connected to cache-service via gRPC' : 'Cache unavailable, using fallback',
            );
        } catch (\Throwable $e) {
            $latencyMs = (hrtime(true) - $startTime) / 1_000_000;

            return new DependencyStatus(
                name: 'cache-service',
                status: self::STATUS_DEGRADED,
                latencyMs: $latencyMs,
                message: 'Cache check failed: ' . $e->getMessage(),
            );
        }
    }

    private function checkLoggingHealth(): DependencyStatus
    {
        $startTime = hrtime(true);

        try {
            $healthy = $this->loggingClient->isHealthy();
            $latencyMs = (hrtime(true) - $startTime) / 1_000_000;

            return new DependencyStatus(
                name: 'logging-service',
                status: $healthy ? self::STATUS_HEALTHY : self::STATUS_DEGRADED,
                latencyMs: $latencyMs,
                message: $healthy ? 'Connected to logging-service via gRPC' : 'Logging unavailable, using fallback',
            );
        } catch (\Throwable $e) {
            $latencyMs = (hrtime(true) - $startTime) / 1_000_000;

            return new DependencyStatus(
                name: 'logging-service',
                status: self::STATUS_DEGRADED,
                latencyMs: $latencyMs,
                message: 'Logging check failed: ' . $e->getMessage(),
            );
        }
    }

    private function checkProvidersHealth(): DependencyStatus
    {
        if ($this->providerChecker === null) {
            return new DependencyStatus(
                name: 'email-providers',
                status: self::STATUS_HEALTHY,
                latencyMs: 0,
                message: 'Provider health check not configured',
            );
        }

        $startTime = hrtime(true);

        try {
            $providerStatuses = $this->providerChecker->checkAll();
            $latencyMs = (hrtime(true) - $startTime) / 1_000_000;

            $healthyCount = count(array_filter($providerStatuses, fn($s) => $s));
            $totalCount = count($providerStatuses);

            $status = match (true) {
                $healthyCount === $totalCount => self::STATUS_HEALTHY,
                $healthyCount > 0 => self::STATUS_DEGRADED,
                default => self::STATUS_UNHEALTHY,
            };

            return new DependencyStatus(
                name: 'email-providers',
                status: $status,
                latencyMs: $latencyMs,
                message: "{$healthyCount}/{$totalCount} providers available",
            );
        } catch (\Throwable $e) {
            $latencyMs = (hrtime(true) - $startTime) / 1_000_000;

            return new DependencyStatus(
                name: 'email-providers',
                status: self::STATUS_UNHEALTHY,
                latencyMs: $latencyMs,
                message: 'Provider check failed: ' . $e->getMessage(),
            );
        }
    }

    /**
     * @param array<string, DependencyStatus> $dependencies
     */
    private function determineOverallStatus(array $dependencies): string
    {
        $hasUnhealthy = false;
        $hasDegraded = false;

        foreach ($dependencies as $dep) {
            if ($dep->status === self::STATUS_UNHEALTHY) {
                $hasUnhealthy = true;
            }
            if ($dep->status === self::STATUS_DEGRADED) {
                $hasDegraded = true;
            }
        }

        return match (true) {
            $hasUnhealthy => self::STATUS_UNHEALTHY,
            $hasDegraded => self::STATUS_DEGRADED,
            default => self::STATUS_HEALTHY,
        };
    }

    private function getServiceVersion(): string
    {
        return $_ENV['SERVICE_VERSION'] ?? '1.0.0';
    }
}
