package com.authplatform.usersvc.infrastructure.logging;

import java.time.Instant;
import java.util.Map;

/**
 * Represents an audit event for logging.
 */
public record AuditEvent(
        String eventType,
        String userId,
        String correlationId,
        String description,
        Map<String, String> metadata,
        Instant timestamp
) {
    public static AuditEvent of(String eventType, String userId, String correlationId, String description) {
        return new AuditEvent(eventType, userId, correlationId, description, Map.of(), Instant.now());
    }

    public static AuditEvent of(String eventType, String userId, String correlationId, 
                                 String description, Map<String, String> metadata) {
        return new AuditEvent(eventType, userId, correlationId, description, metadata, Instant.now());
    }
}
