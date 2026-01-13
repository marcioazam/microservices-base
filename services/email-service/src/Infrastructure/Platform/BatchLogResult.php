<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

/**
 * Result of a batch log operation.
 */
final readonly class BatchLogResult
{
    public function __construct(
        public int $acceptedCount,
        public int $rejectedCount,
        public array $errors = [],
    ) {}

    public static function success(int $count): self
    {
        return new self($count, 0, []);
    }

    public static function partial(int $accepted, int $rejected, array $errors): self
    {
        return new self($accepted, $rejected, $errors);
    }

    public static function failure(int $count, array $errors): self
    {
        return new self(0, $count, $errors);
    }

    public function isFullySuccessful(): bool
    {
        return $this->rejectedCount === 0;
    }
}
