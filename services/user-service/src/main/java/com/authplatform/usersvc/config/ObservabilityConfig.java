package com.authplatform.usersvc.config;

import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.Timer;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class ObservabilityConfig {

    @Bean
    public Counter registrationCounter(MeterRegistry registry) {
        return Counter.builder("user.registration.total")
                .description("Total user registrations")
                .tag("service", "user-service")
                .register(registry);
    }

    @Bean
    public Counter verificationCounter(MeterRegistry registry) {
        return Counter.builder("user.email.verification.total")
                .description("Total email verifications")
                .tag("service", "user-service")
                .register(registry);
    }

    @Bean
    public Counter resendCounter(MeterRegistry registry) {
        return Counter.builder("user.email.resend.total")
                .description("Total verification resend requests")
                .tag("service", "user-service")
                .register(registry);
    }

    @Bean
    public Counter profileUpdateCounter(MeterRegistry registry) {
        return Counter.builder("user.profile.update.total")
                .description("Total profile updates")
                .tag("service", "user-service")
                .register(registry);
    }

    @Bean
    public Timer registrationTimer(MeterRegistry registry) {
        return Timer.builder("user.registration.duration")
                .description("Registration request duration")
                .tag("service", "user-service")
                .register(registry);
    }

    @Bean
    public Counter outboxEventCounter(MeterRegistry registry) {
        return Counter.builder("outbox.events.total")
                .description("Total outbox events published")
                .tag("service", "user-service")
                .register(registry);
    }

    @Bean
    public Counter rateLimitCounter(MeterRegistry registry) {
        return Counter.builder("rate.limit.exceeded.total")
                .description("Total rate limit exceeded events")
                .tag("service", "user-service")
                .register(registry);
    }
}
