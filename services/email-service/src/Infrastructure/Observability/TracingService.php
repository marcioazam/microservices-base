<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Observability;

use OpenTelemetry\API\Trace\SpanInterface;
use OpenTelemetry\API\Trace\SpanKind;
use OpenTelemetry\API\Trace\StatusCode;
use OpenTelemetry\API\Trace\TracerInterface;
use OpenTelemetry\API\Trace\TracerProviderInterface;
use OpenTelemetry\Context\Context;

/**
 * OpenTelemetry tracing service for email operations.
 * Emits traces for email send operations with trace ID, span ID, timestamps, and status.
 */
final readonly class TracingService implements TracingServiceInterface
{
    private const SERVICE_NAME = 'email-service';

    public function __construct(
        private TracerProviderInterface $tracerProvider,
    ) {
    }

    public function startSpan(string $operationName, array $attributes = []): SpanInterface
    {
        $tracer = $this->getTracer();

        $spanBuilder = $tracer->spanBuilder($operationName)
            ->setSpanKind(SpanKind::KIND_INTERNAL);

        foreach ($attributes as $key => $value) {
            $spanBuilder->setAttribute($key, $value);
        }

        return $spanBuilder->startSpan();
    }

    public function startEmailSendSpan(
        string $emailId,
        string $recipientDomain,
        string $provider,
    ): SpanInterface {
        return $this->startSpan('email.send', [
            'email.id' => $emailId,
            'email.recipient_domain' => $recipientDomain,
            'email.provider' => $provider,
            'service.name' => self::SERVICE_NAME,
        ]);
    }

    public function startValidationSpan(string $email): SpanInterface
    {
        $domain = $this->extractDomain($email);

        return $this->startSpan('email.validate', [
            'email.domain' => $domain,
            'service.name' => self::SERVICE_NAME,
        ]);
    }

    public function startTemplateRenderSpan(string $templateId): SpanInterface
    {
        return $this->startSpan('email.template.render', [
            'template.id' => $templateId,
            'service.name' => self::SERVICE_NAME,
        ]);
    }

    public function startQueueSpan(string $emailId, string $queueName): SpanInterface
    {
        return $this->startSpan('email.queue', [
            'email.id' => $emailId,
            'queue.name' => $queueName,
            'service.name' => self::SERVICE_NAME,
        ]);
    }

    public function endSpanSuccess(SpanInterface $span, array $attributes = []): void
    {
        foreach ($attributes as $key => $value) {
            $span->setAttribute($key, $value);
        }

        $span->setStatus(StatusCode::STATUS_OK);
        $span->end();
    }

    public function endSpanError(SpanInterface $span, \Throwable $exception): void
    {
        $span->recordException($exception);
        $span->setStatus(StatusCode::STATUS_ERROR, $exception->getMessage());
        $span->end();
    }

    public function getTraceId(): ?string
    {
        $span = $this->getCurrentSpan();
        if ($span === null) {
            return null;
        }

        $context = $span->getContext();
        return $context->getTraceId();
    }

    public function getSpanId(): ?string
    {
        $span = $this->getCurrentSpan();
        if ($span === null) {
            return null;
        }

        $context = $span->getContext();
        return $context->getSpanId();
    }

    public function getCurrentSpan(): ?SpanInterface
    {
        $context = Context::getCurrent();
        $span = \OpenTelemetry\API\Trace\Span::fromContext($context);

        if (!$span->getContext()->isValid()) {
            return null;
        }

        return $span;
    }

    private function getTracer(): TracerInterface
    {
        return $this->tracerProvider->getTracer(
            self::SERVICE_NAME,
            '1.0.0'
        );
    }

    private function extractDomain(string $email): string
    {
        $parts = explode('@', $email);
        return $parts[1] ?? 'unknown';
    }
}
