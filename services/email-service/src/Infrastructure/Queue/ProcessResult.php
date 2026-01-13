<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Queue;

final readonly class ProcessResult
{
    public function __construct(
        public bool $success,
        public ?string $messageId = null,
        public ?string $errorMessage = null,
        public bool $shouldRetry = false
    ) {
    }

    public static function success(string $messageId): self
    {
        return new self(success: true, messageId: $messageId);
    }

    public static function failure(string $errorMessage, bool $shouldRetry = true): self
    {
        return new self(
            success: false,
            errorMessage: $errorMessage,
            shouldRetry: $shouldRetry
        );
    }

    public static function permanentFailure(string $errorMessage): self
    {
        return new self(
            success: false,
            errorMessage: $errorMessage,
            shouldRetry: false
        );
    }
}
