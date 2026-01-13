<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Domain\Entity\Email;
use EmailService\Domain\ValueObject\Recipient;
use EmailService\Infrastructure\Queue\EmailJob;
use EmailService\Infrastructure\Queue\InMemoryQueueService;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Property 21: Queue Priority and FIFO Ordering
 * For any set of queued emails, processing order SHALL respect:
 * (1) higher priority first, (2) FIFO within same priority level.
 * 
 * Property 12: Retry Count Enforcement
 * For any email that fails delivery, the Queue_Processor SHALL retry exactly
 * up to maxAttempts times before marking as permanently failed.
 * 
 * Property 22: Exponential Backoff Timing
 * For any failed email on attempt N, the next retry SHALL be scheduled
 * after 2^(N-1) seconds.
 * 
 * Property 13: Dead Letter Queue Routing
 * For any email that has failed maxAttempts times, the Queue_Processor SHALL
 * move it to the Dead_Letter_Queue.
 * 
 * Validates: Requirements 3.2, 3.3, 6.1, 6.2, 6.3
 */
class QueueOrderingPropertyTest extends TestCase
{
    use TestTrait;

    private InMemoryQueueService $queueService;

    protected function setUp(): void
    {
        $this->queueService = new InMemoryQueueService();
    }

    /**
     * @test
     * Property 21: Higher priority jobs are dequeued first
     */
    public function higherPriorityJobsAreDequeuedFirst(): void
    {
        $this->queueService->clear();
        
        // Create jobs with different priorities
        $lowPriorityJob = EmailJob::create($this->createEmail('low'), priority: 1);
        $mediumPriorityJob = EmailJob::create($this->createEmail('medium'), priority: 5);
        $highPriorityJob = EmailJob::create($this->createEmail('high'), priority: 10);
        
        // Enqueue in random order
        $this->queueService->enqueue($lowPriorityJob);
        $this->queueService->enqueue($highPriorityJob);
        $this->queueService->enqueue($mediumPriorityJob);
        
        // Dequeue should return in priority order
        $first = $this->queueService->dequeue();
        $second = $this->queueService->dequeue();
        $third = $this->queueService->dequeue();
        
        $this->assertEquals(10, $first->priority);
        $this->assertEquals(5, $second->priority);
        $this->assertEquals(1, $third->priority);
    }

    /**
     * @test
     * Property 21: FIFO order within same priority
     */
    public function fifoOrderWithinSamePriority(): void
    {
        $this->queueService->clear();
        
        $job1 = EmailJob::create($this->createEmail('first'), priority: 5);
        $job2 = EmailJob::create($this->createEmail('second'), priority: 5);
        $job3 = EmailJob::create($this->createEmail('third'), priority: 5);
        
        $this->queueService->enqueue($job1);
        $this->queueService->enqueue($job2);
        $this->queueService->enqueue($job3);
        
        $first = $this->queueService->dequeue();
        $second = $this->queueService->dequeue();
        $third = $this->queueService->dequeue();
        
        $this->assertEquals($job1->id, $first->id);
        $this->assertEquals($job2->id, $second->id);
        $this->assertEquals($job3->id, $third->id);
    }

    /**
     * @test
     * Property 21: Mixed priorities maintain correct order
     */
    public function mixedPrioritiesMaintainCorrectOrder(): void
    {
        $this->forAll(
            Generator\choose(1, 10),
            Generator\choose(1, 10),
            Generator\choose(1, 10)
        )
        ->withMaxSize(100)
        ->then(function (int $p1, int $p2, int $p3): void {
            $this->queueService->clear();
            
            $job1 = EmailJob::create($this->createEmail('job1'), priority: $p1);
            $job2 = EmailJob::create($this->createEmail('job2'), priority: $p2);
            $job3 = EmailJob::create($this->createEmail('job3'), priority: $p3);
            
            $this->queueService->enqueue($job1);
            $this->queueService->enqueue($job2);
            $this->queueService->enqueue($job3);
            
            $dequeued = [];
            while (($job = $this->queueService->dequeue()) !== null) {
                $dequeued[] = $job->priority;
            }
            
            // Verify descending priority order
            for ($i = 0; $i < count($dequeued) - 1; $i++) {
                $this->assertGreaterThanOrEqual($dequeued[$i + 1], $dequeued[$i]);
            }
        });
    }

    /**
     * @test
     * Property 12: Job tracks attempt count correctly
     */
    public function jobTracksAttemptCountCorrectly(): void
    {
        $this->forAll(
            Generator\choose(1, 5)
        )
        ->withMaxSize(100)
        ->then(function (int $maxAttempts): void {
            $job = EmailJob::create($this->createEmail('test'), maxAttempts: $maxAttempts);
            
            $this->assertEquals(0, $job->attempts);
            
            for ($i = 1; $i <= $maxAttempts; $i++) {
                $job->incrementAttempts();
                $this->assertEquals($i, $job->attempts);
            }
            
            // After max attempts, canRetry should be false
            $this->assertFalse($job->canRetry());
        });
    }

    /**
     * @test
     * Property 12: canRetry returns true until max attempts reached
     */
    public function canRetryReturnsTrueUntilMaxAttemptsReached(): void
    {
        $maxAttempts = 3;
        $job = EmailJob::create($this->createEmail('test'), maxAttempts: $maxAttempts);
        
        for ($i = 0; $i < $maxAttempts; $i++) {
            $this->assertTrue($job->canRetry());
            $job->incrementAttempts();
        }
        
        $this->assertFalse($job->canRetry());
    }

    /**
     * @test
     * Property 22: Exponential backoff follows 2^(N-1) pattern
     */
    public function exponentialBackoffFollowsPattern(): void
    {
        $job = EmailJob::create($this->createEmail('test'), maxAttempts: 5);
        
        $expectedDelays = [1, 2, 4, 8, 16]; // 2^0, 2^1, 2^2, 2^3, 2^4
        
        for ($i = 0; $i < 5; $i++) {
            $job->incrementAttempts();
            $delay = $job->getBackoffDelay();
            $this->assertEquals($expectedDelays[$i], $delay);
        }
    }

    /**
     * @test
     * Property 22: scheduleRetry sets correct next retry time
     */
    public function scheduleRetrySetsCorrectNextRetryTime(): void
    {
        $job = EmailJob::create($this->createEmail('test'), maxAttempts: 3);
        
        $job->incrementAttempts(); // Attempt 1
        $job->scheduleRetry();
        
        $this->assertNotNull($job->nextRetryAt);
        
        // Should be approximately 1 second in the future (2^0)
        $diff = $job->nextRetryAt->getTimestamp() - time();
        $this->assertGreaterThanOrEqual(0, $diff);
        $this->assertLessThanOrEqual(2, $diff);
    }

    /**
     * @test
     * Property 13: Jobs are moved to dead letter queue after max retries
     */
    public function jobsAreMovedToDeadLetterQueueAfterMaxRetries(): void
    {
        $this->queueService->clear();
        
        $job = EmailJob::create($this->createEmail('test'), maxAttempts: 3);
        $this->queueService->enqueue($job);
        
        // Simulate max retries
        for ($i = 0; $i < 3; $i++) {
            $job->incrementAttempts();
        }
        
        $this->assertFalse($job->canRetry());
        
        $this->queueService->moveToDeadLetter($job->id, 'Max retries exceeded');
        
        $dlq = $this->queueService->getDeadLetterQueue();
        $this->assertArrayHasKey($job->id, $dlq);
        
        // Should be removed from main queue
        $this->assertNull($this->queueService->getJob($job->id));
    }

    /**
     * @test
     * Property: Queue depth is accurately tracked
     */
    public function queueDepthIsAccuratelyTracked(): void
    {
        $this->forAll(
            Generator\choose(1, 20)
        )
        ->withMaxSize(100)
        ->then(function (int $count): void {
            $this->queueService->clear();
            
            for ($i = 0; $i < $count; $i++) {
                $job = EmailJob::create($this->createEmail("job{$i}"));
                $this->queueService->enqueue($job);
            }
            
            $this->assertEquals($count, $this->queueService->getQueueDepth());
            
            // Dequeue half
            $toDequeue = (int) ($count / 2);
            for ($i = 0; $i < $toDequeue; $i++) {
                $this->queueService->dequeue();
            }
            
            $this->assertEquals($count - $toDequeue, $this->queueService->getQueueDepth());
        });
    }

    /**
     * @test
     * Property: Empty queue returns null on dequeue
     */
    public function emptyQueueReturnsNullOnDequeue(): void
    {
        $this->queueService->clear();
        
        $this->assertNull($this->queueService->dequeue());
        $this->assertEquals(0, $this->queueService->getQueueDepth());
    }

    private function createEmail(string $identifier): Email
    {
        return Email::create(
            from: new Recipient('sender@example.com'),
            recipients: [new Recipient("recipient-{$identifier}@example.com")],
            subject: "Test {$identifier}",
            body: "Body {$identifier}"
        );
    }
}
