<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Application\Service\TemplateNotFoundException;
use EmailService\Application\Service\TemplateService;
use EmailService\Domain\Entity\Template;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Property 15: Template Variable Substitution
 * For any template with variables V and provided values M, the rendered output
 * SHALL contain M[v] for each v in V where M[v] is defined.
 * 
 * Property 16: Default Variable Handling
 * For any template variable v not provided in the request, the Template_Engine
 * SHALL substitute with defaultVariables[v] if defined, otherwise empty string.
 * 
 * Validates: Requirements 4.1, 4.2
 */
class TemplateRenderingPropertyTest extends TestCase
{
    use TestTrait;

    private TemplateService $templateService;

    protected function setUp(): void
    {
        $this->templateService = new TemplateService();
    }

    /**
     * @test
     * Property 15: Variables are correctly substituted in rendered output
     */
    public function variablesAreCorrectlySubstituted(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 50 && preg_match('/^[a-zA-Z0-9 ]+$/', $s),
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 50 && preg_match('/^[a-zA-Z0-9 ]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $name, string $company): void {
            $template = Template::create(
                name: 'test-template',
                subject: 'Hello {{ name }}',
                bodyHtml: '<p>Welcome {{ name }} from {{ company }}!</p>',
                requiredVariables: ['name', 'company']
            );
            
            $this->templateService->registerTemplate($template);
            
            $result = $this->templateService->render($template->id, [
                'name' => $name,
                'company' => $company,
            ]);
            
            $this->assertStringContainsString($name, $result->subject);
            $this->assertStringContainsString($name, $result->bodyHtml);
            $this->assertStringContainsString($company, $result->bodyHtml);
        });
    }

    /**
     * @test
     * Property 16: Default values are used for missing variables
     */
    public function defaultValuesAreUsedForMissingVariables(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 30 && preg_match('/^[a-zA-Z]+$/', $s),
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 30 && preg_match('/^[a-zA-Z]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $providedName, string $defaultGreeting): void {
            $template = Template::create(
                name: 'greeting-template',
                subject: '{{ greeting }} {{ name }}',
                bodyHtml: '<p>{{ greeting }}, {{ name }}!</p>',
                requiredVariables: ['name'],
                defaultVariables: ['greeting' => $defaultGreeting]
            );
            
            $this->templateService->registerTemplate($template);
            
            // Only provide name, not greeting
            $result = $this->templateService->render($template->id, [
                'name' => $providedName,
            ]);
            
            $this->assertStringContainsString($providedName, $result->subject);
            $this->assertStringContainsString($defaultGreeting, $result->subject);
            $this->assertStringContainsString($providedName, $result->bodyHtml);
            $this->assertStringContainsString($defaultGreeting, $result->bodyHtml);
        });
    }

    /**
     * @test
     * Property 16: Undefined variables without defaults become empty strings
     */
    public function undefinedVariablesWithoutDefaultsBecomeEmpty(): void
    {
        $template = Template::create(
            name: 'optional-template',
            subject: 'Hello {{ name }}{{ suffix }}',
            bodyHtml: '<p>Hello {{ name }}{{ suffix }}</p>'
        );
        
        $this->templateService->registerTemplate($template);
        
        $result = $this->templateService->render($template->id, [
            'name' => 'John',
            // suffix not provided and no default
        ]);
        
        $this->assertStringContainsString('John', $result->subject);
        $this->assertStringNotContainsString('{{ suffix }}', $result->subject);
    }

    /**
     * @test
     * Property: Provided values override defaults
     */
    public function providedValuesOverrideDefaults(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 30 && preg_match('/^[a-zA-Z]+$/', $s),
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 30 && preg_match('/^[a-zA-Z]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $defaultValue, string $providedValue): void {
            if ($defaultValue === $providedValue) {
                $this->markTestSkipped('Values are the same');
                return;
            }
            
            $template = Template::create(
                name: 'override-template',
                subject: '{{ greeting }}',
                bodyHtml: '<p>{{ greeting }}</p>',
                defaultVariables: ['greeting' => $defaultValue]
            );
            
            $this->templateService->registerTemplate($template);
            
            $result = $this->templateService->render($template->id, [
                'greeting' => $providedValue,
            ]);
            
            $this->assertStringContainsString($providedValue, $result->subject);
            $this->assertStringNotContainsString($defaultValue, $result->subject);
        });
    }

    /**
     * @test
     * Property: Text body is also rendered when available
     */
    public function textBodyIsRenderedWhenAvailable(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 30 && preg_match('/^[a-zA-Z]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $name): void {
            $template = Template::create(
                name: 'multipart-template',
                subject: 'Hello {{ name }}',
                bodyHtml: '<p>Hello {{ name }}!</p>',
                bodyText: 'Hello {{ name }}!'
            );
            
            $this->templateService->registerTemplate($template);
            
            $result = $this->templateService->render($template->id, ['name' => $name]);
            
            $this->assertNotNull($result->bodyText);
            $this->assertStringContainsString($name, $result->bodyText);
        });
    }

    /**
     * @test
     * Property: Missing template throws TemplateNotFoundException
     */
    public function missingTemplateThrowsException(): void
    {
        $this->expectException(TemplateNotFoundException::class);
        
        $this->templateService->render('non-existent-template', []);
    }

    /**
     * @test
     * Property: validateVariables returns missing required variables
     */
    public function validateVariablesReturnsMissingRequired(): void
    {
        $template = Template::create(
            name: 'validation-template',
            subject: '{{ a }} {{ b }} {{ c }}',
            bodyHtml: '<p>{{ a }} {{ b }} {{ c }}</p>',
            requiredVariables: ['a', 'b', 'c'],
            defaultVariables: ['c' => 'default_c']
        );
        
        $this->templateService->registerTemplate($template);
        
        // Only provide 'a', missing 'b' (c has default)
        $missing = $this->templateService->validateVariables($template->id, ['a' => 'value_a']);
        
        $this->assertContains('b', $missing);
        $this->assertNotContains('a', $missing);
        $this->assertNotContains('c', $missing); // Has default
    }
}
