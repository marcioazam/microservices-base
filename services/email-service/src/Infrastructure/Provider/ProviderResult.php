<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Provider;

final readonly class ProviderResult
{
    public function __construct(
        public bool $success,
        public ?string $messageId = null,
        public ?string $errorCode = null,
        public ?string $errorMessage = null
    ) {
    }

    public static function success(string $messageId): self
    {
        return new self(success: true, messageId: $messageId);
    }

    public static function failure(string $errorCode, string $errorMessage): self
    {
        return new self(
            success: false,
            errorCode: $errorCode,
            errorMessage: $errorMessage
        );
    }
}
