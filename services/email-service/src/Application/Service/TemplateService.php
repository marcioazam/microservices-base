<?php

declare(strict_types=1);

namespace EmailService\Application\Service;

use EmailService\Infrastructure\Platform\CacheClientInterface;
use Twig\Environment;
use Twig\Loader\ArrayLoader;

/**
 * Template service with compilation caching via CacheClient.
 */
final readonly class TemplateService implements TemplateServiceInterface
{
    private const CACHE_NAMESPACE = 'email:template';
    private const CACHE_TTL = 3600; // 1 hour

    public function __construct(
        private CacheClientInterface $cacheClient,
        private ?Environment $twig = null,
    ) {
    }

    public function render(string $templateId, string $templateContent, array $variables = []): string
    {
        $cacheKey = $this->getCacheKey($templateId, $templateContent);

        // Check for cached compiled template
        $cached = $this->cacheClient->get($cacheKey, self::CACHE_NAMESPACE);
        if ($cached !== null) {
            return $this->renderFromCache($cached->value, $variables);
        }

        // Compile and render
        $rendered = $this->compileAndRender($templateId, $templateContent, $variables);

        // Cache the template source for future renders
        $this->cacheClient->set(
            $cacheKey,
            ['source' => $templateContent, 'compiled' => true],
            self::CACHE_TTL,
            self::CACHE_NAMESPACE
        );

        return $rendered;
    }

    public function renderFromTemplate(string $templateId, array $variables = []): string
    {
        $cacheKey = "compiled:{$templateId}";

        $cached = $this->cacheClient->get($cacheKey, self::CACHE_NAMESPACE);
        if ($cached === null) {
            throw new \RuntimeException("Template not found: {$templateId}");
        }

        return $this->renderFromCache($cached->value, $variables);
    }

    public function precompile(string $templateId, string $templateContent): void
    {
        $cacheKey = $this->getCacheKey($templateId, $templateContent);

        // Validate template syntax
        $this->validateTemplate($templateContent);

        // Store in cache
        $this->cacheClient->set(
            $cacheKey,
            ['source' => $templateContent, 'compiled' => true],
            self::CACHE_TTL,
            self::CACHE_NAMESPACE
        );
    }

    public function invalidate(string $templateId): bool
    {
        $cacheKey = "compiled:{$templateId}";
        return $this->cacheClient->delete($cacheKey, self::CACHE_NAMESPACE);
    }

    public function isCached(string $templateId, string $templateContent): bool
    {
        $cacheKey = $this->getCacheKey($templateId, $templateContent);
        return $this->cacheClient->get($cacheKey, self::CACHE_NAMESPACE) !== null;
    }

    private function getCacheKey(string $templateId, string $templateContent): string
    {
        $contentHash = hash('xxh3', $templateContent);
        return "compiled:{$templateId}:{$contentHash}";
    }

    /**
     * @param array{source: string, compiled: bool} $cachedData
     * @param array<string, mixed> $variables
     */
    private function renderFromCache(array $cachedData, array $variables): string
    {
        return $this->compileAndRender('cached', $cachedData['source'], $variables);
    }

    /**
     * @param array<string, mixed> $variables
     */
    private function compileAndRender(string $name, string $source, array $variables): string
    {
        $twig = $this->getTwigEnvironment($name, $source);
        return $twig->render($name, $variables);
    }

    private function validateTemplate(string $templateContent): void
    {
        try {
            $twig = $this->getTwigEnvironment('validation', $templateContent);
            $twig->parse($twig->tokenize(new \Twig\Source($templateContent, 'validation')));
        } catch (\Twig\Error\SyntaxError $e) {
            throw new \InvalidArgumentException("Invalid template syntax: {$e->getMessage()}", 0, $e);
        }
    }

    private function getTwigEnvironment(string $name, string $source): Environment
    {
        if ($this->twig !== null) {
            return $this->twig;
        }

        $loader = new ArrayLoader([$name => $source]);
        return new Environment($loader, [
            'cache' => false,
            'auto_reload' => false,
            'strict_variables' => true,
        ]);
    }
}
