<?php

declare(strict_types=1);

namespace EmailService\Application\Service;

use EmailService\Application\DTO\BatchEmailRequest;
use EmailService\Application\DTO\BatchEmailResult;

/**
 * Interface for batch email processing.
 */
interface BatchEmailServiceInterface
{
    /**
     * Process a batch of emails.
     */
    public function processBatch(BatchEmailRequest $request): BatchEmailResult;

    /**
     * Get maximum allowed batch size.
     */
    public function getMaxBatchSize(): int;
}
