<?php

declare(strict_types=1);

namespace EmailService\Domain\Entity;

use DateTimeImmutable;
use EmailService\Domain\Enum\AuditAction;
use EmailService\Domain\Enum\EmailStatus;
use Symfony\Component\Uid\Uuid;

class AuditLog
{
    /**
     * @param array<string, mixed> $metadata
     */
    public function __construct(
        public readonly string $id,
        public readonly string $emailId,
        public readonly AuditAction $action,
        public readonly string $senderId,
        public readonly string $recipientEmail,
        public readonly string $subject,
        public readonly EmailStatus $status,
        public readonly ?string $errorMessage = null,
        public readonly ?string $providerName = null,
        public readonly ?string $providerMessageId = null,
        public readonly array $metadata = [],
        public readonly DateTimeImmutable $timestamp = new DateTimeImmutable()
    ) {
    }

    public static function forEmailCreated(
        Email $email,
        string $senderId,
        array $metadata = []
    ): self {
        return new self(
            id: Uuid::v4()->toRfc4122(),
            emailId: $email->id,
            action: AuditAction::CREATED,
            senderId: $senderId,
            recipientEmail: self::maskEmail($email->getRecipients()[0]->email),
            subject: $email->subject,
            status: $email->getStatus(),
            metadata: $metadata
        );
    }

    public static function forEmailQueued(
        Email $email,
        string $senderId,
        array $metadata = []
    ): self {
        return new self(
            id: Uuid::v4()->toRfc4122(),
            emailId: $email->id,
            action: AuditAction::QUEUED,
            senderId: $senderId,
            recipientEmail: self::maskEmail($email->getRecipients()[0]->email),
            subject: $email->subject,
            status: EmailStatus::QUEUED,
            metadata: $metadata
        );
    }

    public static function forEmailSent(
        Email $email,
        string $senderId,
        string $providerName,
        string $providerMessageId,
        array $metadata = []
    ): self {
        return new self(
            id: Uuid::v4()->toRfc4122(),
            emailId: $email->id,
            action: AuditAction::SENT,
            senderId: $senderId,
            recipientEmail: self::maskEmail($email->getRecipients()[0]->email),
            subject: $email->subject,
            status: EmailStatus::SENT,
            providerName: $providerName,
            providerMessageId: $providerMessageId,
            metadata: $metadata
        );
    }

    public static function forEmailFailed(
        Email $email,
        string $senderId,
        string $errorMessage,
        ?string $providerName = null,
        array $metadata = []
    ): self {
        return new self(
            id: Uuid::v4()->toRfc4122(),
            emailId: $email->id,
            action: AuditAction::FAILED,
            senderId: $senderId,
            recipientEmail: self::maskEmail($email->getRecipients()[0]->email),
            subject: $email->subject,
            status: EmailStatus::FAILED,
            errorMessage: $errorMessage,
            providerName: $providerName,
            metadata: $metadata
        );
    }

    public static function forEmailRetried(
        Email $email,
        string $senderId,
        int $attemptNumber,
        array $metadata = []
    ): self {
        return new self(
            id: Uuid::v4()->toRfc4122(),
            emailId: $email->id,
            action: AuditAction::RETRIED,
            senderId: $senderId,
            recipientEmail: self::maskEmail($email->getRecipients()[0]->email),
            subject: $email->subject,
            status: $email->getStatus(),
            metadata: array_merge($metadata, ['attempt' => $attemptNumber])
        );
    }

    public static function forEmailDeadLettered(
        Email $email,
        string $senderId,
        string $reason,
        array $metadata = []
    ): self {
        return new self(
            id: Uuid::v4()->toRfc4122(),
            emailId: $email->id,
            action: AuditAction::DEAD_LETTERED,
            senderId: $senderId,
            recipientEmail: self::maskEmail($email->getRecipients()[0]->email),
            subject: $email->subject,
            status: EmailStatus::FAILED,
            errorMessage: $reason,
            metadata: $metadata
        );
    }

    /**
     * Mask email address for PII protection
     * john.doe@example.com -> j***@example.com
     */
    public static function maskEmail(string $email): string
    {
        $parts = explode('@', $email);
        if (count($parts) !== 2) {
            return '***';
        }
        
        $local = $parts[0];
        $domain = $parts[1];
        
        if (strlen($local) <= 1) {
            return '*@' . $domain;
        }
        
        return $local[0] . str_repeat('*', min(3, strlen($local) - 1)) . '@' . $domain;
    }

    public function toArray(): array
    {
        return [
            'id' => $this->id,
            'email_id' => $this->emailId,
            'action' => $this->action->value,
            'sender_id' => $this->senderId,
            'recipient_email' => $this->recipientEmail,
            'subject' => $this->subject,
            'status' => $this->status->value,
            'error_message' => $this->errorMessage,
            'provider_name' => $this->providerName,
            'provider_message_id' => $this->providerMessageId,
            'metadata' => json_encode($this->metadata),
            'timestamp' => $this->timestamp->format('Y-m-d H:i:s.u'),
        ];
    }
}
