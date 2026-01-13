<?php

declare(strict_types=1);

namespace EmailService\Application\Service;

use EmailService\Application\DTO\BatchEmailRequest;
use EmailService\Application\DTO\BatchEmailResult;
use EmailService\Application\DTO\EmailResult;
use EmailService\Domain\Entity\Email;
use EmailService\Infrastructure\Observability\TracingServiceInterface;
use EmailService\Infrastructure\Platform\LoggingClientInterface;
use EmailService\Infrastructure\Platform\LogEntry;
use EmailService\Infrastructure\Platform\LogLevel;

/**
 * Batch email processing service.
 * Processes multiple emails in a single operation, maintaining input order.
 */
final readonly class BatchEmailService implements BatchEmailServiceInterface
{
    private const MAX_BATCH_SIZE = 100;
    private const SERVICE_ID = 'email-service';

    public function __construct(
        private EmailServiceInterface $emailService,
        private ValidationServiceInterface $validationService,
        private LoggingClientInterface $loggingClient,
        private ?TracingServiceInterface $tracingService = null,
    ) {
    }

    public function processBatch(BatchEmailRequest $request): BatchEmailResult
    {
        $correlationId = $request->correlationId ?? $this->generateCorrelationId();
        $emails = $request->emails;

        if (count($emails) > self::MAX_BATCH_SIZE) {
            return BatchEmailResult::error(
                "Batch size exceeds maximum of " . self::MAX_BATCH_SIZE,
                $correlationId
            );
        }

        $results = [];
        $successCount = 0;
        $failureCount = 0;

        $span = $this->tracingService?->startSpan('email.batch.process', [
            'batch.size' => count($emails),
            'correlation_id' => $correlationId,
        ]);

        try {
            foreach ($emails as $index => $email) {
                $result = $this->processEmail($email, $correlationId, $index);
                $results[] = $result;

                if ($result->isSuccess) {
                    $successCount++;
                } else {
                    $failureCount++;
                }
            }

            $this->tracingService?->endSpanSuccess($span, [
                'batch.success_count' => $successCount,
                'batch.failure_count' => $failureCount,
            ]);
        } catch (\Throwable $e) {
            $this->tracingService?->endSpanError($span, $e);
            throw $e;
        }

        $this->logBatchCompletion($correlationId, $successCount, $failureCount);

        return new BatchEmailResult(
            correlationId: $correlationId,
            results: $results,
            totalCount: count($emails),
            successCount: $successCount,
            failureCount: $failureCount,
        );
    }

    public function getMaxBatchSize(): int
    {
        return self::MAX_BATCH_SIZE;
    }

    private function processEmail(Email $email, string $correlationId, int $index): EmailResult
    {
        try {
            // Validate first
            $validationResult = $this->validationService->validateEmail(
                $email->recipients[0]->email ?? ''
            );

            if (!$validationResult->isValid) {
                return EmailResult::failure(
                    emailId: $email->id,
                    errorCode: $validationResult->errorCode ?? 'VALIDATION_ERROR',
                    errorMessage: $validationResult->errorMessage ?? 'Validation failed',
                    index: $index,
                );
            }

            // Send email
            $result = $this->emailService->send($email);

            return EmailResult::success(
                emailId: $email->id,
                messageId: $result->messageId ?? null,
                index: $index,
            );
        } catch (\Throwable $e) {
            $this->logEmailError($email->id, $correlationId, $e);

            return EmailResult::failure(
                emailId: $email->id,
                errorCode: 'PROCESSING_ERROR',
                errorMessage: $e->getMessage(),
                index: $index,
            );
        }
    }

    private function logBatchCompletion(string $correlationId, int $success, int $failure): void
    {
        $this->loggingClient->log(new LogEntry(
            correlationId: $correlationId,
            serviceId: self::SERVICE_ID,
            level: LogLevel::INFO,
            message: "Batch processing completed: {$success} success, {$failure} failures",
            metadata: [
                'success_count' => $success,
                'failure_count' => $failure,
            ],
        ));
    }

    private function logEmailError(string $emailId, string $correlationId, \Throwable $e): void
    {
        $this->loggingClient->log(new LogEntry(
            correlationId: $correlationId,
            serviceId: self::SERVICE_ID,
            level: LogLevel::ERROR,
            message: "Email processing failed: {$emailId}",
            metadata: ['email_id' => $emailId],
            exception: \EmailService\Infrastructure\Platform\ExceptionInfo::fromThrowable($e),
        ));
    }

    private function generateCorrelationId(): string
    {
        return sprintf(
            '%04x%04x-%04x-%04x-%04x-%04x%04x%04x',
            mt_rand(0, 0xffff), mt_rand(0, 0xffff),
            mt_rand(0, 0xffff),
            mt_rand(0, 0x0fff) | 0x4000,
            mt_rand(0, 0x3fff) | 0x8000,
            mt_rand(0, 0xffff), mt_rand(0, 0xffff), mt_rand(0, 0xffff)
        );
    }
}
