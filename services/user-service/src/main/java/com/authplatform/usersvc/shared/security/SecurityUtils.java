package com.authplatform.usersvc.shared.security;

import org.slf4j.MDC;
import org.springframework.stereotype.Component;

import java.util.UUID;
import java.util.regex.Pattern;

/**
 * Centralized security utilities for IP/email masking, correlation ID, and MDC management.
 * Single source of truth for all security-related utility functions.
 */
@Component
public class SecurityUtils {

    private static final String CORRELATION_ID_KEY = "correlationId";
    private static final String USER_ID_KEY = "userId";
    private static final Pattern IPV4_PATTERN = Pattern.compile("^(\\d{1,3}\\.\\d{1,3}\\.\\d{1,3})\\.\\d{1,3}$");
    private static final Pattern EMAIL_PATTERN = Pattern.compile("^(.{2})([^@]*)(@.+)$");

    /**
     * Masks an IPv4 address by replacing the last octet with ***.
     * Example: 192.168.1.100 -> 192.168.1.***
     */
    public String maskIp(String ip) {
        if (ip == null || ip.isBlank()) {
            return "***";
        }
        var matcher = IPV4_PATTERN.matcher(ip.trim());
        if (matcher.matches()) {
            return matcher.group(1) + ".***";
        }
        // For IPv6 or invalid, mask more aggressively
        return ip.length() > 6 ? ip.substring(0, 6) + "***" : "***";
    }

    /**
     * Masks an email address by keeping first 2 characters before @ and masking the rest.
     * Example: john.doe@example.com -> jo***@example.com
     */
    public String maskEmail(String email) {
        if (email == null || email.isBlank()) {
            return "***";
        }
        var matcher = EMAIL_PATTERN.matcher(email.trim().toLowerCase());
        if (matcher.matches()) {
            return matcher.group(1) + "***" + matcher.group(3);
        }
        return email.length() > 2 ? email.substring(0, 2) + "***" : "***";
    }

    /**
     * Gets existing correlation ID or creates a new one.
     */
    public String getOrCreateCorrelationId(String provided) {
        if (provided != null && !provided.isBlank()) {
            return provided.trim();
        }
        return UUID.randomUUID().toString();
    }

    /**
     * Sets MDC context for structured logging.
     */
    public void setMdcContext(String correlationId, String userId) {
        if (correlationId != null) {
            MDC.put(CORRELATION_ID_KEY, correlationId);
        }
        if (userId != null) {
            MDC.put(USER_ID_KEY, userId);
        }
    }

    /**
     * Clears MDC context after request processing.
     */
    public void clearMdcContext() {
        MDC.remove(CORRELATION_ID_KEY);
        MDC.remove(USER_ID_KEY);
    }

    /**
     * Gets current correlation ID from MDC.
     */
    public String getCurrentCorrelationId() {
        return MDC.get(CORRELATION_ID_KEY);
    }
}
