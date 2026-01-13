<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Infrastructure\RateLimiter\InMemoryRateLimiter;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Property 27: Per-Sender Rate Limiting
 * For any sender, the Rate_Limiter SHALL track and enforce the configured limit
 * independently of other senders.
 * 
 * Property 9: Rate Limit Enforcement
 * For any sender who has sent N emails where N ≥ configured limit,
 * subsequent send requests SHALL return HTTP 429 with retry-after header.
 * 
 * Property 29: Graceful Rate Limit Approach
 * For any sender approaching rate limit (≥80% of limit), new emails SHALL be
 * queued for delayed delivery rather than rejected.
 * 
 * Validates: Requirements 2.4, 9.1, 9.3, 9.4
 */
class RateLimitPropertyTest extends TestCase
{
    use TestTrait;

    /**
     * @test
     * Property 27: Each sender has independent rate limit tracking
     */
    public function eachSenderHasIndependentRateLimitTracking(): void
    {
        $limit = 10;
        $rateLimiter = new InMemoryRateLimiter($limit, 60);

        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $sender1, string $sender2) use ($rateLimiter, $limit): void {
            if ($sender1 === $sender2) {
                $this->markTestSkipped('Senders are the same');
                return;
            }
            
            // Reset both senders
            $rateLimiter->reset($sender1);
            $rateLimiter->reset($sender2);
            
            // Hit sender1 multiple times
            for ($i = 0; $i < 5; $i++) {
                $rateLimiter->hit($sender1);
            }
            
            // Sender2 should still have full limit
            $result2 = $rateLimiter->check($sender2);
            $this->assertTrue($result2->isAllowed);
            $this->assertEquals($limit, $result2->remaining);
            
            // Sender1 should have reduced remaining
            $result1 = $rateLimiter->check($sender1);
            $this->assertTrue($result1->isAllowed);
            $this->assertEquals($limit - 5, $result1->remaining);
        });
    }

    /**
     * @test
     * Property 9: Rate limit is enforced after reaching limit
     */
    public function rateLimitIsEnforcedAfterReachingLimit(): void
    {
        $limit = 5;
        $rateLimiter = new InMemoryRateLimiter($limit, 60);

        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $sender) use ($rateLimiter, $limit): void {
            $rateLimiter->reset($sender);
            
            // Use up all the limit
            for ($i = 0; $i < $limit; $i++) {
                $result = $rateLimiter->hit($sender);
                $this->assertTrue($result->isAllowed);
            }
            
            // Next request should be rejected
            $result = $rateLimiter->hit($sender);
            $this->assertFalse($result->isAllowed);
            $this->assertEquals(0, $result->remaining);
            $this->assertNotNull($result->retryAfter);
            $this->assertGreaterThan(0, $result->retryAfter);
        });
    }

    /**
     * @test
     * Property 9: Retry-after header is provided when limit exceeded
     */
    public function retryAfterIsProvidedWhenLimitExceeded(): void
    {
        $limit = 3;
        $windowSeconds = 60;
        $rateLimiter = new InMemoryRateLimiter($limit, $windowSeconds);

        $sender = 'test-sender';
        $rateLimiter->reset($sender);
        
        // Exhaust the limit
        for ($i = 0; $i < $limit + 1; $i++) {
            $rateLimiter->hit($sender);
        }
        
        $result = $rateLimiter->check($sender);
        
        $this->assertFalse($result->isAllowed);
        $this->assertNotNull($result->retryAfter);
        $this->assertLessThanOrEqual($windowSeconds, $result->retryAfter);
        $this->assertGreaterThan(0, $result->retryAfter);
    }

    /**
     * @test
     * Property 29: isApproachingLimit correctly identifies when near limit
     */
    public function isApproachingLimitCorrectlyIdentifiesNearLimit(): void
    {
        $limit = 10;
        $rateLimiter = new InMemoryRateLimiter($limit, 60);

        $sender = 'approaching-sender';
        $rateLimiter->reset($sender);
        
        // At 0% usage - not approaching
        $result = $rateLimiter->check($sender);
        $this->assertFalse($result->isApproachingLimit(0.8));
        
        // Use 7 of 10 (70%) - not approaching 80%
        for ($i = 0; $i < 7; $i++) {
            $rateLimiter->hit($sender);
        }
        $result = $rateLimiter->check($sender);
        $this->assertFalse($result->isApproachingLimit(0.8));
        
        // Use 1 more (80%) - now approaching
        $rateLimiter->hit($sender);
        $result = $rateLimiter->check($sender);
        $this->assertTrue($result->isApproachingLimit(0.8));
        
        // Use 1 more (90%) - still approaching
        $rateLimiter->hit($sender);
        $result = $rateLimiter->check($sender);
        $this->assertTrue($result->isApproachingLimit(0.8));
    }

    /**
     * @test
     * Property: Count is accurately tracked
     */
    public function countIsAccuratelyTracked(): void
    {
        $rateLimiter = new InMemoryRateLimiter(100, 60);

        $this->forAll(
            Generator\choose(1, 50)
        )
        ->withMaxSize(100)
        ->then(function (int $hitCount) use ($rateLimiter): void {
            $sender = 'count-test-' . uniqid();
            $rateLimiter->reset($sender);
            
            for ($i = 0; $i < $hitCount; $i++) {
                $rateLimiter->hit($sender);
            }
            
            $this->assertEquals($hitCount, $rateLimiter->getCount($sender));
        });
    }

    /**
     * @test
     * Property: Reset clears the count
     */
    public function resetClearsTheCount(): void
    {
        $rateLimiter = new InMemoryRateLimiter(100, 60);

        $this->forAll(
            Generator\choose(1, 20)
        )
        ->withMaxSize(100)
        ->then(function (int $hitCount) use ($rateLimiter): void {
            $sender = 'reset-test-' . uniqid();
            
            for ($i = 0; $i < $hitCount; $i++) {
                $rateLimiter->hit($sender);
            }
            
            $this->assertEquals($hitCount, $rateLimiter->getCount($sender));
            
            $rateLimiter->reset($sender);
            
            $this->assertEquals(0, $rateLimiter->getCount($sender));
            
            $result = $rateLimiter->check($sender);
            $this->assertTrue($result->isAllowed);
            $this->assertEquals(100, $result->remaining);
        });
    }

    /**
     * @test
     * Property: Remaining count decreases with each hit
     */
    public function remainingCountDecreasesWithEachHit(): void
    {
        $limit = 20;
        $rateLimiter = new InMemoryRateLimiter($limit, 60);

        $sender = 'decrement-test';
        $rateLimiter->reset($sender);

        for ($i = 0; $i < $limit; $i++) {
            $result = $rateLimiter->hit($sender);
            
            $expectedRemaining = $limit - ($i + 1);
            $this->assertEquals($expectedRemaining, $result->remaining);
        }
    }
}
