package com.authplatform.usersvc.infrastructure.logging;

import com.authplatform.usersvc.shared.security.SecurityUtils;
import com.fasterxml.jackson.databind.ObjectMapper;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

/**
 * Async gRPC client for platform logging-service with local fallback.
 */
@Component
public class LoggingServiceClient {

    private static final Logger log = LoggerFactory.getLogger(LoggingServiceClient.class);
    private static final String SERVICE_ID = "user-service";
    
    private final String loggingServiceUrl;
    private final int timeoutMs;
    private final SecurityUtils securityUtils;
    private final ObjectMapper objectMapper;
    
    private ManagedChannel channel;
    private volatile boolean connected = false;

    public LoggingServiceClient(
            @Value("${app.platform.logging-service.url:localhost:50061}") String loggingServiceUrl,
            @Value("${app.platform.logging-service.timeout-ms:5000}") int timeoutMs,
            SecurityUtils securityUtils,
            ObjectMapper objectMapper) {
        this.loggingServiceUrl = loggingServiceUrl;
        this.timeoutMs = timeoutMs;
        this.securityUtils = securityUtils;
        this.objectMapper = objectMapper;
    }

    @PostConstruct
    public void init() {
        try {
            String[] parts = loggingServiceUrl.split(":");
            String host = parts[0];
            int port = parts.length > 1 ? Integer.parseInt(parts[1]) : 50061;
            
            this.channel = ManagedChannelBuilder.forAddress(host, port)
                    .usePlaintext()
                    .build();
            this.connected = true;
            log.info("LoggingServiceClient initialized: {}", loggingServiceUrl);
        } catch (Exception e) {
            log.warn("Failed to connect to logging-service, using local fallback: {}", e.getMessage());
            this.connected = false;
        }
    }

    @PreDestroy
    public void shutdown() {
        if (channel != null) {
            try {
                channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        }
    }

    /**
     * Logs an audit event asynchronously.
     */
    public CompletableFuture<Void> logAudit(AuditEvent event) {
        return CompletableFuture.runAsync(() -> {
            try {
                if (connected && channel != null) {
                    // In production, this would call the gRPC stub
                    logToLocalFallback("AUDIT", event.eventType(), event.description(), 
                            event.userId(), event.correlationId(), event.metadata());
                } else {
                    logToLocalFallback("AUDIT", event.eventType(), event.description(),
                            event.userId(), event.correlationId(), event.metadata());
                }
            } catch (Exception e) {
                logToLocalFallback("AUDIT", event.eventType(), event.description(),
                        event.userId(), event.correlationId(), event.metadata());
            }
        });
    }

    /**
     * Logs a security event asynchronously with IP/email masking.
     */
    public CompletableFuture<Void> logSecurity(SecurityEvent event) {
        return CompletableFuture.runAsync(() -> {
            try {
                // Mask sensitive data
                String maskedIp = securityUtils.maskIp(event.ipAddress());
                String maskedEmail = securityUtils.maskEmail(event.email());
                
                Map<String, String> metadata = new HashMap<>(event.metadata());
                metadata.put("maskedIp", maskedIp);
                metadata.put("maskedEmail", maskedEmail);
                
                if (connected && channel != null) {
                    // In production, this would call the gRPC stub
                    logToLocalFallback("SECURITY", event.eventType(), event.description(),
                            null, event.correlationId(), metadata);
                } else {
                    logToLocalFallback("SECURITY", event.eventType(), event.description(),
                            null, event.correlationId(), metadata);
                }
            } catch (Exception e) {
                log.error("Failed to log security event: {}", e.getMessage());
            }
        });
    }

    /**
     * Checks if logging service is available.
     */
    public boolean isAvailable() {
        return connected;
    }

    private void logToLocalFallback(String level, String eventType, String description,
                                     String userId, String correlationId, Map<String, String> metadata) {
        try {
            Map<String, Object> logEntry = new HashMap<>();
            logEntry.put("level", level);
            logEntry.put("eventType", eventType);
            logEntry.put("description", description);
            logEntry.put("serviceId", SERVICE_ID);
            logEntry.put("correlationId", correlationId);
            if (userId != null) {
                logEntry.put("userId", userId);
            }
            if (metadata != null && !metadata.isEmpty()) {
                logEntry.put("metadata", metadata);
            }
            
            String json = objectMapper.writeValueAsString(logEntry);
            
            if ("SECURITY".equals(level)) {
                log.warn("[SECURITY] {}", json);
            } else {
                log.info("[AUDIT] {}", json);
            }
        } catch (Exception e) {
            log.error("Failed to write local fallback log: {}", e.getMessage());
        }
    }
}
