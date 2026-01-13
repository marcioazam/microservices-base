<?php

declare(strict_types=1);

namespace EmailService\Api\Request;

use EmailService\Domain\Enum\ContentType;
use EmailService\Domain\Enum\EmailType;

final readonly class SendEmailRequest
{
    /**
     * @param string[] $recipients
     * @param string[] $cc
     * @param string[] $bcc
     * @param array<string, string> $attachments
     * @param array<string, mixed> $metadata
     * @param array<string, mixed> $templateVariables
     */
    public function __construct(
        public string $from,
        public array $recipients,
        public string $subject,
        public string $body,
        public string $contentType = 'text/html',
        public array $cc = [],
        public array $bcc = [],
        public array $attachments = [],
        public array $metadata = [],
        public string $type = 'transactional',
        public ?string $templateId = null,
        public array $templateVariables = [],
        public ?string $fromName = null
    ) {
    }

    public static function fromArray(array $data): self
    {
        return new self(
            from: $data['from'] ?? '',
            recipients: $data['recipients'] ?? [],
            subject: $data['subject'] ?? '',
            body: $data['body'] ?? '',
            contentType: $data['content_type'] ?? 'text/html',
            cc: $data['cc'] ?? [],
            bcc: $data['bcc'] ?? [],
            attachments: $data['attachments'] ?? [],
            metadata: $data['metadata'] ?? [],
            type: $data['type'] ?? 'transactional',
            templateId: $data['template_id'] ?? null,
            templateVariables: $data['template_variables'] ?? [],
            fromName: $data['from_name'] ?? null
        );
    }

    public function getContentType(): ContentType
    {
        return $this->contentType === 'text/plain' ? ContentType::PLAIN : ContentType::HTML;
    }

    public function getEmailType(): EmailType
    {
        return match ($this->type) {
            'marketing' => EmailType::MARKETING,
            'verification' => EmailType::VERIFICATION,
            default => EmailType::TRANSACTIONAL,
        };
    }

    /**
     * @return array<string, string>
     */
    public function validate(): array
    {
        $errors = [];

        if (empty($this->from)) {
            $errors['from'] = 'From address is required';
        } elseif (!filter_var($this->from, FILTER_VALIDATE_EMAIL)) {
            $errors['from'] = 'Invalid from email address';
        }

        if (empty($this->recipients)) {
            $errors['recipients'] = 'At least one recipient is required';
        } else {
            foreach ($this->recipients as $i => $recipient) {
                if (!filter_var($recipient, FILTER_VALIDATE_EMAIL)) {
                    $errors["recipients.{$i}"] = "Invalid recipient email: {$recipient}";
                }
            }
        }

        if (empty($this->subject)) {
            $errors['subject'] = 'Subject is required';
        }

        if (empty($this->body) && $this->templateId === null) {
            $errors['body'] = 'Body is required when no template is specified';
        }

        return $errors;
    }
}
