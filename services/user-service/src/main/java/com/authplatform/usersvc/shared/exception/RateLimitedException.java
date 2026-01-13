package com.authplatform.usersvc.shared.exception;

import java.time.Duration;

public final class RateLimitedException extends UserServiceException {
    
    private final Duration retryAfter;

    public RateLimitedException(Duration retryAfter) {
        super("Rate limit exceeded");
        this.retryAfter = retryAfter;
    }

    public Duration getRetryAfter() {
        return retryAfter;
    }

    public long getRetryAfterSeconds() {
        return retryAfter.toSeconds();
    }

    @Override
    public String getErrorCode() {
        return "RATE_LIMITED";
    }

    @Override
    public int getHttpStatus() {
        return 429;
    }
}
