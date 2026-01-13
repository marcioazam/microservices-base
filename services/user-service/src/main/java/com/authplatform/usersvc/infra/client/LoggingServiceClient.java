package com.authplatform.usersvc.infra.client;

import io.github.resilience4j.circuitbreaker.annotation.CircuitBreaker;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import java.time.Instant;
import java.util.Map;

@Component
@Slf4j
public class LoggingServiceClient {

    @Value("${app.logging.service.enabled:false}")
    private boolean loggingServiceEnabled;

    @CircuitBreaker(name = "loggingService", fallbackMethod = "logAuditFallback")
    public void logAudit(String eventType, String userId, Map<String, Object> metadata) {
        if (!loggingServiceEnabled) {
            logAuditLocal(eventType, userId, metadata);
            return;
        }
        // TODO: gRPC call to logging-service
        log.debug("Calling logging-service for audit event: {}", eventType);
        logAuditLocal(eventType, userId, metadata);
    }

    @CircuitBreaker(name = "loggingService", fallbackMethod = "logSecurityFallback")
    public void logSecurity(String eventType, String ipAddress, Map<String, Object> metadata) {
        if (!loggingServiceEnabled) {
            logSecurityLocal(eventType, ipAddress, metadata);
            return;
        }
        // TODO: gRPC call to logging-service
        log.debug("Calling logging-service for security event: {}", eventType);
        logSecurityLocal(eventType, ipAddress, metadata);
    }

    public void logAuditFallback(String eventType, String userId, Map<String, Object> metadata, Throwable t) {
        log.warn("Logging service unavailable, using local fallback: {}", t.getMessage());
        logAuditLocal(eventType, userId, metadata);
    }

    public void logSecurityFallback(String eventType, String ipAddress, Map<String, Object> metadata, Throwable t) {
        log.warn("Logging service unavailable, using local fallback: {}", t.getMessage());
        logSecurityLocal(eventType, ipAddress, metadata);
    }

    private void logAuditLocal(String eventType, String userId, Map<String, Object> metadata) {
        log.info("AUDIT: event={}, userId={}, timestamp={}, metadata={}",
                eventType, userId, Instant.now(), metadata);
    }

    private void logSecurityLocal(String eventType, String ipAddress, Map<String, Object> metadata) {
        log.info("SECURITY: event={}, ip={}, timestamp={}, metadata={}",
                eventType, maskIp(ipAddress), Instant.now(), metadata);
    }

    private String maskIp(String ip) {
        if (ip == null) return "unknown";
        int lastDot = ip.lastIndexOf('.');
        return lastDot > 0 ? ip.substring(0, lastDot) + ".***" : "***";
    }
}
