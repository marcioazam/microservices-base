<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Domain\Entity\Email;
use EmailService\Domain\Enum\ContentType;
use EmailService\Domain\Exception\AttachmentTooLargeException;
use EmailService\Domain\ValueObject\Attachment;
use EmailService\Domain\ValueObject\Recipient;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Property 3: Attachment Size Validation
 * For any email request with attachments, IF total attachment size is ≤ 25MB
 * THEN attachments SHALL be included, ELSE the request SHALL be rejected.
 * 
 * Validates: Requirements 1.3
 */
class AttachmentSizePropertyTest extends TestCase
{
    use TestTrait;

    private const MAX_SIZE = 26214400; // 25MB

    /**
     * @test
     * Property: Attachments under 25MB are accepted
     */
    public function attachmentsUnderMaxSizeAreAccepted(): void
    {
        $this->forAll(
            Generator\choose(1, 1000000) // 1 byte to 1MB
        )
        ->withMaxSize(100)
        ->then(function (int $size): void {
            $content = str_repeat('a', $size);
            
            $attachment = new Attachment(
                filename: 'test.txt',
                content: $content,
                mimeType: 'text/plain',
                size: $size
            );
            
            $this->assertEquals($size, $attachment->size);
            $this->assertEquals('test.txt', $attachment->filename);
        });
    }

    /**
     * @test
     * Property: Attachments over 25MB are rejected
     */
    public function attachmentsOverMaxSizeAreRejected(): void
    {
        $this->forAll(
            Generator\choose(self::MAX_SIZE + 1, self::MAX_SIZE + 1000000)
        )
        ->withMaxSize(50)
        ->then(function (int $size): void {
            $this->expectException(AttachmentTooLargeException::class);
            
            new Attachment(
                filename: 'large.txt',
                content: str_repeat('a', min($size, 1000)), // Don't actually allocate huge memory
                mimeType: 'text/plain',
                size: $size
            );
        });
    }

    /**
     * @test
     * Property: Multiple attachments with total under 25MB are accepted
     */
    public function multipleAttachmentsUnderTotalMaxAreAccepted(): void
    {
        $this->forAll(
            Generator\choose(1, 5), // Number of attachments
            Generator\choose(1000, 100000) // Size per attachment
        )
        ->withMaxSize(100)
        ->then(function (int $count, int $sizeEach): void {
            $totalSize = $count * $sizeEach;
            
            if ($totalSize > self::MAX_SIZE) {
                $this->markTestSkipped('Total size exceeds max');
                return;
            }
            
            $attachments = [];
            for ($i = 0; $i < $count; $i++) {
                $attachments[] = new Attachment(
                    filename: "file{$i}.txt",
                    content: str_repeat('a', $sizeEach),
                    mimeType: 'text/plain',
                    size: $sizeEach
                );
            }
            
            // Should not throw
            Attachment::validateTotalSize($attachments);
            
            $this->assertCount($count, $attachments);
        });
    }

    /**
     * @test
     * Property: Email with valid attachments preserves them
     */
    public function emailWithValidAttachmentsPreservesThem(): void
    {
        $this->forAll(
            Generator\choose(1, 3),
            Generator\choose(100, 10000)
        )
        ->withMaxSize(100)
        ->then(function (int $count, int $sizeEach): void {
            $attachments = [];
            for ($i = 0; $i < $count; $i++) {
                $attachments[] = new Attachment(
                    filename: "file{$i}.txt",
                    content: str_repeat('x', $sizeEach),
                    mimeType: 'text/plain',
                    size: $sizeEach
                );
            }
            
            $email = Email::create(
                from: new Recipient('sender@example.com'),
                recipients: [new Recipient('recipient@example.com')],
                subject: 'Test',
                body: 'Body',
                contentType: ContentType::PLAIN,
                attachments: $attachments
            );
            
            $this->assertCount($count, $email->getAttachments());
            $this->assertEquals($count * $sizeEach, $email->getTotalAttachmentSize());
            $this->assertTrue($email->hasAttachments());
        });
    }

    /**
     * @test
     * Property: Attachment exactly at 25MB limit is accepted
     */
    public function attachmentAtExactLimitIsAccepted(): void
    {
        // Test boundary condition
        $attachment = new Attachment(
            filename: 'exact.txt',
            content: 'test',
            mimeType: 'text/plain',
            size: self::MAX_SIZE
        );
        
        $this->assertEquals(self::MAX_SIZE, $attachment->size);
    }

    /**
     * @test
     * Property: Base64 attachment decoding preserves content
     */
    public function base64AttachmentDecodingPreservesContent(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 1000,
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $content): void {
            $base64 = base64_encode($content);
            
            $attachment = Attachment::fromBase64(
                filename: 'test.bin',
                base64Content: $base64,
                mimeType: 'application/octet-stream'
            );
            
            $this->assertEquals($content, $attachment->content);
            $this->assertEquals(strlen($content), $attachment->size);
            $this->assertEquals($base64, $attachment->getBase64Content());
        });
    }
}
