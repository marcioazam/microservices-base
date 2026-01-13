<?php

declare(strict_types=1);

namespace EmailService\Domain\Entity;

use DateTimeImmutable;
use Symfony\Component\Uid\Uuid;

class Template
{
    /**
     * @param string[] $requiredVariables
     * @param array<string, mixed> $defaultVariables
     */
    public function __construct(
        public readonly string $id,
        public readonly string $name,
        public readonly string $subject,
        public readonly string $bodyHtml,
        public readonly ?string $bodyText = null,
        public readonly array $requiredVariables = [],
        public readonly array $defaultVariables = [],
        public readonly DateTimeImmutable $createdAt = new DateTimeImmutable(),
        public readonly DateTimeImmutable $updatedAt = new DateTimeImmutable()
    ) {
    }

    /**
     * @param string[] $requiredVariables
     * @param array<string, mixed> $defaultVariables
     */
    public static function create(
        string $name,
        string $subject,
        string $bodyHtml,
        ?string $bodyText = null,
        array $requiredVariables = [],
        array $defaultVariables = []
    ): self {
        return new self(
            id: Uuid::v4()->toRfc4122(),
            name: $name,
            subject: $subject,
            bodyHtml: $bodyHtml,
            bodyText: $bodyText,
            requiredVariables: $requiredVariables,
            defaultVariables: $defaultVariables
        );
    }

    /**
     * Extract variables from template content using Twig syntax {{ variable }}
     * @return string[]
     */
    public function extractVariables(): array
    {
        $variables = [];
        
        // Match {{ variable }} and {{ variable|filter }}
        preg_match_all('/\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*(?:\|[^}]*)?\}\}/', $this->bodyHtml, $matches);
        $variables = array_merge($variables, $matches[1]);
        
        preg_match_all('/\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*(?:\|[^}]*)?\}\}/', $this->subject, $matches);
        $variables = array_merge($variables, $matches[1]);
        
        if ($this->bodyText !== null) {
            preg_match_all('/\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*(?:\|[^}]*)?\}\}/', $this->bodyText, $matches);
            $variables = array_merge($variables, $matches[1]);
        }
        
        return array_unique($variables);
    }

    /**
     * Validate that all required variables are provided
     * @param array<string, mixed> $providedVariables
     * @return string[] Missing variable names
     */
    public function getMissingVariables(array $providedVariables): array
    {
        $missing = [];
        
        foreach ($this->requiredVariables as $required) {
            if (!array_key_exists($required, $providedVariables) 
                && !array_key_exists($required, $this->defaultVariables)) {
                $missing[] = $required;
            }
        }
        
        return $missing;
    }

    /**
     * Merge provided variables with defaults
     * @param array<string, mixed> $providedVariables
     * @return array<string, mixed>
     */
    public function mergeWithDefaults(array $providedVariables): array
    {
        return array_merge($this->defaultVariables, $providedVariables);
    }

    public function hasTextVersion(): bool
    {
        return $this->bodyText !== null && $this->bodyText !== '';
    }
}
