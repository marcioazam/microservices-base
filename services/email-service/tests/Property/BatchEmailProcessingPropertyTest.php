<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Application\DTO\BatchEmailRequest;
use EmailService\Application\DTO\EmailResult;
use EmailService\Application\DTO\ValidationResult;
use EmailService\Application\Service\BatchEmailService;
use EmailService\Application\Service\EmailServiceInterface;
use EmailService\Application\Service\ValidationServiceInterface;
use EmailService\Domain\Entity\Email;
use EmailService\Domain\Enum\EmailStatus;
use EmailService\Domain\ValueObject\Recipient;
use EmailService\Infrastructure\Observability\TracingServiceInterface;
use EmailService\Infrastructure\Platform\LoggingClientInterface;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Feature: email-service-modernization-2025
 * Property 8: Batch Email Processing Completeness
 * Validates: Requirements 9.2
 *
 * For any batch of N valid emails submitted for processing, the batch processor SHALL:
 * - Process all N emails
 * - Return N results (success or failure for each)
 * - Maintain the order correspondence between input and output
 */
final class BatchEmailProcessingPropertyTest extends TestCase
{
    use TestTrait;

    private BatchEmailService $batchService;
    private EmailServiceInterface $emailService;
    private ValidationServiceInterface $validationService;
    private LoggingClientInterface $loggingClient;
    private TracingServiceInterface $tracingService;

    protected function setUp(): void
    {
        $this->emailService = $this->createMock(EmailServiceInterface::class);
        $this->validationService = $this->createMock(ValidationServiceInterface::class);
        $this->loggingClient = $this->createMock(LoggingClientInterface::class);
        $this->tracingService = $this->createMock(TracingServiceInterface::class);

        $this->loggingClient->method('log')->willReturn(null);

        $this->batchService = new BatchEmailService(
            $this->emailService,
            $this->validationService,
            $this->loggingClient,
            $this->tracingService
        );
    }

    /**
     * @test
     */
    public function batchProcessingReturnsResultForEveryInput(): void
    {
        $this->forAll(Generator\choose(1, 50))
            ->withMaxSize(100)
            ->then(function (int $batchSize): void {
                $emails = $this->createEmailBatch($batchSize);

                $this->validationService->method('validateEmail')
                    ->willReturn(ValidationResult::valid());

                $this->emailService->method('send')
                    ->willReturn(EmailResult::success('test', 'msg-123'));

                $request = new BatchEmailRequest($emails, 'test-correlation');
                $result = $this->batchService->processBatch($request);

                $this->assertCount(
                    $batchSize,
                    $result->results,
                    "Batch of {$batchSize} emails should return {$batchSize} results"
                );
                $this->assertEquals($batchSize, $result->totalCount);
            });
    }

    /**
     * @test
     */
    public function batchProcessingMaintainsInputOutputOrder(): void
    {
        $this->forAll(Generator\choose(2, 20))
            ->withMaxSize(100)
            ->then(function (int $batchSize): void {
                $emails = $this->createEmailBatch($batchSize);
                $emailIds = array_map(fn(Email $e) => $e->id, $emails);

                $this->validationService->method('validateEmail')
                    ->willReturn(ValidationResult::valid());

                $this->emailService->method('send')
                    ->willReturnCallback(fn(Email $e) => EmailResult::success($e->id, 'msg-' . $e->id));

                $request = new BatchEmailRequest($emails, 'test-correlation');
                $result = $this->batchService->processBatch($request);

                foreach ($result->results as $index => $emailResult) {
                    $this->assertEquals(
                        $emailIds[$index],
                        $emailResult->emailId,
                        "Result at index {$index} should match input email ID"
                    );
                    $this->assertEquals($index, $emailResult->index);
                }
            });
    }

    /**
     * @test
     */
    public function batchProcessingCountsSuccessesAndFailuresCorrectly(): void
    {
        $this->forAll(
            Generator\choose(1, 20),
            Generator\choose(0, 10)
        )
            ->withMaxSize(100)
            ->then(function (int $successCount, int $failureCount): void {
                $totalCount = $successCount + $failureCount;
                $emails = $this->createEmailBatch($totalCount);

                $callCount = 0;
                $this->validationService->method('validateEmail')
                    ->willReturnCallback(function () use (&$callCount, $successCount): ValidationResult {
                        $callCount++;
                        return $callCount <= $successCount
                            ? ValidationResult::valid()
                            : ValidationResult::invalidFormat();
                    });

                $this->emailService->method('send')
                    ->willReturn(EmailResult::success('test', 'msg-123'));

                $request = new BatchEmailRequest($emails, 'test-correlation');
                $result = $this->batchService->processBatch($request);

                $this->assertEquals($successCount, $result->successCount);
                $this->assertEquals($failureCount, $result->failureCount);
                $this->assertEquals($totalCount, $result->totalCount);
            });
    }

    /**
     * @test
     */
    public function batchProcessingRejectsOversizedBatches(): void
    {
        $maxSize = $this->batchService->getMaxBatchSize();
        $emails = $this->createEmailBatch($maxSize + 1);

        $request = new BatchEmailRequest($emails, 'test-correlation');
        $result = $this->batchService->processBatch($request);

        $this->assertNotNull($result->errorMessage);
        $this->assertStringContainsString((string)$maxSize, $result->errorMessage);
    }

    /**
     * @test
     */
    public function batchProcessingIncludesCorrelationId(): void
    {
        $this->forAll(Generator\elements([
            'corr-123',
            'batch-456',
            'request-789',
        ]))
            ->withMaxSize(100)
            ->then(function (string $correlationId): void {
                $emails = $this->createEmailBatch(3);

                $this->validationService->method('validateEmail')
                    ->willReturn(ValidationResult::valid());

                $this->emailService->method('send')
                    ->willReturn(EmailResult::success('test', 'msg-123'));

                $request = new BatchEmailRequest($emails, $correlationId);
                $result = $this->batchService->processBatch($request);

                $this->assertEquals($correlationId, $result->correlationId);
            });
    }

    /**
     * @test
     */
    public function batchProcessingGeneratesCorrelationIdWhenNotProvided(): void
    {
        $emails = $this->createEmailBatch(2);

        $this->validationService->method('validateEmail')
            ->willReturn(ValidationResult::valid());

        $this->emailService->method('send')
            ->willReturn(EmailResult::success('test', 'msg-123'));

        $request = new BatchEmailRequest($emails, null);
        $result = $this->batchService->processBatch($request);

        $this->assertNotEmpty($result->correlationId);
        $this->assertMatchesRegularExpression(
            '/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/',
            $result->correlationId
        );
    }

    /**
     * @param int $count
     * @return Email[]
     */
    private function createEmailBatch(int $count): array
    {
        $emails = [];
        for ($i = 0; $i < $count; $i++) {
            $emails[] = new Email(
                id: "email-{$i}",
                from: new Recipient("sender@example.com", "Sender"),
                recipients: [new Recipient("recipient{$i}@example.com", "Recipient {$i}")],
                subject: "Test Subject {$i}",
                body: "Test body {$i}",
                status: EmailStatus::PENDING,
            );
        }
        return $emails;
    }
}
