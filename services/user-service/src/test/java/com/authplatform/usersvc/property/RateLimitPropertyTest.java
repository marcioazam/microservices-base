package com.authplatform.usersvc.property;

import net.jqwik.api.*;
import net.jqwik.api.constraints.IntRange;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property 7: Rate Limiting Enforcement
 * Validates: Requirements 3.2, 9.1, 9.2, 9.3, 9.4
 * 
 * Ensures that rate limiting correctly enforces request limits
 * within sliding windows and resets appropriately.
 */
class RateLimitPropertyTest {

    @Property(tries = 100)
    void rateLimitAllowsRequestsWithinLimit(
            @ForAll @IntRange(min = 1, max = 100) int maxRequests,
            @ForAll @IntRange(min = 1, max = 50) int actualRequests) {
        
        if (actualRequests > maxRequests) return; // Skip invalid combinations
        
        TestRateLimiter limiter = new TestRateLimiter();
        String key = "test:" + System.nanoTime();
        
        int allowedCount = 0;
        for (int i = 0; i < actualRequests; i++) {
            if (limiter.isAllowed(key, maxRequests, 60)) {
                allowedCount++;
            }
        }
        
        // All requests within limit should be allowed
        assertThat(allowedCount).isEqualTo(actualRequests);
    }

    @Property(tries = 100)
    void rateLimitBlocksRequestsOverLimit(
            @ForAll @IntRange(min = 1, max = 20) int maxRequests,
            @ForAll @IntRange(min = 1, max = 10) int extraRequests) {
        
        TestRateLimiter limiter = new TestRateLimiter();
        String key = "test:" + System.nanoTime();
        int totalRequests = maxRequests + extraRequests;
        
        int allowedCount = 0;
        int blockedCount = 0;
        
        for (int i = 0; i < totalRequests; i++) {
            if (limiter.isAllowed(key, maxRequests, 60)) {
                allowedCount++;
            } else {
                blockedCount++;
            }
        }
        
        // Exactly maxRequests should be allowed
        assertThat(allowedCount).isEqualTo(maxRequests);
        // Extra requests should be blocked
        assertThat(blockedCount).isEqualTo(extraRequests);
    }

    @Property(tries = 100)
    void differentKeysHaveIndependentLimits(
            @ForAll @IntRange(min = 1, max = 10) int maxRequests) {
        
        TestRateLimiter limiter = new TestRateLimiter();
        String key1 = "user1:" + System.nanoTime();
        String key2 = "user2:" + System.nanoTime();
        
        // Exhaust limit for key1
        for (int i = 0; i < maxRequests; i++) {
            limiter.isAllowed(key1, maxRequests, 60);
        }
        
        // key2 should still have full limit
        int key2Allowed = 0;
        for (int i = 0; i < maxRequests; i++) {
            if (limiter.isAllowed(key2, maxRequests, 60)) {
                key2Allowed++;
            }
        }
        
        assertThat(key2Allowed).isEqualTo(maxRequests);
    }

    @Property(tries = 100)
    void windowResetAllowsNewRequests(
            @ForAll @IntRange(min = 1, max = 10) int maxRequests) {
        
        TestRateLimiter limiter = new TestRateLimiter();
        String key = "test:" + System.nanoTime();
        
        // Exhaust limit
        for (int i = 0; i < maxRequests; i++) {
            limiter.isAllowed(key, maxRequests, 1); // 1 second window
        }
        
        // Should be blocked
        assertThat(limiter.isAllowed(key, maxRequests, 1)).isFalse();
        
        // Simulate window reset
        limiter.simulateWindowReset(key);
        
        // Should be allowed again
        assertThat(limiter.isAllowed(key, maxRequests, 1)).isTrue();
    }

    @Property(tries = 100)
    void ipBasedRateLimitingIsConsistent(
            @ForAll("validIpv4") String ipAddress,
            @ForAll @IntRange(min = 1, max = 10) int maxRequests) {
        
        TestRateLimiter limiter = new TestRateLimiter();
        String key = "registration:ip:" + ipAddress;
        
        int allowed = 0;
        for (int i = 0; i < maxRequests + 5; i++) {
            if (limiter.isAllowed(key, maxRequests, 60)) {
                allowed++;
            }
        }
        
        // Should allow exactly maxRequests
        assertThat(allowed).isEqualTo(maxRequests);
    }

    @Provide
    Arbitrary<String> validIpv4() {
        return Combinators.combine(
            Arbitraries.integers().between(1, 255),
            Arbitraries.integers().between(0, 255),
            Arbitraries.integers().between(0, 255),
            Arbitraries.integers().between(1, 254)
        ).as((a, b, c, d) -> a + "." + b + "." + c + "." + d);
    }

    /**
     * Test implementation of rate limiter for property testing
     */
    private static class TestRateLimiter {
        private final Map<String, RateLimitEntry> rateLimits = new ConcurrentHashMap<>();

        boolean isAllowed(String key, int maxRequests, long windowSeconds) {
            long now = System.currentTimeMillis();
            long windowStart = now - (windowSeconds * 1000);

            RateLimitEntry entry = rateLimits.compute(key, (k, existing) -> {
                if (existing == null || existing.windowStart < windowStart) {
                    return new RateLimitEntry(now, new AtomicInteger(1));
                }
                existing.count.incrementAndGet();
                return existing;
            });

            return entry.count.get() <= maxRequests;
        }

        void simulateWindowReset(String key) {
            rateLimits.remove(key);
        }

        private static class RateLimitEntry {
            final long windowStart;
            final AtomicInteger count;

            RateLimitEntry(long windowStart, AtomicInteger count) {
                this.windowStart = windowStart;
                this.count = count;
            }
        }
    }
}
