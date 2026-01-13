<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Application\Service\TemplateService;
use EmailService\Infrastructure\Platform\CacheClientInterface;
use EmailService\Infrastructure\Platform\CacheSource;
use EmailService\Infrastructure\Platform\CacheValue;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Feature: email-service-modernization-2025
 * Property 9: Template Cache Effectiveness
 * Validates: Requirements 9.3
 *
 * For any template ID, after the first render, subsequent renders with the same
 * template ID SHALL use the cached compilation (cache hit), resulting in faster execution.
 */
final class TemplateCachePropertyTest extends TestCase
{
    use TestTrait;

    private TemplateService $templateService;
    private CacheClientInterface $cacheClient;

    protected function setUp(): void
    {
        $this->cacheClient = $this->createMock(CacheClientInterface::class);
        $this->templateService = new TemplateService($this->cacheClient);
    }

    /**
     * @test
     */
    public function firstRenderCachesTemplate(): void
    {
        $this->forAll(
            Generator\elements(['welcome', 'password-reset', 'notification']),
            Generator\elements(['Hello {{ name }}!', 'Reset: {{ link }}', 'New: {{ message }}'])
        )
            ->withMaxSize(100)
            ->then(function (string $templateId, string $content): void {
                $this->cacheClient->method('get')->willReturn(null);

                $this->cacheClient->expects($this->once())
                    ->method('set')
                    ->with(
                        $this->stringContains("compiled:{$templateId}"),
                        $this->callback(fn($v) => isset($v['source']) && isset($v['compiled'])),
                        $this->anything(),
                        'email:template'
                    )
                    ->willReturn(true);

                $this->templateService->render($templateId, $content, ['name' => 'Test']);
            });
    }

    /**
     * @test
     */
    public function subsequentRendersUseCachedTemplate(): void
    {
        $templateId = 'cached-template';
        $content = 'Hello {{ name }}!';

        $cachedValue = new CacheValue(
            ['source' => $content, 'compiled' => true],
            CacheSource::REDIS
        );

        $this->cacheClient->method('get')->willReturn($cachedValue);

        // set should NOT be called on cache hit
        $this->cacheClient->expects($this->never())->method('set');

        $result = $this->templateService->render($templateId, $content, ['name' => 'World']);

        $this->assertEquals('Hello World!', $result);
    }

    /**
     * @test
     */
    public function isCachedReturnsTrueForCachedTemplates(): void
    {
        $this->forAll(Generator\elements(['t1', 't2', 't3']))
            ->withMaxSize(100)
            ->then(function (string $templateId): void {
                $content = 'Template {{ var }}';

                $this->cacheClient->method('get')
                    ->willReturn(new CacheValue(
                        ['source' => $content, 'compiled' => true],
                        CacheSource::REDIS
                    ));

                $this->assertTrue($this->templateService->isCached($templateId, $content));
            });
    }

    /**
     * @test
     */
    public function isCachedReturnsFalseForUncachedTemplates(): void
    {
        $this->forAll(Generator\elements(['new1', 'new2', 'new3']))
            ->withMaxSize(100)
            ->then(function (string $templateId): void {
                $content = 'New template {{ var }}';

                $this->cacheClient->method('get')->willReturn(null);

                $this->assertFalse($this->templateService->isCached($templateId, $content));
            });
    }

    /**
     * @test
     */
    public function precompileStoresTemplateInCache(): void
    {
        $templateId = 'precompiled';
        $content = 'Precompiled {{ value }}';

        $this->cacheClient->expects($this->once())
            ->method('set')
            ->with(
                $this->stringContains("compiled:{$templateId}"),
                $this->callback(fn($v) => $v['source'] === $content && $v['compiled'] === true),
                3600,
                'email:template'
            )
            ->willReturn(true);

        $this->templateService->precompile($templateId, $content);
    }

    /**
     * @test
     */
    public function invalidateRemovesTemplateFromCache(): void
    {
        $templateId = 'to-invalidate';

        $this->cacheClient->expects($this->once())
            ->method('delete')
            ->with("compiled:{$templateId}", 'email:template')
            ->willReturn(true);

        $result = $this->templateService->invalidate($templateId);

        $this->assertTrue($result);
    }

    /**
     * @test
     */
    public function differentContentGeneratesDifferentCacheKeys(): void
    {
        $templateId = 'same-id';
        $content1 = 'Version 1: {{ var }}';
        $content2 = 'Version 2: {{ var }}';

        $cacheKeys = [];

        $this->cacheClient->method('get')->willReturn(null);
        $this->cacheClient->method('set')
            ->willReturnCallback(function ($key) use (&$cacheKeys) {
                $cacheKeys[] = $key;
                return true;
            });

        $this->templateService->render($templateId, $content1, ['var' => 'test']);
        $this->templateService->render($templateId, $content2, ['var' => 'test']);

        $this->assertCount(2, $cacheKeys);
        $this->assertNotEquals($cacheKeys[0], $cacheKeys[1]);
    }

    /**
     * @test
     */
    public function templateRenderingProducesCorrectOutput(): void
    {
        $this->forAll(
            Generator\elements(['Alice', 'Bob', 'Charlie']),
            Generator\elements(['Welcome', 'Hello', 'Hi'])
        )
            ->withMaxSize(100)
            ->then(function (string $name, string $greeting): void {
                $content = "{{ greeting }}, {{ name }}!";

                $this->cacheClient->method('get')->willReturn(null);
                $this->cacheClient->method('set')->willReturn(true);

                $result = $this->templateService->render(
                    'greeting',
                    $content,
                    ['greeting' => $greeting, 'name' => $name]
                );

                $this->assertEquals("{$greeting}, {$name}!", $result);
            });
    }

    /**
     * @test
     */
    public function invalidTemplateSyntaxThrowsException(): void
    {
        $this->expectException(\InvalidArgumentException::class);
        $this->expectExceptionMessage('Invalid template syntax');

        $this->templateService->precompile('invalid', '{{ unclosed');
    }
}
