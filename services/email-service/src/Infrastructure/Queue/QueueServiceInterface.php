<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Queue;

interface QueueServiceInterface
{
    /**
     * Add a job to the queue
     */
    public function enqueue(EmailJob $job): string;

    /**
     * Process a job
     */
    public function process(EmailJob $job): ProcessResult;

    /**
     * Schedule a job for retry
     */
    public function retry(string $jobId): void;

    /**
     * Move a job to dead letter queue
     */
    public function moveToDeadLetter(string $jobId, string $reason): void;

    /**
     * Get current queue depth
     */
    public function getQueueDepth(): int;

    /**
     * Get next job from queue
     */
    public function dequeue(): ?EmailJob;
}
