package com.authplatform.usersvc.domain.ratelimit;

import com.authplatform.usersvc.infrastructure.cache.CacheServiceClient;
import com.authplatform.usersvc.shared.exception.RateLimitedException;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.nio.ByteBuffer;
import java.time.Duration;
import java.time.Instant;
import java.util.Optional;

/**
 * Distributed rate limiting service using cache-service.
 * Uses namespace "user-service:ratelimit" for all keys.
 */
@Service
public class RateLimitService {

    public static final String NAMESPACE = "user-service:ratelimit";
    
    private final CacheServiceClient cacheClient;
    private final int registrationLimitPerHour;
    private final int resendLimitPerHour;
    private final int verifyLimitPerHour;

    public RateLimitService(
            CacheServiceClient cacheClient,
            @Value("${app.rate-limit.registration.per-minute:5}") int registrationLimitPerMinute,
            @Value("${app.rate-limit.resend.per-hour:3}") int resendLimitPerHour,
            @Value("${app.rate-limit.verify.per-minute:10}") int verifyLimitPerMinute) {
        this.cacheClient = cacheClient;
        this.registrationLimitPerHour = registrationLimitPerMinute * 60; // Convert to per hour
        this.resendLimitPerHour = resendLimitPerHour;
        this.verifyLimitPerHour = verifyLimitPerMinute * 60; // Convert to per hour
    }

    /**
     * Checks registration rate limit by IP.
     * @throws RateLimitedException if limit exceeded
     */
    public void checkRegistrationLimit(String ipAddress) {
        String key = "registration:ip:" + ipAddress;
        checkLimit(key, registrationLimitPerHour, Duration.ofHours(1));
    }

    /**
     * Checks resend verification rate limit by email and IP.
     * @throws RateLimitedException if limit exceeded
     */
    public void checkResendLimit(String email, String ipAddress) {
        // Check email-based limit (stricter: 3 per hour)
        String emailKey = "resend:email:" + email.toLowerCase();
        checkLimit(emailKey, resendLimitPerHour, Duration.ofHours(1));
        
        // Check IP-based limit
        String ipKey = "resend:ip:" + ipAddress;
        checkLimit(ipKey, registrationLimitPerHour, Duration.ofHours(1));
    }

    /**
     * Checks verification rate limit by IP.
     * @throws RateLimitedException if limit exceeded
     */
    public void checkVerifyLimit(String ipAddress) {
        String key = "verify:ip:" + ipAddress;
        checkLimit(key, verifyLimitPerHour, Duration.ofHours(1));
    }

    /**
     * Gets the full cache key with namespace prefix.
     */
    public String getFullKey(String key) {
        return NAMESPACE + ":" + key;
    }

    private void checkLimit(String key, int limit, Duration window) {
        Optional<byte[]> cached = cacheClient.get(NAMESPACE, key);
        
        int currentCount = 0;
        long windowStart = Instant.now().toEpochMilli();
        
        if (cached.isPresent()) {
            byte[] data = cached.get();
            if (data.length >= 12) {
                ByteBuffer buffer = ByteBuffer.wrap(data);
                currentCount = buffer.getInt();
                windowStart = buffer.getLong();
            }
        }
        
        long now = Instant.now().toEpochMilli();
        long windowEnd = windowStart + window.toMillis();
        
        // Reset if window expired
        if (now > windowEnd) {
            currentCount = 0;
            windowStart = now;
        }
        
        if (currentCount >= limit) {
            long retryAfterSeconds = (windowEnd - now) / 1000;
            throw new RateLimitedException(Duration.ofSeconds(Math.max(1, retryAfterSeconds)));
        }
        
        // Increment counter
        currentCount++;
        ByteBuffer buffer = ByteBuffer.allocate(12);
        buffer.putInt(currentCount);
        buffer.putLong(windowStart);
        
        cacheClient.set(NAMESPACE, key, buffer.array(), window);
    }
}
