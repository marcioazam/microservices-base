<?php

declare(strict_types=1);

namespace EmailService\Application\DTO;

use DateTimeImmutable;
use EmailService\Domain\Enum\EmailStatus;

final readonly class AuditQuery
{
    public function __construct(
        public ?DateTimeImmutable $startDate = null,
        public ?DateTimeImmutable $endDate = null,
        public ?EmailStatus $status = null,
        public ?string $senderId = null,
        public ?string $emailId = null,
        public int $limit = 100,
        public int $offset = 0
    ) {
    }

    public function hasFilters(): bool
    {
        return $this->startDate !== null
            || $this->endDate !== null
            || $this->status !== null
            || $this->senderId !== null
            || $this->emailId !== null;
    }
}
