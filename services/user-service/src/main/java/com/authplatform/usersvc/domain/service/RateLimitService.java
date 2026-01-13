package com.authplatform.usersvc.domain.service;

public interface RateLimitService {
    void checkRegistrationLimit(String ipAddress);
    void checkResendLimit(String email, String ipAddress);
    void checkVerifyLimit(String ipAddress);
    boolean isAllowed(String key, int maxRequests, long windowSeconds);
}
