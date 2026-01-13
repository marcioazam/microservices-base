<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Application\DTO\DomainValidationResult;
use EmailService\Application\DTO\RateLimitResult;
use EmailService\Application\Service\DnsResolverInterface;
use EmailService\Application\Service\ValidationService;
use EmailService\Infrastructure\Platform\CacheClientInterface;
use EmailService\Infrastructure\RateLimiter\RateLimiterInterface;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Property 7: MX Record Validation
 * For any email address with a syntactically valid format, the ValidationService
 * SHALL verify the domain has at least one valid MX record before accepting.
 *
 * Property 8: Disposable Domain Rejection
 * For any email address with a domain in the disposable domain list,
 * the ValidationService SHALL reject the address with error code DISPOSABLE_EMAIL.
 *
 * Property 10: Validation Error Specificity
 * For any invalid email request, the error response SHALL contain a specific
 * error code that uniquely identifies the validation failure type.
 *
 * Validates: Requirements 2.2, 2.3, 2.5
 */
final class EmailValidationPropertyTest extends TestCase
{
    use TestTrait;

    private ValidationService $validationService;
    private RateLimiterInterface $rateLimiter;
    private CacheClientInterface $cacheClient;
    private DnsResolverInterface $dnsResolver;

    protected function setUp(): void
    {
        $this->rateLimiter = $this->createMock(RateLimiterInterface::class);
        $this->rateLimiter->method('check')->willReturn(
            new RateLimitResult(true, 100, 100, 60)
        );

        $this->cacheClient = $this->createMock(CacheClientInterface::class);
        $this->cacheClient->method('get')->willReturn(null);
        $this->cacheClient->method('set')->willReturn(true);

        $this->dnsResolver = $this->createMock(DnsResolverInterface::class);
    }

    /**
     * @test
     * Property 7: Valid domains with MX records are accepted
     */
    public function validDomainsWithMxRecordsAreAccepted(): void
    {
        $this->dnsResolver->method('getMxRecords')->willReturn(
            DomainValidationResult::valid(['mx1.example.com', 'mx2.example.com'])
        );

        $this->validationService = new ValidationService(
            $this->rateLimiter,
            $this->cacheClient,
            $this->dnsResolver
        );

        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            )
        )
            ->withMaxSize(100)
            ->then(function (string $local): void {
                $email = "{$local}@validexample.com";

                $result = $this->validationService->validateEmail($email);

                $this->assertTrue($result->isValid);
                $this->assertNull($result->errorCode);
            });
    }

    /**
     * @test
     * Property 7: Domains without MX records are rejected
     */
    public function domainsWithoutMxRecordsAreRejected(): void
    {
        $this->dnsResolver->method('getMxRecords')->willReturn(
            DomainValidationResult::invalid('No MX records found')
        );

        $this->validationService = new ValidationService(
            $this->rateLimiter,
            $this->cacheClient,
            $this->dnsResolver
        );

        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            )
        )
            ->withMaxSize(100)
            ->then(function (string $local): void {
                $email = "{$local}@invaliddomain.invalid";

                $result = $this->validationService->validateEmail($email);

                $this->assertFalse($result->isValid);
                $this->assertEquals('INVALID_DOMAIN', $result->errorCode);
            });
    }

    /**
     * @test
     * Property 8: Disposable email domains are rejected
     */
    public function disposableEmailDomainsAreRejected(): void
    {
        $disposableDomains = [
            'tempmail.com', 'throwaway.email', 'guerrillamail.com',
            'mailinator.com', '10minutemail.com', 'yopmail.com'
        ];

        $this->dnsResolver->method('getMxRecords')->willReturn(
            DomainValidationResult::valid(['mx.disposable.com'])
        );

        $this->validationService = new ValidationService(
            $this->rateLimiter,
            $this->cacheClient,
            $this->dnsResolver
        );

        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            ),
            Generator\elements($disposableDomains)
        )
            ->withMaxSize(100)
            ->then(function (string $local, string $domain): void {
                $email = "{$local}@{$domain}";

                $result = $this->validationService->validateEmail($email);

                $this->assertFalse($result->isValid);
                $this->assertEquals('DISPOSABLE_EMAIL', $result->errorCode);
            });
    }

    /**
     * @test
     * Property 10: Invalid format returns INVALID_EMAIL_FORMAT error code
     */
    public function invalidFormatReturnsSpecificErrorCode(): void
    {
        $this->validationService = new ValidationService(
            $this->rateLimiter,
            $this->cacheClient,
            $this->dnsResolver
        );

        $invalidEmails = [
            'notanemail',
            '@nodomain.com',
            'no@',
            'spaces in@email.com',
            'double@@at.com',
        ];

        foreach ($invalidEmails as $email) {
            $result = $this->validationService->validateEmail($email);

            $this->assertFalse($result->isValid);
            $this->assertEquals('INVALID_EMAIL_FORMAT', $result->errorCode);
            $this->assertNotEmpty($result->errorMessage);
        }
    }

    /**
     * @test
     * Property 10: Each validation failure type has unique error code
     */
    public function eachValidationFailureHasUniqueErrorCode(): void
    {
        $errorCodes = [];

        // Invalid format
        $this->validationService = new ValidationService(
            $this->rateLimiter,
            $this->cacheClient,
            $this->dnsResolver
        );
        $result = $this->validationService->validateEmail('invalid');
        $errorCodes[] = $result->errorCode;

        // Disposable domain
        $dnsResolver = $this->createMock(DnsResolverInterface::class);
        $dnsResolver->method('getMxRecords')->willReturn(
            DomainValidationResult::valid(['mx.example.com'])
        );
        $this->validationService = new ValidationService(
            $this->rateLimiter,
            $this->cacheClient,
            $dnsResolver
        );
        $result = $this->validationService->validateEmail('test@tempmail.com');
        $errorCodes[] = $result->errorCode;

        // Invalid domain
        $dnsResolver2 = $this->createMock(DnsResolverInterface::class);
        $dnsResolver2->method('getMxRecords')->willReturn(
            DomainValidationResult::invalid('No MX records')
        );
        $this->validationService = new ValidationService(
            $this->rateLimiter,
            $this->cacheClient,
            $dnsResolver2
        );
        $result = $this->validationService->validateEmail('test@nonexistent.invalid');
        $errorCodes[] = $result->errorCode;

        // All error codes should be unique
        $this->assertEquals(count($errorCodes), count(array_unique($errorCodes)));
    }

    /**
     * @test
     * Property: isDisposable correctly identifies disposable domains
     */
    public function isDisposableCorrectlyIdentifiesDisposableDomains(): void
    {
        $this->validationService = new ValidationService(
            $this->rateLimiter,
            $this->cacheClient,
            $this->dnsResolver
        );

        $this->forAll(
            Generator\elements([
                'tempmail.com', 'mailinator.com', 'guerrillamail.com',
                '10minutemail.com', 'yopmail.com', 'trashmail.com'
            ])
        )
            ->then(function (string $domain): void {
                $email = "test@{$domain}";

                $this->assertTrue($this->validationService->isDisposable($email));
            });
    }

    /**
     * @test
     * Property: Non-disposable domains are not flagged
     */
    public function nonDisposableDomainsAreNotFlagged(): void
    {
        $this->validationService = new ValidationService(
            $this->rateLimiter,
            $this->cacheClient,
            $this->dnsResolver
        );

        $legitimateDomains = [
            'gmail.com', 'yahoo.com', 'outlook.com', 'hotmail.com',
            'company.com', 'university.edu', 'government.gov'
        ];

        foreach ($legitimateDomains as $domain) {
            $email = "test@{$domain}";

            $this->assertFalse($this->validationService->isDisposable($email));
        }
    }
}
