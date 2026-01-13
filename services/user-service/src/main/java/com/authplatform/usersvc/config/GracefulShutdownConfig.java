package com.authplatform.usersvc.config;

import lombok.extern.slf4j.Slf4j;
import org.springframework.context.ApplicationListener;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.event.ContextClosedEvent;
import java.util.concurrent.atomic.AtomicBoolean;

@Configuration
@Slf4j
public class GracefulShutdownConfig {

    private final AtomicBoolean shuttingDown = new AtomicBoolean(false);

    @Bean
    public ApplicationListener<ContextClosedEvent> gracefulShutdownListener() {
        return event -> {
            log.info("Received shutdown signal, initiating graceful shutdown");
            shuttingDown.set(true);
            
            try {
                // Allow in-flight requests to complete
                Thread.sleep(5000);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                log.warn("Graceful shutdown interrupted");
            }
            
            log.info("Graceful shutdown complete");
        };
    }

    public boolean isShuttingDown() {
        return shuttingDown.get();
    }
}
