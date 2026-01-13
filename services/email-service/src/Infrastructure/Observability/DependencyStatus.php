<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Observability;

/**
 * Status of a single dependency.
 */
final readonly class DependencyStatus
{
    public function __construct(
        public string $name,
        public string $status,
        public float $latencyMs,
        public ?string $message = null,
    ) {
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'name' => $this->name,
            'status' => $this->status,
            'latency_ms' => round($this->latencyMs, 2),
            'message' => $this->message,
        ];
    }
}
