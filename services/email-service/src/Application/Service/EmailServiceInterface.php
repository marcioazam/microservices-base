<?php

declare(strict_types=1);

namespace EmailService\Application\Service;

use EmailService\Application\DTO\EmailResult;
use EmailService\Domain\Entity\Email;

/**
 * Interface for email sending operations.
 */
interface EmailServiceInterface
{
    /**
     * Send a single email.
     */
    public function send(Email $email): EmailResult;

    /**
     * Queue an email for later delivery.
     */
    public function queue(Email $email): EmailResult;

    /**
     * Get email by ID.
     */
    public function getById(string $emailId): ?Email;

    /**
     * Cancel a queued email.
     */
    public function cancel(string $emailId): bool;
}
