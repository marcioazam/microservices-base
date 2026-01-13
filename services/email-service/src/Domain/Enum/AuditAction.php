<?php

declare(strict_types=1);

namespace EmailService\Domain\Enum;

enum AuditAction: string
{
    case CREATED = 'created';
    case QUEUED = 'queued';
    case SENT = 'sent';
    case DELIVERED = 'delivered';
    case FAILED = 'failed';
    case RETRIED = 'retried';
    case DEAD_LETTERED = 'dead_lettered';
    case RESENT = 'resent';
}
