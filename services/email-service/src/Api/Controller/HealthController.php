<?php

declare(strict_types=1);

namespace EmailService\Api\Controller;

use EmailService\Infrastructure\Observability\HealthCheck;
use EmailService\Infrastructure\Observability\EmailMetrics;

class HealthController
{
    public function __construct(
        private readonly HealthCheck $healthCheck,
        private readonly EmailMetrics $metrics
    ) {
    }

    public function liveness(): array
    {
        return [
            'status' => 'ok',
            'timestamp' => gmdate('Y-m-d\TH:i:s\Z'),
        ];
    }

    public function readiness(): array
    {
        $result = $this->healthCheck->check();
        
        return [
            'statusCode' => $result->getHttpStatusCode(),
            'body' => $result->toArray(),
        ];
    }

    public function metrics(): string
    {
        return $this->metrics->getPrometheusOutput();
    }
}
