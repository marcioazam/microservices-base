<?php

declare(strict_types=1);

namespace EmailService\Domain\Enum;

enum EmailStatus: string
{
    case PENDING = 'pending';
    case QUEUED = 'queued';
    case SENDING = 'sending';
    case SENT = 'sent';
    case DELIVERED = 'delivered';
    case FAILED = 'failed';
    case BOUNCED = 'bounced';
    case REJECTED = 'rejected';

    public function isFinal(): bool
    {
        return in_array($this, [self::DELIVERED, self::FAILED, self::BOUNCED, self::REJECTED], true);
    }

    public function canRetry(): bool
    {
        return $this === self::FAILED;
    }
}
