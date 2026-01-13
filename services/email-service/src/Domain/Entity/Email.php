<?php

declare(strict_types=1);

namespace EmailService\Domain\Entity;

use DateTimeImmutable;
use EmailService\Domain\Enum\ContentType;
use EmailService\Domain\Enum\EmailStatus;
use EmailService\Domain\Enum\EmailType;
use EmailService\Domain\ValueObject\Attachment;
use EmailService\Domain\ValueObject\Recipient;
use Symfony\Component\Uid\Uuid;

class Email
{
    /** @var Recipient[] */
    private array $recipients;
    /** @var Recipient[] */
    private array $cc;
    /** @var Recipient[] */
    private array $bcc;
    /** @var Attachment[] */
    private array $attachments;
    private EmailStatus $status;
    private ?DateTimeImmutable $sentAt;
    private ?string $providerMessageId;
    private ?string $errorMessage;
    private int $attempts;

    /**
     * @param Recipient[] $recipients
     * @param Recipient[] $cc
     * @param Recipient[] $bcc
     * @param Attachment[] $attachments
     * @param array<string, mixed> $metadata
     * @param array<string, string> $headers
     */
    public function __construct(
        public readonly string $id,
        array $recipients,
        public readonly Recipient $from,
        public readonly string $subject,
        public readonly string $body,
        public readonly ContentType $contentType = ContentType::HTML,
        array $cc = [],
        array $bcc = [],
        array $attachments = [],
        public readonly array $headers = [],
        public readonly array $metadata = [],
        public readonly EmailType $type = EmailType::TRANSACTIONAL,
        public readonly DateTimeImmutable $createdAt = new DateTimeImmutable(),
        public readonly ?string $templateId = null
    ) {
        $this->setRecipients($recipients);
        $this->cc = $cc;
        $this->bcc = $bcc;
        $this->setAttachments($attachments);
        $this->status = EmailStatus::PENDING;
        $this->sentAt = null;
        $this->providerMessageId = null;
        $this->errorMessage = null;
        $this->attempts = 0;
    }

    /**
     * @param Recipient[] $recipients
     */
    private function setRecipients(array $recipients): void
    {
        if (empty($recipients)) {
            throw new \InvalidArgumentException('At least one recipient is required');
        }
        $this->recipients = $recipients;
    }

    /**
     * @param Attachment[] $attachments
     */
    private function setAttachments(array $attachments): void
    {
        if (!empty($attachments)) {
            Attachment::validateTotalSize($attachments);
        }
        $this->attachments = $attachments;
    }

    public static function create(
        Recipient $from,
        array $recipients,
        string $subject,
        string $body,
        ContentType $contentType = ContentType::HTML,
        array $cc = [],
        array $bcc = [],
        array $attachments = [],
        array $headers = [],
        array $metadata = [],
        EmailType $type = EmailType::TRANSACTIONAL,
        ?string $templateId = null
    ): self {
        return new self(
            id: Uuid::v4()->toRfc4122(),
            recipients: $recipients,
            from: $from,
            subject: $subject,
            body: $body,
            contentType: $contentType,
            cc: $cc,
            bcc: $bcc,
            attachments: $attachments,
            headers: $headers,
            metadata: $metadata,
            type: $type,
            templateId: $templateId
        );
    }

    /** @return Recipient[] */
    public function getRecipients(): array
    {
        return $this->recipients;
    }

    /** @return Recipient[] */
    public function getCc(): array
    {
        return $this->cc;
    }

    /** @return Recipient[] */
    public function getBcc(): array
    {
        return $this->bcc;
    }

    /** @return Attachment[] */
    public function getAttachments(): array
    {
        return $this->attachments;
    }

    public function getStatus(): EmailStatus
    {
        return $this->status;
    }

    public function getSentAt(): ?DateTimeImmutable
    {
        return $this->sentAt;
    }

    public function getProviderMessageId(): ?string
    {
        return $this->providerMessageId;
    }

    public function getErrorMessage(): ?string
    {
        return $this->errorMessage;
    }

    public function getAttempts(): int
    {
        return $this->attempts;
    }

    public function markAsQueued(): void
    {
        $this->status = EmailStatus::QUEUED;
    }

    public function markAsSending(): void
    {
        $this->status = EmailStatus::SENDING;
        $this->attempts++;
    }

    public function markAsSent(string $providerMessageId): void
    {
        $this->status = EmailStatus::SENT;
        $this->sentAt = new DateTimeImmutable();
        $this->providerMessageId = $providerMessageId;
        $this->errorMessage = null;
    }

    public function markAsDelivered(): void
    {
        $this->status = EmailStatus::DELIVERED;
    }

    public function markAsFailed(string $errorMessage): void
    {
        $this->status = EmailStatus::FAILED;
        $this->errorMessage = $errorMessage;
    }

    public function markAsBounced(): void
    {
        $this->status = EmailStatus::BOUNCED;
    }

    public function markAsRejected(): void
    {
        $this->status = EmailStatus::REJECTED;
    }

    public function canRetry(): bool
    {
        return $this->status->canRetry();
    }

    public function hasAttachments(): bool
    {
        return !empty($this->attachments);
    }

    public function getTotalAttachmentSize(): int
    {
        return array_reduce(
            $this->attachments,
            fn(int $carry, Attachment $a) => $carry + $a->size,
            0
        );
    }

    public function getAllRecipientEmails(): array
    {
        $emails = array_map(fn(Recipient $r) => $r->email, $this->recipients);
        $emails = array_merge($emails, array_map(fn(Recipient $r) => $r->email, $this->cc));
        $emails = array_merge($emails, array_map(fn(Recipient $r) => $r->email, $this->bcc));
        return array_unique($emails);
    }
}
