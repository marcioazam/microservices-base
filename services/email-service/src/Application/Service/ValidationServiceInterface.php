<?php

declare(strict_types=1);

namespace EmailService\Application\Service;

use EmailService\Application\DTO\ValidationResult;
use EmailService\Application\DTO\DomainValidationResult;
use EmailService\Application\DTO\RateLimitResult;

interface ValidationServiceInterface
{
    /**
     * Validate email address format and domain
     */
    public function validateEmail(string $email): ValidationResult;

    /**
     * Validate domain has valid MX records
     */
    public function validateDomain(string $domain): DomainValidationResult;

    /**
     * Check if email is from a disposable domain
     */
    public function isDisposable(string $email): bool;

    /**
     * Check rate limit for sender
     */
    public function checkRateLimit(string $senderId): RateLimitResult;
}
