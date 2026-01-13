<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Observability;

use OpenTelemetry\API\Trace\SpanInterface;

/**
 * Interface for OpenTelemetry tracing operations.
 */
interface TracingServiceInterface
{
    /**
     * Start a new span with the given operation name.
     *
     * @param array<string, mixed> $attributes
     */
    public function startSpan(string $operationName, array $attributes = []): SpanInterface;

    /**
     * Start a span for email send operation.
     */
    public function startEmailSendSpan(
        string $emailId,
        string $recipientDomain,
        string $provider,
    ): SpanInterface;

    /**
     * Start a span for email validation.
     */
    public function startValidationSpan(string $email): SpanInterface;

    /**
     * Start a span for template rendering.
     */
    public function startTemplateRenderSpan(string $templateId): SpanInterface;

    /**
     * Start a span for queue operations.
     */
    public function startQueueSpan(string $emailId, string $queueName): SpanInterface;

    /**
     * End span with success status.
     *
     * @param array<string, mixed> $attributes
     */
    public function endSpanSuccess(SpanInterface $span, array $attributes = []): void;

    /**
     * End span with error status.
     */
    public function endSpanError(SpanInterface $span, \Throwable $exception): void;

    /**
     * Get current trace ID.
     */
    public function getTraceId(): ?string;

    /**
     * Get current span ID.
     */
    public function getSpanId(): ?string;

    /**
     * Get current active span.
     */
    public function getCurrentSpan(): ?SpanInterface;
}
