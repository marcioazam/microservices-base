<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

enum LogLevel: int
{
    case DEBUG = 1;
    case INFO = 2;
    case WARN = 3;
    case ERROR = 4;
    case FATAL = 5;

    public function toString(): string
    {
        return match ($this) {
            self::DEBUG => 'DEBUG',
            self::INFO => 'INFO',
            self::WARN => 'WARN',
            self::ERROR => 'ERROR',
            self::FATAL => 'FATAL',
        };
    }
}
