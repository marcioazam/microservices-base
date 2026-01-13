<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Infrastructure\Platform\InMemoryCacheClient;
use EmailService\Infrastructure\RateLimiter\CacheBackedRateLimiter;
use EmailService\Infrastructure\RateLimiter\InMemoryRateLimiter;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Feature: email-service-modernization-2025
 * Property 3: Rate Limiting State Consistency
 * 
 * For any sender identifier, the rate limit count tracked via Cache_Service
 * SHALL accurately reflect the number of hits recorded, and the remaining
 * count SHALL equal (limit - hits).
 * 
 * Validates: Requirements 1.3
 */
class CacheBackedRateLimitPropertyTest extends TestCase
{
    use TestTrait;

    private InMemoryCacheClient $cacheClient;
    private CacheBackedRateLimiter $rateLimiter;

    protected function setUp(): void
    {
        $this->cacheClient = new InMemoryCacheClient();
    }

    /**
     * @test
     * Property 3: Count accurately reflects number of hits
     */
    public function countAccuratelyReflectsNumberOfHits(): void
    {
        $limit = 100;
        $this->rateLimiter = new CacheBackedRateLimiter(
            cacheClient: $this->cacheClient,
            limit: $limit,
            windowSeconds: 60
        );

        $this->forAll(
            Generator\choose(1, 50)
        )
        ->withMaxSize(100)
        ->then(function (int $hitCount) use ($limit): void {
            $this->cacheClient->clear();
            $sender = 'sender-' . uniqid();

            for ($i = 0; $i < $hitCount; $i++) {
                $this->rateLimiter->hit($sender);
            }

            $count = $this->rateLimiter->getCount($sender);
            $this->assertEquals($hitCount, $count);
        });
    }

    /**
     * @test
     * Property 3: Remaining equals limit minus hits
     */
    public function remainingEqualsLimitMinusHits(): void
    {
        $limit = 20;
        $this->rateLimiter = new CacheBackedRateLimiter(
            cacheClient: $this->cacheClient,
            limit: $limit,
            windowSeconds: 60
        );

        $this->forAll(
            Generator\choose(1, 15)
        )
        ->withMaxSize(100)
        ->then(function (int $hitCount) use ($limit): void {
            $this->cacheClient->clear();
            $sender = 'sender-' . uniqid();

            for ($i = 0; $i < $hitCount; $i++) {
                $this->rateLimiter->hit($sender);
            }

            $result = $this->rateLimiter->check($sender);
            $expectedRemaining = $limit - $hitCount;

            $this->assertEquals($expectedRemaining, $result->remaining);
        });
    }

    /**
     * @test
     * Property 3: Rate limit is enforced at limit
     */
    public function rateLimitIsEnforcedAtLimit(): void
    {
        $limit = 5;
        $this->rateLimiter = new CacheBackedRateLimiter(
            cacheClient: $this->cacheClient,
            limit: $limit,
            windowSeconds: 60
        );

        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $sender) use ($limit): void {
            $this->cacheClient->clear();

            // Use up all the limit
            for ($i = 0; $i < $limit; $i++) {
                $result = $this->rateLimiter->hit($sender);
                $this->assertTrue($result->isAllowed);
            }

            // Next request should be rejected
            $result = $this->rateLimiter->hit($sender);
            $this->assertFalse($result->isAllowed);
            $this->assertEquals(0, $result->remaining);
            $this->assertNotNull($result->retryAfter);
        });
    }

    /**
     * @test
     * Property 3: Each sender has independent rate limit
     */
    public function eachSenderHasIndependentRateLimit(): void
    {
        $limit = 10;
        $this->rateLimiter = new CacheBackedRateLimiter(
            cacheClient: $this->cacheClient,
            limit: $limit,
            windowSeconds: 60
        );

        $this->cacheClient->clear();

        $sender1 = 'sender-1';
        $sender2 = 'sender-2';

        // Hit sender1 multiple times
        for ($i = 0; $i < 5; $i++) {
            $this->rateLimiter->hit($sender1);
        }

        // Sender2 should still have full limit
        $result2 = $this->rateLimiter->check($sender2);
        $this->assertTrue($result2->isAllowed);
        $this->assertEquals($limit, $result2->remaining);

        // Sender1 should have reduced remaining
        $result1 = $this->rateLimiter->check($sender1);
        $this->assertTrue($result1->isAllowed);
        $this->assertEquals($limit - 5, $result1->remaining);
    }

    /**
     * @test
     * Property 3: Reset clears the count
     */
    public function resetClearsTheCount(): void
    {
        $limit = 100;
        $this->rateLimiter = new CacheBackedRateLimiter(
            cacheClient: $this->cacheClient,
            limit: $limit,
            windowSeconds: 60
        );

        $this->forAll(
            Generator\choose(1, 20)
        )
        ->withMaxSize(100)
        ->then(function (int $hitCount) use ($limit): void {
            $this->cacheClient->clear();
            $sender = 'reset-test-' . uniqid();

            for ($i = 0; $i < $hitCount; $i++) {
                $this->rateLimiter->hit($sender);
            }

            $this->assertEquals($hitCount, $this->rateLimiter->getCount($sender));

            $this->rateLimiter->reset($sender);

            $this->assertEquals(0, $this->rateLimiter->getCount($sender));

            $result = $this->rateLimiter->check($sender);
            $this->assertTrue($result->isAllowed);
            $this->assertEquals($limit, $result->remaining);
        });
    }

    /**
     * @test
     * Property 3: Fallback is used when cache fails
     */
    public function fallbackIsUsedWhenCacheFails(): void
    {
        $limit = 10;
        $fallback = new InMemoryRateLimiter($limit, 60);

        // Create a mock cache client that always throws
        $failingCache = $this->createMock(InMemoryCacheClient::class);
        $failingCache->method('get')->willThrowException(new \RuntimeException('Cache unavailable'));
        $failingCache->method('set')->willThrowException(new \RuntimeException('Cache unavailable'));

        $rateLimiter = new CacheBackedRateLimiter(
            cacheClient: $failingCache,
            limit: $limit,
            windowSeconds: 60,
            fallback: $fallback
        );

        $sender = 'fallback-test';

        // Should use fallback and not throw
        $result = $rateLimiter->check($sender);
        $this->assertTrue($result->isAllowed);
        $this->assertEquals($limit, $result->remaining);
    }

    /**
     * @test
     * Property 3: isApproachingLimit correctly identifies when near limit
     */
    public function isApproachingLimitCorrectlyIdentifiesNearLimit(): void
    {
        $limit = 10;
        $this->rateLimiter = new CacheBackedRateLimiter(
            cacheClient: $this->cacheClient,
            limit: $limit,
            windowSeconds: 60
        );

        $this->cacheClient->clear();
        $sender = 'approaching-sender';

        // At 0% usage - not approaching
        $result = $this->rateLimiter->check($sender);
        $this->assertFalse($result->isApproachingLimit(0.8));

        // Use 7 of 10 (70%) - not approaching 80%
        for ($i = 0; $i < 7; $i++) {
            $this->rateLimiter->hit($sender);
        }
        $result = $this->rateLimiter->check($sender);
        $this->assertFalse($result->isApproachingLimit(0.8));

        // Use 1 more (80%) - now approaching
        $this->rateLimiter->hit($sender);
        $result = $this->rateLimiter->check($sender);
        $this->assertTrue($result->isApproachingLimit(0.8));
    }

    /**
     * @test
     * Property 3: Retry-after is provided when limit exceeded
     */
    public function retryAfterIsProvidedWhenLimitExceeded(): void
    {
        $limit = 3;
        $windowSeconds = 60;
        $this->rateLimiter = new CacheBackedRateLimiter(
            cacheClient: $this->cacheClient,
            limit: $limit,
            windowSeconds: $windowSeconds
        );

        $this->cacheClient->clear();
        $sender = 'retry-test';

        // Exhaust the limit
        for ($i = 0; $i < $limit + 1; $i++) {
            $this->rateLimiter->hit($sender);
        }

        $result = $this->rateLimiter->check($sender);

        $this->assertFalse($result->isAllowed);
        $this->assertNotNull($result->retryAfter);
        $this->assertLessThanOrEqual($windowSeconds, $result->retryAfter);
        $this->assertGreaterThan(0, $result->retryAfter);
    }
}
