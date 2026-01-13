<?php

declare(strict_types=1);

namespace EmailService\Application\Service;

use RuntimeException;

class TemplateNotFoundException extends RuntimeException
{
    public function __construct(
        public readonly string $templateId
    ) {
        parent::__construct("Template not found: {$templateId}");
    }
}
