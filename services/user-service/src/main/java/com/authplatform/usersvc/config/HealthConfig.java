package com.authplatform.usersvc.config;

import lombok.RequiredArgsConstructor;
import org.springframework.boot.actuate.health.Health;
import org.springframework.boot.actuate.health.HealthIndicator;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import javax.sql.DataSource;
import java.sql.Connection;

@Configuration
@RequiredArgsConstructor
public class HealthConfig {

    private final DataSource dataSource;

    @Bean
    public HealthIndicator databaseHealthIndicator() {
        return () -> {
            try (Connection conn = dataSource.getConnection()) {
                if (conn.isValid(2)) {
                    return Health.up()
                            .withDetail("database", "PostgreSQL")
                            .withDetail("status", "connected")
                            .build();
                }
            } catch (Exception e) {
                return Health.down()
                        .withDetail("database", "PostgreSQL")
                        .withDetail("error", e.getMessage())
                        .build();
            }
            return Health.down().withDetail("database", "connection invalid").build();
        };
    }

    @Bean
    public HealthIndicator livenessIndicator() {
        return () -> Health.up()
                .withDetail("service", "user-service")
                .withDetail("status", "alive")
                .build();
    }
}
