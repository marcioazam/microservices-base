<?php

declare(strict_types=1);

namespace EmailService\Application\Service;

use EmailService\Application\DTO\AuditQuery;
use EmailService\Application\DTO\AuditResultSet;
use EmailService\Application\Util\PiiMasker;
use EmailService\Domain\Entity\AuditLog;
use EmailService\Infrastructure\Platform\LogEntry;
use EmailService\Infrastructure\Platform\LoggingClientInterface;
use EmailService\Infrastructure\Platform\LogLevel;
use EmailService\Infrastructure\Repository\AuditLogRepositoryInterface;

/**
 * Audit service using platform logging-service for centralized logging.
 */
final readonly class AuditService implements AuditServiceInterface
{
    public function __construct(
        private AuditLogRepositoryInterface $repository,
        private ?LoggingClientInterface $loggingClient = null,
    ) {}

    public function log(AuditLog $entry): void
    {
        // Save to repository
        $this->repository->save($entry);

        // Send to platform logging service
        $this->sendToLoggingService($entry);
    }

    public function query(AuditQuery $query): AuditResultSet
    {
        $items = $this->repository->findByQuery($query);
        $total = $this->repository->countByQuery($query);

        return new AuditResultSet(
            items: $items,
            total: $total,
            limit: $query->limit,
            offset: $query->offset
        );
    }

    public function getByEmailId(string $emailId): array
    {
        return $this->repository->findByEmailId($emailId);
    }

    private function sendToLoggingService(AuditLog $entry): void
    {
        if ($this->loggingClient === null) {
            return;
        }

        $level = $this->mapStatusToLogLevel($entry);
        $correlationId = $entry->emailId;

        $logEntry = new LogEntry(
            correlationId: $correlationId,
            serviceId: 'email-service',
            level: $level,
            message: $this->buildLogMessage($entry),
            metadata: $this->buildMetadata($entry),
        );

        $this->loggingClient->log($logEntry);
    }

    private function mapStatusToLogLevel(AuditLog $entry): LogLevel
    {
        return match ($entry->status->value) {
            'failed' => LogLevel::ERROR,
            'bounced' => LogLevel::WARN,
            default => LogLevel::INFO,
        };
    }

    private function buildLogMessage(AuditLog $entry): string
    {
        return sprintf(
            'Email %s: %s -> %s [%s]',
            $entry->action->value,
            PiiMasker::maskEmail($entry->recipientEmail),
            $entry->status->value,
            $entry->providerName ?? 'unknown'
        );
    }

    private function buildMetadata(AuditLog $entry): array
    {
        $metadata = [
            'audit_id' => $entry->id,
            'email_id' => $entry->emailId,
            'action' => $entry->action->value,
            'sender_id' => $entry->senderId,
            'recipient_masked' => PiiMasker::maskEmail($entry->recipientEmail),
            'subject' => $entry->subject,
            'status' => $entry->status->value,
            'timestamp' => $entry->timestamp->format(\DateTimeInterface::RFC3339),
        ];

        if ($entry->providerName !== null) {
            $metadata['provider'] = $entry->providerName;
        }

        if ($entry->messageId !== null) {
            $metadata['message_id'] = $entry->messageId;
        }

        if ($entry->errorMessage !== null) {
            $metadata['error_message'] = $entry->errorMessage;
        }

        if ($entry->attemptNumber !== null) {
            $metadata['attempt_number'] = $entry->attemptNumber;
        }

        return $metadata;
    }
}
