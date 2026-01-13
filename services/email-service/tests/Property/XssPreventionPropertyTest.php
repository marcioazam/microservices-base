<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Application\Service\TemplateService;
use EmailService\Domain\Entity\Template;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Property 17: XSS Prevention
 * For any template variable value containing HTML special characters (<, >, &, ", '),
 * the rendered output SHALL contain the escaped equivalents.
 * 
 * Validates: Requirements 4.3
 */
class XssPreventionPropertyTest extends TestCase
{
    use TestTrait;

    private TemplateService $templateService;

    protected function setUp(): void
    {
        $this->templateService = new TemplateService();
    }

    /**
     * @test
     * Property 17: HTML special characters are escaped
     */
    public function htmlSpecialCharactersAreEscaped(): void
    {
        $template = Template::create(
            name: 'xss-test',
            subject: 'Test {{ content }}',
            bodyHtml: '<p>{{ content }}</p>'
        );
        
        $this->templateService->registerTemplate($template);

        $maliciousInputs = [
            '<script>alert("xss")</script>',
            '<img src="x" onerror="alert(1)">',
            '"><script>alert(1)</script>',
            "javascript:alert('xss')",
            '<a href="javascript:alert(1)">click</a>',
            '&lt;already&gt;escaped&amp;',
        ];

        foreach ($maliciousInputs as $input) {
            $result = $this->templateService->render($template->id, ['content' => $input]);
            
            // Should not contain unescaped script tags
            $this->assertStringNotContainsString('<script>', $result->bodyHtml);
            $this->assertStringNotContainsString('</script>', $result->bodyHtml);
            
            // Should not contain unescaped event handlers
            $this->assertStringNotContainsString('onerror=', $result->bodyHtml);
        }
    }

    /**
     * @test
     * Property 17: Less than sign is escaped
     */
    public function lessThanSignIsEscaped(): void
    {
        $template = Template::create(
            name: 'lt-test',
            subject: '{{ value }}',
            bodyHtml: '<p>{{ value }}</p>'
        );
        
        $this->templateService->registerTemplate($template);

        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20,
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $prefix): void {
            $input = $prefix . '<dangerous>';
            
            $result = $this->templateService->render('lt-test', ['value' => $input]);
            
            // The < should be escaped to &lt;
            $this->assertStringContainsString('&lt;', $result->bodyHtml);
            $this->assertStringNotContainsString('<dangerous>', $result->bodyHtml);
        });
    }

    /**
     * @test
     * Property 17: Greater than sign is escaped
     */
    public function greaterThanSignIsEscaped(): void
    {
        $template = Template::create(
            name: 'gt-test',
            subject: '{{ value }}',
            bodyHtml: '<p>{{ value }}</p>'
        );
        
        $this->templateService->registerTemplate($template);

        $result = $this->templateService->render($template->id, ['value' => 'test>value']);
        
        $this->assertStringContainsString('&gt;', $result->bodyHtml);
    }

    /**
     * @test
     * Property 17: Ampersand is escaped
     */
    public function ampersandIsEscaped(): void
    {
        $template = Template::create(
            name: 'amp-test',
            subject: '{{ value }}',
            bodyHtml: '<p>{{ value }}</p>'
        );
        
        $this->templateService->registerTemplate($template);

        $result = $this->templateService->render($template->id, ['value' => 'test&value']);
        
        $this->assertStringContainsString('&amp;', $result->bodyHtml);
    }

    /**
     * @test
     * Property 17: Double quotes are escaped
     */
    public function doubleQuotesAreEscaped(): void
    {
        $template = Template::create(
            name: 'quote-test',
            subject: '{{ value }}',
            bodyHtml: '<p data-value="{{ value }}">test</p>'
        );
        
        $this->templateService->registerTemplate($template);

        $result = $this->templateService->render($template->id, ['value' => 'test"value']);
        
        $this->assertStringContainsString('&quot;', $result->bodyHtml);
    }

    /**
     * @test
     * Property 17: Combined XSS attempts are neutralized
     */
    public function combinedXssAttemptsAreNeutralized(): void
    {
        $template = Template::create(
            name: 'combined-test',
            subject: 'Hello {{ name }}',
            bodyHtml: '<div class="user">{{ name }}</div><p>{{ message }}</p>'
        );
        
        $this->templateService->registerTemplate($template);

        $result = $this->templateService->render($template->id, [
            'name' => '<script>alert("name")</script>',
            'message' => '"><img src=x onerror=alert(1)><"',
        ]);
        
        // Verify no executable code remains
        $this->assertStringNotContainsString('<script>', $result->bodyHtml);
        $this->assertStringNotContainsString('onerror=', $result->bodyHtml);
        $this->assertStringNotContainsString('<img', $result->bodyHtml);
        
        // Verify escaped versions are present
        $this->assertStringContainsString('&lt;script&gt;', $result->bodyHtml);
    }

    /**
     * @test
     * Property 17: Safe content is not double-escaped
     */
    public function safeContentIsNotDoubleEscaped(): void
    {
        $template = Template::create(
            name: 'safe-test',
            subject: '{{ value }}',
            bodyHtml: '<p>{{ value }}</p>'
        );
        
        $this->templateService->registerTemplate($template);

        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 50 && preg_match('/^[a-zA-Z0-9 ]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $safeContent): void {
            $result = $this->templateService->render('safe-test', ['value' => $safeContent]);
            
            // Safe content should appear as-is
            $this->assertStringContainsString($safeContent, $result->bodyHtml);
            
            // Should not have unnecessary escaping
            $this->assertStringNotContainsString('&amp;amp;', $result->bodyHtml);
        });
    }

    /**
     * @test
     * Property 17: Subject line is also escaped
     */
    public function subjectLineIsAlsoEscaped(): void
    {
        $template = Template::create(
            name: 'subject-xss-test',
            subject: 'Hello {{ name }}',
            bodyHtml: '<p>Body</p>'
        );
        
        $this->templateService->registerTemplate($template);

        $result = $this->templateService->render($template->id, [
            'name' => '<script>alert(1)</script>',
        ]);
        
        $this->assertStringNotContainsString('<script>', $result->subject);
        $this->assertStringContainsString('&lt;script&gt;', $result->subject);
    }
}
