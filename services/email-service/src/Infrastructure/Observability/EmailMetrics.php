<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Observability;

class EmailMetrics
{
    public function __construct(
        private readonly MetricsCollector $collector
    ) {
    }

    public function recordEmailSent(string $provider, string $type): void
    {
        $this->collector->incrementCounter('email_sent_total', [
            'provider' => $provider,
            'type' => $type,
        ]);
    }

    public function recordEmailFailed(string $provider, string $type, string $errorCode): void
    {
        $this->collector->incrementCounter('email_failed_total', [
            'provider' => $provider,
            'type' => $type,
            'error_code' => $errorCode,
        ]);
    }

    public function recordEmailQueued(): void
    {
        $this->collector->incrementCounter('email_queued_total');
    }

    public function recordEmailDelivered(string $provider): void
    {
        $this->collector->incrementCounter('email_delivered_total', [
            'provider' => $provider,
        ]);
    }

    public function recordEmailBounced(string $provider): void
    {
        $this->collector->incrementCounter('email_bounced_total', [
            'provider' => $provider,
        ]);
    }

    public function setQueueDepth(int $depth): void
    {
        $this->collector->setGauge('email_queue_depth', (float) $depth);
    }

    public function recordProcessingTime(float $seconds, string $provider): void
    {
        $this->collector->observeHistogram('email_processing_seconds', $seconds, [
            'provider' => $provider,
        ]);
    }

    public function recordValidationTime(float $seconds): void
    {
        $this->collector->observeHistogram('email_validation_seconds', $seconds);
    }

    public function recordRateLimitHit(string $senderId): void
    {
        $this->collector->incrementCounter('email_rate_limit_hit_total', [
            'sender_id' => $senderId,
        ]);
    }

    public function getPrometheusOutput(): string
    {
        return $this->collector->toPrometheusFormat();
    }
}
