package com.authplatform.usersvc.api.interceptor;

import com.authplatform.usersvc.common.errors.RateLimitExceededException;
import com.authplatform.usersvc.domain.service.RateLimitService;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.HandlerInterceptor;

@Component
@RequiredArgsConstructor
@Slf4j
public class RateLimitInterceptor implements HandlerInterceptor {

    private final RateLimitService rateLimitService;

    @Override
    public boolean preHandle(HttpServletRequest request, HttpServletResponse response, Object handler) {
        String path = request.getRequestURI();
        String method = request.getMethod();
        String ipAddress = extractIpAddress(request);

        try {
            if ("POST".equals(method) && path.equals("/v1/users")) {
                rateLimitService.checkRegistrationLimit(ipAddress);
            } else if ("POST".equals(method) && path.equals("/v1/users/email/verify")) {
                rateLimitService.checkVerifyLimit(ipAddress);
            }
            // Resend rate limit is handled in controller with email
            return true;
        } catch (RateLimitExceededException e) {
            response.setStatus(429);
            response.setHeader("Retry-After", String.valueOf(e.getRetryAfterSeconds()));
            return false;
        }
    }

    private String extractIpAddress(HttpServletRequest request) {
        String xForwardedFor = request.getHeader("X-Forwarded-For");
        if (xForwardedFor != null && !xForwardedFor.isEmpty()) {
            return xForwardedFor.split(",")[0].trim();
        }
        String xRealIp = request.getHeader("X-Real-IP");
        if (xRealIp != null && !xRealIp.isEmpty()) {
            return xRealIp;
        }
        return request.getRemoteAddr();
    }
}
