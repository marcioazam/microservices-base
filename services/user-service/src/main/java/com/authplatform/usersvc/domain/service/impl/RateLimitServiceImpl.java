package com.authplatform.usersvc.domain.service.impl;

import com.authplatform.usersvc.common.errors.RateLimitExceededException;
import com.authplatform.usersvc.domain.service.RateLimitService;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;

@Service
@Slf4j
public class RateLimitServiceImpl implements RateLimitService {

    private final Map<String, RateLimitEntry> rateLimits = new ConcurrentHashMap<>();

    @Value("${app.rate-limit.registration.per-minute:5}")
    private int registrationPerMinute;

    @Value("${app.rate-limit.resend.per-hour:3}")
    private int resendPerHour;

    @Value("${app.rate-limit.verify.per-minute:10}")
    private int verifyPerMinute;

    @Override
    public void checkRegistrationLimit(String ipAddress) {
        String key = "registration:ip:" + ipAddress;
        if (!isAllowed(key, registrationPerMinute, 60)) {
            log.warn("Registration rate limit exceeded for IP: {}", maskIp(ipAddress));
            throw new RateLimitExceededException("Too many registration attempts", 60);
        }
    }

    @Override
    public void checkResendLimit(String email, String ipAddress) {
        String emailKey = "resend:email:" + email;
        String ipKey = "resend:ip:" + ipAddress;

        if (!isAllowed(emailKey, resendPerHour, 3600)) {
            log.warn("Resend rate limit exceeded for email");
            throw new RateLimitExceededException("Too many resend attempts", 3600);
        }

        if (!isAllowed(ipKey, 10, 3600)) {
            log.warn("Resend rate limit exceeded for IP: {}", maskIp(ipAddress));
            throw new RateLimitExceededException("Too many resend attempts", 3600);
        }
    }

    @Override
    public void checkVerifyLimit(String ipAddress) {
        String key = "verify:ip:" + ipAddress;
        if (!isAllowed(key, verifyPerMinute, 60)) {
            log.warn("Verify rate limit exceeded for IP: {}", maskIp(ipAddress));
            throw new RateLimitExceededException("Too many verification attempts", 60);
        }
    }

    @Override
    public boolean isAllowed(String key, int maxRequests, long windowSeconds) {
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

    private String maskIp(String ip) {
        if (ip == null) return "unknown";
        int lastDot = ip.lastIndexOf('.');
        return lastDot > 0 ? ip.substring(0, lastDot) + ".***" : "***";
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
