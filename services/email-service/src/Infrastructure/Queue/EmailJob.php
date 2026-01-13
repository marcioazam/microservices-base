<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Queue;

use DateTimeImmutable;
use EmailService\Domain\Entity\Email;
use Symfony\Component\Uid\Uuid;

class EmailJob
{
    public function __construct(
        public readonly string $id,
        public readonly Email $email,
        public readonly int $priority = 0,
        public int $attempts = 0,
        public readonly int $maxAttempts = 3,
        public ?DateTimeImmutable $nextRetryAt = null,
        public readonly DateTimeImmutable $createdAt = new DateTimeImmutable()
    ) {
    }

    public static function create(Email $email, int $priority = 0, int $maxAttempts = 3): self
    {
        return new self(
            id: Uuid::v4()->toRfc4122(),
            email: $email,
            priority: $priority,
            maxAttempts: $maxAttempts
        );
    }

    public function incrementAttempts(): void
    {
        $this->attempts++;
    }

    public function canRetry(): bool
    {
        return $this->attempts < $this->maxAttempts;
    }

    public function scheduleRetry(): void
    {
        // Exponential backoff: 1s, 2s, 4s, 8s, 16s
        $delaySeconds = (int) pow(2, $this->attempts - 1);
        $this->nextRetryAt = new DateTimeImmutable("+{$delaySeconds} seconds");
    }

    public function getBackoffDelay(): int
    {
        return (int) pow(2, $this->attempts - 1);
    }

    public function isReadyForRetry(): bool
    {
        if ($this->nextRetryAt === null) {
            return true;
        }
        
        return $this->nextRetryAt <= new DateTimeImmutable();
    }

    public function toArray(): array
    {
        return [
            'id' => $this->id,
            'email_id' => $this->email->id,
            'priority' => $this->priority,
            'attempts' => $this->attempts,
            'max_attempts' => $this->maxAttempts,
            'next_retry_at' => $this->nextRetryAt?->format('Y-m-d H:i:s'),
            'created_at' => $this->createdAt->format('Y-m-d H:i:s'),
        ];
    }
}
