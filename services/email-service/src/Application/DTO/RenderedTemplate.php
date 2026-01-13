<?php

declare(strict_types=1);

namespace EmailService\Application\DTO;

final readonly class RenderedTemplate
{
    public function __construct(
        public string $subject,
        public string $bodyHtml,
        public ?string $bodyText = null
    ) {
    }
}
