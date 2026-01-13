<?php

declare(strict_types=1);

namespace EmailService\Domain\Enum;

enum EmailType: string
{
    case TRANSACTIONAL = 'transactional';
    case MARKETING = 'marketing';
    case VERIFICATION = 'verification';
}
