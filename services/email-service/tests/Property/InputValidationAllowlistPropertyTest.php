<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Application\DTO\RateLimitResult;
use EmailService\Application\Service\DnsResolverInterface;
use EmailService\Application\Service\ValidationService;
use EmailService\Application\DTO\DomainValidationResult;
use EmailService\Infrastructure\Platform\CacheClientInterface;
use EmailService\Infrastructure\Platform\CacheValue;
use EmailService\Infrastructure\Platform\CacheSource;
use EmailService\Infrastructure\RateLimiter\RateLimiterInterface;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Feature: email-service-modernization-2025
 * Property 7: Input Validation Allowlist
 * Validates: Requirements 8.1
 *
 * For any email address input, the ValidationService SHALL:
 * - Accept only addresses matching the RFC 5322 format
 * - Reject addresses with domains in the disposable domain list
 * - Reject addresses with domains lacking MX records
 */
final class InputValidationAllowlistPropertyTest extends TestCase
{
    use TestTrait;

    private ValidationService $service;
    private CacheClientInterface $cacheClient;
    private RateLimiterInterface $rateLimiter;
    private DnsResolverInterface $dnsResolver;

    protected function setUp(): void
    {
        $this->cacheClient = $this->createMock(CacheClientInterface::class);
        $this->rateLimiter = $this->createMock(RateLimiterInterface::class);
        $this->dnsResolver = $this->createMock(DnsResolverInterface::class);

        $this->rateLimiter->method('check')
            ->willReturn(new RateLimitResult(true, 100, 99, 60));

        $this->service = new ValidationService(
            $this->rateLimiter,
            $this->cacheClient,
            $this->dnsResolver
        );
    }

    /**
     * @test
     */
    public function validEmailsWithValidDomainsAreAccepted(): void
    {
        $this->forAll(
            Generator\elements([
                'user@example.com',
                'test.user@domain.org',
                'name+tag@company.net',
                'first.last@subdomain.example.com',
            ])
        )
            ->withMaxSize(100)
            ->then(function (string $email): void {
                $this->cacheClient->method('get')->willReturn(null);
                $this->cacheClient->method('set')->willReturn(true);
                $this->dnsResolver->method('getMxRecords')
                    ->willReturn(DomainValidationResult::valid(['mx.example.com']));

                $result = $this->service->validateEmail($email);

                $this->assertTrue($result->isValid, "Valid email {$email} should be accepted");
            });
    }

    /**
     * @test
     */
    public function invalidFormatEmailsAreRejected(): void
    {
        $this->forAll(
            Generator\elements([
                'notanemail',
                '@nodomain.com',
                'noat.com',
                'spaces in@email.com',
                'double@@at.com',
                '',
                'missing@',
                '@missing.local',
            ])
        )
            ->withMaxSize(100)
            ->then(function (string $email): void {
                $result = $this->service->validateEmail($email);

                $this->assertFalse($result->isValid, "Invalid format {$email} should be rejected");
                $this->assertEquals('INVALID_EMAIL_FORMAT', $result->errorCode);
            });
    }

    /**
     * @test
     */
    public function disposableEmailDomainsAreRejected(): void
    {
        $disposableDomains = [
            'tempmail.com',
            'throwaway.email',
            'guerrillamail.com',
            'mailinator.com',
            '10minutemail.com',
            'temp-mail.org',
            'fakeinbox.com',
            'trashmail.com',
            'getnada.com',
            'maildrop.cc',
            'yopmail.com',
        ];

        $this->forAll(
            Generator\elements($disposableDomains),
            Generator\suchThat(
                fn($s) => strlen($s) > 0 && preg_match('/^[a-z]+$/', $s) === 1,
                Generator\string()
            )
        )
            ->withMaxSize(100)
            ->then(function (string $domain, string $localPart): void {
                if (empty($localPart)) {
                    $localPart = 'user';
                }
                $email = "{$localPart}@{$domain}";

                $result = $this->service->validateEmail($email);

                $this->assertFalse($result->isValid, "Disposable email {$email} should be rejected");
                $this->assertEquals('DISPOSABLE_EMAIL', $result->errorCode);
            });
    }

    /**
     * @test
     */
    public function emailsWithInvalidDomainsAreRejected(): void
    {
        $this->forAll(
            Generator\elements([
                'user@nonexistent-domain-xyz123.com',
                'test@invalid-mx-domain.org',
                'name@no-records-here.net',
            ])
        )
            ->withMaxSize(100)
            ->then(function (string $email): void {
                $this->cacheClient->method('get')->willReturn(null);
                $this->cacheClient->method('set')->willReturn(true);
                $this->dnsResolver->method('getMxRecords')
                    ->willReturn(DomainValidationResult::invalid('No MX records found'));

                $result = $this->service->validateEmail($email);

                $this->assertFalse($result->isValid, "Email with invalid domain {$email} should be rejected");
                $this->assertEquals('INVALID_DOMAIN', $result->errorCode);
            });
    }

    /**
     * @test
     */
    public function domainValidationResultsAreCached(): void
    {
        $domain = 'cached-domain.com';
        $email = "user@{$domain}";

        $cachedResult = new CacheValue(
            [
                'isValid' => true,
                'mxRecords' => ['mx.cached-domain.com'],
                'errorMessage' => null,
            ],
            CacheSource::REDIS
        );

        $this->cacheClient->method('get')->willReturn($cachedResult);

        $result = $this->service->validateEmail($email);

        $this->assertTrue($result->isValid, 'Cached valid domain should be accepted');
    }

    /**
     * @test
     */
    public function attachmentSizeValidationEnforcesLimits(): void
    {
        $this->forAll(Generator\choose(1, 25 * 1024 * 1024))
            ->withMaxSize(100)
            ->then(function (int $size): void {
                $result = $this->service->validateAttachmentSize($size);
                $this->assertTrue($result, "Size {$size} should be valid");
            });

        $this->forAll(Generator\choose(25 * 1024 * 1024 + 1, 100 * 1024 * 1024))
            ->withMaxSize(100)
            ->then(function (int $size): void {
                $result = $this->service->validateAttachmentSize($size);
                $this->assertFalse($result, "Size {$size} should be invalid (exceeds 25MB)");
            });
    }

    /**
     * @test
     */
    public function recipientCountValidationEnforcesLimits(): void
    {
        $this->forAll(Generator\choose(1, 50))
            ->withMaxSize(100)
            ->then(function (int $count): void {
                $result = $this->service->validateRecipientCount($count);
                $this->assertTrue($result, "Count {$count} should be valid");
            });

        $this->forAll(Generator\choose(51, 1000))
            ->withMaxSize(100)
            ->then(function (int $count): void {
                $result = $this->service->validateRecipientCount($count);
                $this->assertFalse($result, "Count {$count} should be invalid (exceeds 50)");
            });
    }

    /**
     * @test
     */
    public function isDisposableDetectsAllKnownDisposableDomains(): void
    {
        $disposableDomains = [
            'tempmail.com', 'throwaway.email', 'guerrillamail.com', 'mailinator.com',
            '10minutemail.com', 'temp-mail.org', 'fakeinbox.com', 'trashmail.com',
            'getnada.com', 'maildrop.cc', 'yopmail.com', 'dispostable.com',
            'sharklasers.com', 'guerrillamail.info', 'grr.la', 'spam4.me',
            'tempail.com', 'emailondeck.com', 'mohmal.com', 'tempmailo.com',
        ];

        foreach ($disposableDomains as $domain) {
            $email = "test@{$domain}";
            $this->assertTrue(
                $this->service->isDisposable($email),
                "Domain {$domain} should be detected as disposable"
            );
        }
    }
}
