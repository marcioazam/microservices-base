<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Provider;

use EmailService\Domain\Entity\Email;

interface EmailProviderInterface
{
    /**
     * Send an email through this provider
     */
    public function send(Email $email): ProviderResult;

    /**
     * Get delivery status for a message
     */
    public function getDeliveryStatus(string $messageId): DeliveryStatus;

    /**
     * Get provider name
     */
    public function getName(): string;

    /**
     * Check if provider is healthy
     */
    public function isHealthy(): bool;
}
