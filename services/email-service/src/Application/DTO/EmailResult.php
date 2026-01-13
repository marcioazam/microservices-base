<?php

declare(strict_types=1);

namespace EmailService\Application\DTO;

/**
 * Result DTO for individual email processing.
 */
final readonly class EmailResult
{
    public function __construct(
        public string $emailId,
        public bool $isSuccess,
        public ?string $messageId = null,
        public ?string $errorCode = null,
        public ?string $errorMessage = null,
        public ?int $index = null,
    ) {
    }

    public static function success(string $emailId, ?string $messageId = null, ?int $index = null): self
    {
        return new self(
            emailId: $emailId,
            isSuccess: true,
            messageId: $messageId,
            index: $index,
        );
    }

    public static function failure(
        string $emailId,
        string $errorCode,
        string $errorMessage,
        ?int $index = null,
    ): self {
        return new self(
            emailId: $emailId,
            isSuccess: false,
            errorCode: $errorCode,
            errorMessage: $errorMessage,
            index: $index,
        );
    }
}
