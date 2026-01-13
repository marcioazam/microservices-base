<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Domain\Exception\InvalidEmailException;
use EmailService\Domain\ValueObject\Recipient;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Property 6: RFC 5322 Email Validation
 * For any string submitted as an email address, the ValidationService SHALL return
 * valid=true if and only if the string conforms to RFC 5322 format specification.
 * 
 * Validates: Requirements 2.1
 */
class RecipientValidationPropertyTest extends TestCase
{
    use TestTrait;

    /**
     * @test
     * Property: Valid emails with standard format are accepted
     */
    public function validEmailsAreAccepted(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 2 && strlen($s) <= 10 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            ),
            Generator\elements(['com', 'org', 'net', 'io', 'dev'])
        )
        ->withMaxSize(100)
        ->then(function (string $local, string $domain, string $tld): void {
            $email = "{$local}@{$domain}.{$tld}";
            
            $recipient = new Recipient($email);
            
            $this->assertEquals(strtolower($email), $recipient->email);
            $this->assertEquals("{$domain}.{$tld}", $recipient->getDomain());
            $this->assertEquals($local, $recipient->getLocalPart());
        });
    }

    /**
     * @test
     * Property: Emails without @ symbol are rejected
     */
    public function emailsWithoutAtSymbolAreRejected(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 50 && strpos($s, '@') === false,
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $invalidEmail): void {
            $this->expectException(InvalidEmailException::class);
            new Recipient($invalidEmail);
        });
    }

    /**
     * @test
     * Property: Emails without domain part are rejected
     */
    public function emailsWithoutDomainAreRejected(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $local): void {
            $invalidEmail = "{$local}@";
            
            $this->expectException(InvalidEmailException::class);
            new Recipient($invalidEmail);
        });
    }

    /**
     * @test
     * Property: Empty strings are rejected
     */
    public function emptyStringsAreRejected(): void
    {
        $this->expectException(InvalidEmailException::class);
        new Recipient('');
    }

    /**
     * @test
     * Property: Whitespace-only strings are rejected
     */
    public function whitespaceOnlyStringsAreRejected(): void
    {
        $this->forAll(
            Generator\elements(['   ', "\t", "\n", "  \t  ", "\r\n"])
        )
        ->then(function (string $whitespace): void {
            $this->expectException(InvalidEmailException::class);
            new Recipient($whitespace);
        });
    }

    /**
     * @test
     * Property: Email addresses are normalized to lowercase
     */
    public function emailsAreNormalizedToLowercase(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-zA-Z]+$/', $s),
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 2 && strlen($s) <= 10 && preg_match('/^[a-zA-Z]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $local, string $domain): void {
            $mixedCaseEmail = "{$local}@{$domain}.com";
            
            $recipient = new Recipient($mixedCaseEmail);
            
            $this->assertEquals(strtolower($mixedCaseEmail), $recipient->email);
        });
    }

    /**
     * @test
     * Property: Recipient with name formats correctly
     */
    public function recipientWithNameFormatsCorrectly(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 30 && preg_match('/^[a-zA-Z ]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $local, string $name): void {
            $email = "{$local}@example.com";
            
            $recipient = new Recipient($email, $name);
            
            $this->assertStringContainsString($email, $recipient->getFormatted());
            $this->assertStringContainsString(trim($name), $recipient->getFormatted());
        });
    }

    /**
     * @test
     * Property: Two recipients with same email are equal
     */
    public function recipientsWithSameEmailAreEqual(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $local): void {
            $email = "{$local}@example.com";
            
            $recipient1 = new Recipient($email, 'Name 1');
            $recipient2 = new Recipient($email, 'Name 2');
            
            $this->assertTrue($recipient1->equals($recipient2));
        });
    }
}
