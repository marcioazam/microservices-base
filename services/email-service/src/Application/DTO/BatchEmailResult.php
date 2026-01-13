<?php

declare(strict_types=1);

namespace EmailService\Application\DTO;

/**
 * Result DTO for batch email processing.
 */
final readonly class BatchEmailResult
{
    /**
     * @param EmailResult[] $results
     */
    public function __construct(
        public string $correlationId,
        public array $results,
        public int $totalCount,
        public int $successCount,
        public int $failureCount,
        public ?string $errorMessage = null,
    ) {
    }

    public static function error(string $message, string $correlationId): self
    {
        return new self(
            correlationId: $correlationId,
            results: [],
            totalCount: 0,
            successCount: 0,
            failureCount: 0,
            errorMessage: $message,
        );
    }

    public function isSuccess(): bool
    {
        return $this->errorMessage === null && $this->failureCount === 0;
    }

    public function isPartialSuccess(): bool
    {
        return $this->errorMessage === null && $this->successCount > 0 && $this->failureCount > 0;
    }
}
