<?php

declare(strict_types=1);

namespace EmailService\Application\Service;

/**
 * Interface for template rendering with caching.
 */
interface TemplateServiceInterface
{
    /**
     * Render a template with variables.
     *
     * @param array<string, mixed> $variables
     */
    public function render(string $templateId, string $templateContent, array $variables = []): string;

    /**
     * Render from a pre-cached template.
     *
     * @param array<string, mixed> $variables
     */
    public function renderFromTemplate(string $templateId, array $variables = []): string;

    /**
     * Precompile and cache a template.
     */
    public function precompile(string $templateId, string $templateContent): void;

    /**
     * Invalidate a cached template.
     */
    public function invalidate(string $templateId): bool;

    /**
     * Check if a template is cached.
     */
    public function isCached(string $templateId, string $templateContent): bool;
}
