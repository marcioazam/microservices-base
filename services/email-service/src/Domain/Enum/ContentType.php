<?php

declare(strict_types=1);

namespace EmailService\Domain\Enum;

enum ContentType: string
{
    case HTML = 'text/html';
    case PLAIN = 'text/plain';

    public function isHtml(): bool
    {
        return $this === self::HTML;
    }
}
