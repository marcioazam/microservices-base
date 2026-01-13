<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Queue;

use EmailService\Infrastructure\Provider\ProviderRouter;

class InMemoryQueueService implements QueueServiceInterface
{
    /** @var array<int, EmailJob[]> Priority-keyed queues */
    private array $queues = [];
    
    /** @var array<string, EmailJob> */
    private array $jobs = [];
    
    /** @var array<string, EmailJob> */
    private array $deadLetterQueue = [];
    
    /** @var array<string, EmailJob> */
    private array $retryQueue = [];

    public function __construct(
        private readonly ?ProviderRouter $providerRouter = null
    ) {
    }

    public function enqueue(EmailJob $job): string
    {
        $this->jobs[$job->id] = $job;
        
        if (!isset($this->queues[$job->priority])) {
            $this->queues[$job->priority] = [];
        }
        
        $this->queues[$job->priority][] = $job;
        
        // Sort by priority (higher first)
        krsort($this->queues);
        
        return $job->id;
    }

    public function process(EmailJob $job): ProcessResult
    {
        $job->incrementAttempts();
        
        if ($this->providerRouter === null) {
            return ProcessResult::failure('No provider configured', true);
        }
        
        try {
            $result = $this->providerRouter->send($job->email);
            
            if ($result->success) {
                $job->email->markAsSent($result->messageId);
                unset($this->jobs[$job->id]);
                return ProcessResult::success($result->messageId);
            }
            
            $job->email->markAsFailed($result->errorMessage ?? 'Unknown error');
            
            if ($job->canRetry()) {
                return ProcessResult::failure($result->errorMessage ?? 'Unknown error', true);
            }
            
            return ProcessResult::permanentFailure($result->errorMessage ?? 'Max retries exceeded');
        } catch (\Exception $e) {
            $job->email->markAsFailed($e->getMessage());
            
            if ($job->canRetry()) {
                return ProcessResult::failure($e->getMessage(), true);
            }
            
            return ProcessResult::permanentFailure($e->getMessage());
        }
    }

    public function retry(string $jobId): void
    {
        if (!isset($this->jobs[$jobId])) {
            return;
        }
        
        $job = $this->jobs[$jobId];
        $job->scheduleRetry();
        $this->retryQueue[$jobId] = $job;
    }

    public function moveToDeadLetter(string $jobId, string $reason): void
    {
        if (!isset($this->jobs[$jobId])) {
            return;
        }
        
        $job = $this->jobs[$jobId];
        $this->deadLetterQueue[$jobId] = $job;
        
        // Remove from main queue
        unset($this->jobs[$jobId]);
        unset($this->retryQueue[$jobId]);
        
        foreach ($this->queues as $priority => &$queue) {
            $queue = array_filter($queue, fn(EmailJob $j) => $j->id !== $jobId);
        }
    }

    public function getQueueDepth(): int
    {
        $count = 0;
        foreach ($this->queues as $queue) {
            $count += count($queue);
        }
        return $count;
    }

    public function dequeue(): ?EmailJob
    {
        // Check retry queue first for ready jobs
        foreach ($this->retryQueue as $jobId => $job) {
            if ($job->isReadyForRetry()) {
                unset($this->retryQueue[$jobId]);
                return $job;
            }
        }
        
        // Get from main queue (priority order)
        foreach ($this->queues as $priority => &$queue) {
            if (!empty($queue)) {
                return array_shift($queue);
            }
        }
        
        return null;
    }

    public function getDeadLetterQueue(): array
    {
        return $this->deadLetterQueue;
    }

    public function getRetryQueue(): array
    {
        return $this->retryQueue;
    }

    public function getJob(string $jobId): ?EmailJob
    {
        return $this->jobs[$jobId] ?? null;
    }

    public function clear(): void
    {
        $this->queues = [];
        $this->jobs = [];
        $this->deadLetterQueue = [];
        $this->retryQueue = [];
    }
}
