<?php

declare(strict_types=1);

namespace EmailService\Application\DTO;

use EmailService\Domain\Entity\Email;

/**
 * Request DTO for batch email processing.
 */
final readonly class BatchEmailRequest
{
    /**
     * @param Email[] $emails
     */
    public function __construct(
        public array $emails,
        public ?string $correlationId = null,
    ) {
    }
}
