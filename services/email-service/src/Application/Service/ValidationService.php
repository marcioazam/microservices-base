<?php

declare(strict_types=1);

namespace EmailService\Application\Service;

use EmailService\Application\DTO\DomainValidationResult;
use EmailService\Application\DTO\RateLimitResult;
use EmailService\Application\DTO\ValidationResult;
use EmailService\Domain\ValueObject\Recipient;
use EmailService\Infrastructure\Platform\CacheClientInterface;
use EmailService\Infrastructure\RateLimiter\RateLimiterInterface;

/**
 * Centralized validation service for email operations.
 * Uses CacheClient for domain validation caching.
 */
final readonly class ValidationService implements ValidationServiceInterface
{
    private const DISPOSABLE_DOMAINS = [
        'tempmail.com', 'throwaway.email', 'guerrillamail.com', 'mailinator.com',
        '10minutemail.com', 'temp-mail.org', 'fakeinbox.com', 'trashmail.com',
        'getnada.com', 'maildrop.cc', 'yopmail.com', 'dispostable.com',
        'sharklasers.com', 'guerrillamail.info', 'grr.la', 'spam4.me',
        'tempail.com', 'emailondeck.com', 'mohmal.com', 'tempmailo.com',
    ];

    private const DOMAIN_CACHE_TTL = 3600; // 1 hour
    private const DOMAIN_CACHE_NAMESPACE = 'email:domain';

    public function __construct(
        private RateLimiterInterface $rateLimiter,
        private CacheClientInterface $cacheClient,
        private ?DnsResolverInterface $dnsResolver = null,
    ) {
    }

    public function validateEmail(string $email): ValidationResult
    {
        if (!Recipient::isValidFormat($email)) {
            return ValidationResult::invalidFormat();
        }

        if ($this->isDisposable($email)) {
            return ValidationResult::disposableEmail();
        }

        $domain = $this->extractDomain($email);
        $domainResult = $this->validateDomain($domain);

        if (!$domainResult->isValid) {
            return ValidationResult::invalidDomain();
        }

        return ValidationResult::valid();
    }

    public function validateDomain(string $domain): DomainValidationResult
    {
        $normalizedDomain = strtolower(trim($domain));
        $cacheKey = "mx:{$normalizedDomain}";

        // Try cache first
        $cached = $this->cacheClient->get($cacheKey, self::DOMAIN_CACHE_NAMESPACE);
        if ($cached !== null) {
            return $this->deserializeDomainResult($cached->value);
        }

        // Perform DNS lookup
        $result = $this->performDnsLookup($normalizedDomain);

        // Cache the result
        $this->cacheClient->set(
            $cacheKey,
            $this->serializeDomainResult($result),
            self::DOMAIN_CACHE_TTL,
            self::DOMAIN_CACHE_NAMESPACE
        );

        return $result;
    }

    public function isDisposable(string $email): bool
    {
        $domain = $this->extractDomain($email);
        return in_array(strtolower($domain), self::DISPOSABLE_DOMAINS, true);
    }

    public function checkRateLimit(string $senderId): RateLimitResult
    {
        return $this->rateLimiter->check($senderId);
    }

    public function validateAttachmentSize(int $sizeBytes): bool
    {
        $maxSize = 25 * 1024 * 1024; // 25MB
        return $sizeBytes > 0 && $sizeBytes <= $maxSize;
    }

    public function validateRecipientCount(int $count): bool
    {
        $maxRecipients = 50;
        return $count > 0 && $count <= $maxRecipients;
    }

    private function performDnsLookup(string $domain): DomainValidationResult
    {
        if ($this->dnsResolver !== null) {
            return $this->dnsResolver->getMxRecords($domain);
        }

        $mxRecords = [];
        if (!getmxrr($domain, $mxRecords)) {
            $aRecord = gethostbyname($domain);
            if ($aRecord === $domain) {
                return DomainValidationResult::invalid("No MX or A records found for domain: {$domain}");
            }
            return DomainValidationResult::valid([$aRecord]);
        }

        return DomainValidationResult::valid($mxRecords);
    }

    private function extractDomain(string $email): string
    {
        $parts = explode('@', $email);
        return $parts[1] ?? '';
    }

    /**
     * @return array{isValid: bool, mxRecords: string[], errorMessage: ?string}
     */
    private function serializeDomainResult(DomainValidationResult $result): array
    {
        return [
            'isValid' => $result->isValid,
            'mxRecords' => $result->mxRecords,
            'errorMessage' => $result->errorMessage,
        ];
    }

    /**
     * @param array{isValid: bool, mxRecords: string[], errorMessage: ?string} $data
     */
    private function deserializeDomainResult(array $data): DomainValidationResult
    {
        if ($data['isValid']) {
            return DomainValidationResult::valid($data['mxRecords']);
        }
        return DomainValidationResult::invalid($data['errorMessage'] ?? 'Unknown error');
    }
}
