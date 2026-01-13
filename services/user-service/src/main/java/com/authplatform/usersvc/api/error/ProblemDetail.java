package com.authplatform.usersvc.api.error;

import java.time.Instant;
import java.util.Map;

/**
 * RFC 7807 Problem Detail response format.
 */
public record ProblemDetail(
        String type,
        String title,
        int status,
        String detail,
        String instance,
        Instant timestamp,
        String correlationId,
        String errorCode,
        Map<String, Object> extensions
) {
    public static ProblemDetail of(
            String type, String title, int status, String detail,
            String instance, String correlationId, String errorCode) {
        return new ProblemDetail(type, title, status, detail, instance, 
                Instant.now(), correlationId, errorCode, Map.of());
    }

    public static ProblemDetail of(
            String type, String title, int status, String detail,
            String instance, String correlationId, String errorCode,
            Map<String, Object> extensions) {
        return new ProblemDetail(type, title, status, detail, instance,
                Instant.now(), correlationId, errorCode, extensions);
    }
}
