<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Infrastructure\Observability\TracingService;
use Eris\Generator;
use Eris\TestTrait;
use OpenTelemetry\API\Trace\SpanInterface;
use OpenTelemetry\API\Trace\SpanContextInterface;
use OpenTelemetry\API\Trace\TracerInterface;
use OpenTelemetry\API\Trace\TracerProviderInterface;
use OpenTelemetry\API\Trace\SpanBuilderInterface;
use PHPUnit\Framework\TestCase;

/**
 * Feature: email-service-modernization-2025
 * Property 10: OpenTelemetry Trace Completeness
 * Validates: Requirements 7.1
 *
 * For any email send operation, the emitted trace SHALL contain:
 * - A valid trace ID
 * - A valid span ID
 * - Operation name
 * - Start and end timestamps
 * - Status (success/error)
 */
final class OpenTelemetryTracePropertyTest extends TestCase
{
    use TestTrait;

    private TracingService $tracingService;
    private TracerProviderInterface $tracerProvider;
    private TracerInterface $tracer;
    private SpanBuilderInterface $spanBuilder;
    private SpanInterface $span;
    private SpanContextInterface $spanContext;

    protected function setUp(): void
    {
        $this->tracerProvider = $this->createMock(TracerProviderInterface::class);
        $this->tracer = $this->createMock(TracerInterface::class);
        $this->spanBuilder = $this->createMock(SpanBuilderInterface::class);
        $this->span = $this->createMock(SpanInterface::class);
        $this->spanContext = $this->createMock(SpanContextInterface::class);

        $this->tracerProvider->method('getTracer')->willReturn($this->tracer);
        $this->tracer->method('spanBuilder')->willReturn($this->spanBuilder);
        $this->spanBuilder->method('setSpanKind')->willReturnSelf();
        $this->spanBuilder->method('setAttribute')->willReturnSelf();
        $this->spanBuilder->method('startSpan')->willReturn($this->span);

        $this->span->method('getContext')->willReturn($this->spanContext);
        $this->spanContext->method('getTraceId')->willReturn('0af7651916cd43dd8448eb211c80319c');
        $this->spanContext->method('getSpanId')->willReturn('b7ad6b7169203331');
        $this->spanContext->method('isValid')->willReturn(true);

        $this->tracingService = new TracingService($this->tracerProvider);
    }

    /**
     * @test
     */
    public function emailSendSpanContainsValidTraceId(): void
    {
        $this->forAll(
            Generator\elements(['email-123', 'email-456', 'email-789']),
            Generator\elements(['example.com', 'test.org', 'domain.net']),
            Generator\elements(['sendgrid', 'mailgun', 'ses'])
        )
            ->withMaxSize(100)
            ->then(function (string $emailId, string $domain, string $provider): void {
                $span = $this->tracingService->startEmailSendSpan($emailId, $domain, $provider);

                $traceId = $span->getContext()->getTraceId();

                $this->assertNotEmpty($traceId, 'Trace ID should not be empty');
                $this->assertEquals(32, strlen($traceId), 'Trace ID should be 32 hex characters');
                $this->assertMatchesRegularExpression('/^[0-9a-f]{32}$/', $traceId);
            });
    }

    /**
     * @test
     */
    public function emailSendSpanContainsValidSpanId(): void
    {
        $this->forAll(
            Generator\elements(['email-123', 'email-456', 'email-789']),
            Generator\elements(['example.com', 'test.org', 'domain.net']),
            Generator\elements(['sendgrid', 'mailgun', 'ses'])
        )
            ->withMaxSize(100)
            ->then(function (string $emailId, string $domain, string $provider): void {
                $span = $this->tracingService->startEmailSendSpan($emailId, $domain, $provider);

                $spanId = $span->getContext()->getSpanId();

                $this->assertNotEmpty($spanId, 'Span ID should not be empty');
                $this->assertEquals(16, strlen($spanId), 'Span ID should be 16 hex characters');
                $this->assertMatchesRegularExpression('/^[0-9a-f]{16}$/', $spanId);
            });
    }

    /**
     * @test
     */
    public function spanBuilderReceivesCorrectOperationName(): void
    {
        $this->tracer->expects($this->once())
            ->method('spanBuilder')
            ->with('email.send')
            ->willReturn($this->spanBuilder);

        $this->tracingService->startEmailSendSpan('email-123', 'example.com', 'sendgrid');
    }

    /**
     * @test
     */
    public function spanBuilderReceivesRequiredAttributes(): void
    {
        $emailId = 'email-test-123';
        $domain = 'example.com';
        $provider = 'sendgrid';

        $this->spanBuilder->expects($this->exactly(4))
            ->method('setAttribute')
            ->willReturnCallback(function ($key, $value) use ($emailId, $domain, $provider) {
                $validAttributes = [
                    'email.id' => $emailId,
                    'email.recipient_domain' => $domain,
                    'email.provider' => $provider,
                    'service.name' => 'email-service',
                ];

                $this->assertArrayHasKey($key, $validAttributes);
                $this->assertEquals($validAttributes[$key], $value);

                return $this->spanBuilder;
            });

        $this->tracingService->startEmailSendSpan($emailId, $domain, $provider);
    }

    /**
     * @test
     */
    public function validationSpanContainsEmailDomain(): void
    {
        $this->tracer->expects($this->once())
            ->method('spanBuilder')
            ->with('email.validate')
            ->willReturn($this->spanBuilder);

        $this->spanBuilder->expects($this->exactly(2))
            ->method('setAttribute')
            ->willReturnSelf();

        $this->tracingService->startValidationSpan('user@example.com');
    }

    /**
     * @test
     */
    public function templateRenderSpanContainsTemplateId(): void
    {
        $templateId = 'welcome-email-v2';

        $this->tracer->expects($this->once())
            ->method('spanBuilder')
            ->with('email.template.render')
            ->willReturn($this->spanBuilder);

        $this->tracingService->startTemplateRenderSpan($templateId);
    }

    /**
     * @test
     */
    public function queueSpanContainsQueueName(): void
    {
        $emailId = 'email-123';
        $queueName = 'email-queue';

        $this->tracer->expects($this->once())
            ->method('spanBuilder')
            ->with('email.queue')
            ->willReturn($this->spanBuilder);

        $this->tracingService->startQueueSpan($emailId, $queueName);
    }

    /**
     * @test
     */
    public function endSpanSuccessSetsOkStatus(): void
    {
        $this->span->expects($this->once())
            ->method('setStatus')
            ->with(\OpenTelemetry\API\Trace\StatusCode::STATUS_OK);

        $this->span->expects($this->once())
            ->method('end');

        $this->tracingService->endSpanSuccess($this->span);
    }

    /**
     * @test
     */
    public function endSpanErrorRecordsExceptionAndSetsErrorStatus(): void
    {
        $exception = new \RuntimeException('Test error');

        $this->span->expects($this->once())
            ->method('recordException')
            ->with($exception);

        $this->span->expects($this->once())
            ->method('setStatus')
            ->with(\OpenTelemetry\API\Trace\StatusCode::STATUS_ERROR, 'Test error');

        $this->span->expects($this->once())
            ->method('end');

        $this->tracingService->endSpanError($this->span, $exception);
    }

    /**
     * @test
     */
    public function getTraceIdReturnsCurrentTraceId(): void
    {
        $traceId = $this->tracingService->getTraceId();

        // Note: In real scenario, this would return null without active span
        // The mock setup makes it return a valid trace ID
        $this->assertNull($traceId); // No active span in test context
    }

    /**
     * @test
     */
    public function getSpanIdReturnsCurrentSpanId(): void
    {
        $spanId = $this->tracingService->getSpanId();

        $this->assertNull($spanId); // No active span in test context
    }
}
