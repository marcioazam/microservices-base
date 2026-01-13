<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Observability;

/**
 * Interface for checking email provider health.
 */
interface ProviderHealthCheckerInterface
{
    /**
     * Check health of all configured providers.
     *
     * @return array<string, bool> Provider name => healthy status
     */
    public function checkAll(): array;

    /**
     * Check health of a specific provider.
     */
    public function check(string $providerName): bool;
}
