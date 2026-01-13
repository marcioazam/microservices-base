<?php

declare(strict_types=1);

namespace EmailService\Application\DTO;

final readonly class ValidationResult
{
    public function __construct(
        public bool $isValid,
        public ?string $errorCode = null,
        public ?string $errorMessage = null
    ) {
    }

    public static function valid(): self
    {
        return new self(isValid: true);
    }

    public static function invalid(string $errorCode, string $errorMessage): self
    {
        return new self(
            isValid: false,
            errorCode: $errorCode,
            errorMessage: $errorMessage
        );
    }

    public static function invalidFormat(): self
    {
        return self::invalid('INVALID_EMAIL_FORMAT', 'Email address format is invalid');
    }

    public static function invalidDomain(): self
    {
        return self::invalid('INVALID_DOMAIN', 'Email domain has no valid MX records');
    }

    public static function disposableEmail(): self
    {
        return self::invalid('DISPOSABLE_EMAIL', 'Disposable email addresses are not allowed');
    }
}
