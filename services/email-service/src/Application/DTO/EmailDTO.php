<?php

declare(strict_types=1);

namespace EmailService\Application\DTO;

use EmailService\Domain\Enum\ContentType;
use EmailService\Domain\Enum\EmailType;

final readonly class EmailDTO
{
    /**
     * @param string[] $recipients
     * @param string[] $cc
     * @param string[] $bcc
     * @param array<string, string> $attachments filename => base64 content
     * @param array<string, mixed> $metadata
     * @param array<string, mixed> $templateVariables
     */
    public function __construct(
        public string $from,
        public array $recipients,
        public string $subject,
        public string $body,
        public ContentType $contentType = ContentType::HTML,
        public array $cc = [],
        public array $bcc = [],
        public array $attachments = [],
        public array $metadata = [],
        public EmailType $type = EmailType::TRANSACTIONAL,
        public ?string $templateId = null,
        public array $templateVariables = [],
        public ?string $fromName = null
    ) {
    }
}
