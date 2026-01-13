<?php

declare(strict_types=1);

namespace EmailService\Api\Request;

use DateTimeImmutable;
use EmailService\Application\DTO\AuditQuery;
use EmailService\Domain\Enum\EmailStatus;

final readonly class AuditQueryRequest
{
    public function __construct(
        public ?string $startDate = null,
        public ?string $endDate = null,
        public ?string $status = null,
        public ?string $senderId = null,
        public ?string $emailId = null,
        public int $limit = 100,
        public int $offset = 0
    ) {
    }

    public static function fromArray(array $data): self
    {
        return new self(
            startDate: $data['start_date'] ?? null,
            endDate: $data['end_date'] ?? null,
            status: $data['status'] ?? null,
            senderId: $data['sender_id'] ?? null,
            emailId: $data['email_id'] ?? null,
            limit: (int) ($data['limit'] ?? 100),
            offset: (int) ($data['offset'] ?? 0)
        );
    }

    public function toAuditQuery(): AuditQuery
    {
        return new AuditQuery(
            startDate: $this->startDate ? new DateTimeImmutable($this->startDate) : null,
            endDate: $this->endDate ? new DateTimeImmutable($this->endDate) : null,
            status: $this->status ? EmailStatus::tryFrom($this->status) : null,
            senderId: $this->senderId,
            emailId: $this->emailId,
            limit: min($this->limit, 1000),
            offset: max($this->offset, 0)
        );
    }

    /**
     * @return array<string, string>
     */
    public function validate(): array
    {
        $errors = [];

        if ($this->startDate !== null) {
            try {
                new DateTimeImmutable($this->startDate);
            } catch (\Exception) {
                $errors['start_date'] = 'Invalid start date format';
            }
        }

        if ($this->endDate !== null) {
            try {
                new DateTimeImmutable($this->endDate);
            } catch (\Exception) {
                $errors['end_date'] = 'Invalid end date format';
            }
        }

        if ($this->status !== null && EmailStatus::tryFrom($this->status) === null) {
            $errors['status'] = 'Invalid status value';
        }

        if ($this->limit < 1 || $this->limit > 1000) {
            $errors['limit'] = 'Limit must be between 1 and 1000';
        }

        if ($this->offset < 0) {
            $errors['offset'] = 'Offset must be non-negative';
        }

        return $errors;
    }
}
