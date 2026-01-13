package com.authplatform.usersvc.infrastructure.logging;

import java.time.Instant;
import java.util.Map;

/**
 * Represents a security event for logging.
 */
public record SecurityEvent(
        String eventType,
        String ipAddress,
        String email,
        String correlationId,
        String description,
        Map<String, String> metadata,
        Instant timestamp
) {
    public static SecurityEvent of(String eventType, String ipAddress, String email, 
                                    String correlationId, String description) {
        return new SecurityEvent(eventType, ipAddress, email, correlationId, description, Map.of(), Instant.now());
    }

    public static SecurityEvent of(String eventType, String ipAddress, String email,
                                    String correlationId, String description, Map<String, String> metadata) {
        return new SecurityEvent(eventType, ipAddress, email, correlationId, description, metadata, Instant.now());
    }
}
