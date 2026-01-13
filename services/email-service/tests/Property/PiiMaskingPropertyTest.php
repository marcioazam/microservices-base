<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Application\Util\PiiMasker;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Feature: email-service-modernization-2025
 * Property 4: PII Masking Consistency
 * 
 * For any email address, the PiiMasker.maskEmail() function SHALL:
 * - Return a string that does NOT contain the full local part
 * - Preserve the domain portion after the @ symbol
 * - Start with the first character of the local part (if length > 1)
 * - Be idempotent: masking an already-masked email produces the same result
 * 
 * Validates: Requirements 2.2, 8.3
 */
class PiiMaskingPropertyTest extends TestCase
{
    use TestTrait;

    /**
     * @test
     * Property 4: Masked email does not contain full local part
     */
    public function maskedEmailDoesNotContainFullLocalPart(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 2 && strlen($s) <= 30 && preg_match('/^[a-z0-9.]+$/', $s),
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 3 && strlen($s) <= 20 && preg_match('/^[a-z0-9]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $localPart, string $domain): void {
            $email = "{$localPart}@{$domain}.com";
            $masked = PiiMasker::maskEmail($email);

            // Should not contain the full local part (if local part > 1 char)
            if (strlen($localPart) > 1) {
                $this->assertStringNotContainsString($localPart, $masked);
            }
        });
    }

    /**
     * @test
     * Property 4: Masked email preserves domain
     */
    public function maskedEmailPreservesDomain(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 30 && preg_match('/^[a-z0-9.]+$/', $s),
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 3 && strlen($s) <= 20 && preg_match('/^[a-z0-9]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $localPart, string $domain): void {
            $fullDomain = "{$domain}.com";
            $email = "{$localPart}@{$fullDomain}";
            $masked = PiiMasker::maskEmail($email);

            $this->assertStringContainsString("@{$fullDomain}", $masked);
        });
    }

    /**
     * @test
     * Property 4: Masked email starts with first character of local part
     */
    public function maskedEmailStartsWithFirstCharacterOfLocalPart(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 2 && strlen($s) <= 30 && preg_match('/^[a-z0-9.]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $localPart): void {
            $email = "{$localPart}@example.com";
            $masked = PiiMasker::maskEmail($email);

            $this->assertStringStartsWith($localPart[0], $masked);
        });
    }

    /**
     * @test
     * Property 4: Email masking is idempotent
     */
    public function emailMaskingIsIdempotent(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 30 && preg_match('/^[a-z0-9.]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $localPart): void {
            $email = "{$localPart}@example.com";
            
            $maskedOnce = PiiMasker::maskEmail($email);
            $maskedTwice = PiiMasker::maskEmail($maskedOnce);

            $this->assertEquals($maskedOnce, $maskedTwice);
        });
    }

    /**
     * @test
     * Property 4: Phone masking preserves last 4 digits
     */
    public function phoneMaskingPreservesLastFourDigits(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 5 && strlen($s) <= 15 && preg_match('/^[0-9]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $phone): void {
            $masked = PiiMasker::maskPhone($phone);
            $lastFour = substr($phone, -4);

            $this->assertStringEndsWith($lastFour, $masked);
        });
    }

    /**
     * @test
     * Property 4: Phone masking hides all but last 4 digits
     */
    public function phoneMaskingHidesAllButLastFourDigits(): void
    {
        $phone = '1234567890';
        $masked = PiiMasker::maskPhone($phone);

        $this->assertEquals('******7890', $masked);
        $this->assertStringNotContainsString('123456', $masked);
    }

    /**
     * @test
     * Property 4: Name masking preserves first character of each word
     */
    public function nameMaskingPreservesFirstCharacterOfEachWord(): void
    {
        $testCases = [
            'John' => 'J***',
            'John Doe' => 'J*** D**',
            'Alice Bob Charlie' => 'A**** B** C******',
        ];

        foreach ($testCases as $name => $expected) {
            $masked = PiiMasker::maskName($name);
            $this->assertEquals($expected, $masked);
        }
    }

    /**
     * @test
     * Property 4: Credit card masking preserves last 4 digits
     */
    public function creditCardMaskingPreservesLastFourDigits(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 13 && strlen($s) <= 19 && preg_match('/^[0-9]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $cardNumber): void {
            $masked = PiiMasker::maskCreditCard($cardNumber);
            $lastFour = substr($cardNumber, -4);

            $this->assertStringEndsWith($lastFour, $masked);
            
            // Should not contain any other digits from the card
            $firstPart = substr($cardNumber, 0, -4);
            foreach (str_split($firstPart) as $digit) {
                // The masked part should only contain asterisks
                $maskedPart = substr($masked, 0, -4);
                $this->assertStringNotContainsString($digit, $maskedPart);
            }
        });
    }

    /**
     * @test
     * Property 4: IPv4 masking preserves first two octets
     */
    public function ipv4MaskingPreservesFirstTwoOctets(): void
    {
        $testCases = [
            '192.168.1.100' => '192.168.*.*',
            '10.0.0.1' => '10.0.*.*',
            '172.16.254.1' => '172.16.*.*',
        ];

        foreach ($testCases as $ip => $expected) {
            $masked = PiiMasker::maskIpAddress($ip);
            $this->assertEquals($expected, $masked);
        }
    }

    /**
     * @test
     * Property 4: Sensitive data masking shows first and last characters
     */
    public function sensitiveDataMaskingShowsFirstAndLastCharacters(): void
    {
        $data = 'secretpassword';
        $masked = PiiMasker::maskSensitive($data, 2);

        $this->assertStringStartsWith('se', $masked);
        $this->assertStringEndsWith('rd', $masked);
        $this->assertStringNotContainsString('cretpasswo', $masked);
    }

    /**
     * @test
     * Property 4: Empty and edge case handling
     */
    public function emptyAndEdgeCaseHandling(): void
    {
        // Empty email local part
        $this->assertEquals('***', PiiMasker::maskEmail('invalid'));
        
        // Single character local part
        $this->assertEquals('*@example.com', PiiMasker::maskEmail('a@example.com'));
        
        // Empty name
        $this->assertEquals('', PiiMasker::maskName(''));
        
        // Short phone (not masked)
        $this->assertEquals('1234', PiiMasker::maskPhone('1234'));
    }

    /**
     * @test
     * Property 4: Masking is consistent across multiple calls
     */
    public function maskingIsConsistentAcrossMultipleCalls(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 3 && strlen($s) <= 20 && preg_match('/^[a-z0-9]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $localPart): void {
            $email = "{$localPart}@example.com";
            
            $masked1 = PiiMasker::maskEmail($email);
            $masked2 = PiiMasker::maskEmail($email);
            $masked3 = PiiMasker::maskEmail($email);

            $this->assertEquals($masked1, $masked2);
            $this->assertEquals($masked2, $masked3);
        });
    }
}
