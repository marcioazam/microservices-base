<?php

declare(strict_types=1);

namespace EmailService\Domain\Exception;

use InvalidArgumentException;

class InvalidEmailException extends InvalidArgumentException
{
    public function __construct(
        public readonly string $email,
        public readonly string $errorCode = 'INVALID_EMAIL_FORMAT'
    ) {
        parent::__construct("Invalid email address: {$email}");
    }
}
