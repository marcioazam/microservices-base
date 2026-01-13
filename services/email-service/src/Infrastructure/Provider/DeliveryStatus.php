<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Provider;

use EmailService\Domain\Enum\EmailStatus;

final readonly class DeliveryStatus
{
    public function __construct(
        public EmailStatus $status,
        public ?string $details = null,
        public ?\DateTimeImmutable $timestamp = null
    ) {
    }

    public static function delivered(): self
    {
        return new self(EmailStatus::DELIVERED);
    }

    public static function bounced(string $reason): self
    {
        return new self(EmailStatus::BOUNCED, $reason);
    }

    public static function pending(): self
    {
        return new self(EmailStatus::SENT);
    }

    public static function failed(string $reason): self
    {
        return new self(EmailStatus::FAILED, $reason);
    }
}
