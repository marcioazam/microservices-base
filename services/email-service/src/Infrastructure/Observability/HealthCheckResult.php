<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Observability;

/**
 * Health check result with dependency statuses.
 */
final readonly class HealthCheckResult
{
    /**
     * @param array<string, DependencyStatus> $dependencies
     */
    public function __construct(
        public string $status,
        public \DateTimeImmutable $timestamp,
        public array $dependencies,
        public string $version,
    ) {
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'status' => $this->status,
            'timestamp' => $this->timestamp->format(\DateTimeInterface::RFC3339),
            'version' => $this->version,
            'dependencies' => array_map(
                fn(DependencyStatus $dep) => $dep->toArray(),
                $this->dependencies
            ),
        ];
    }
}
