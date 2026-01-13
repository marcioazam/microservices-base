package com.authplatform.usersvc.api.controller;

import com.authplatform.usersvc.infrastructure.cache.CacheServiceClient;
import com.authplatform.usersvc.infrastructure.logging.LoggingServiceClient;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import javax.sql.DataSource;
import java.sql.Connection;
import java.util.HashMap;
import java.util.Map;

@RestController
@RequiredArgsConstructor
@Tag(name = "Health", description = "Health check endpoints")
public class HealthController {

    private final DataSource dataSource;
    private final CacheServiceClient cacheClient;
    private final LoggingServiceClient loggingClient;

    @GetMapping("/health")
    @Operation(summary = "Basic health check")
    public ResponseEntity<Map<String, String>> health() {
        return ResponseEntity.ok(Map.of("status", "UP"));
    }

    @GetMapping("/health/ready")
    @Operation(summary = "Readiness check", description = "Checks database, cache, and logging service")
    public ResponseEntity<Map<String, Object>> ready() {
        Map<String, Object> health = new HashMap<>();
        boolean allHealthy = true;
        
        // Check database
        try (Connection conn = dataSource.getConnection()) {
            health.put("database", Map.of("status", "UP"));
        } catch (Exception e) {
            health.put("database", Map.of("status", "DOWN", "error", e.getMessage()));
            allHealthy = false;
        }
        
        // Check cache service
        if (cacheClient.isAvailable()) {
            health.put("cacheService", Map.of("status", "UP"));
        } else {
            health.put("cacheService", Map.of("status", "DOWN", "fallback", "local"));
        }
        
        // Check logging service
        if (loggingClient.isAvailable()) {
            health.put("loggingService", Map.of("status", "UP"));
        } else {
            health.put("loggingService", Map.of("status", "DOWN", "fallback", "local"));
        }
        
        health.put("status", allHealthy ? "UP" : "DEGRADED");
        
        return ResponseEntity.ok(health);
    }
}
