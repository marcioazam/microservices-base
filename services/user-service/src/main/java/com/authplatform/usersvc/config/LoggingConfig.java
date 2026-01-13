package com.authplatform.usersvc.config;

import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.filter.Filter;
import ch.qos.logback.core.spi.FilterReply;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import java.util.Set;
import java.util.regex.Pattern;

@Configuration
public class LoggingConfig {

    private static final Set<String> SENSITIVE_FIELDS = Set.of(
            "password", "passwordHash", "token", "secret", "apiKey", "authorization"
    );

    private static final Pattern EMAIL_PATTERN = Pattern.compile(
            "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"
    );

    @Bean
    public Filter<ILoggingEvent> sensitiveDataFilter() {
        return new Filter<>() {
            @Override
            public FilterReply decide(ILoggingEvent event) {
                String message = event.getFormattedMessage();
                if (message != null) {
                    for (String field : SENSITIVE_FIELDS) {
                        if (message.toLowerCase().contains(field)) {
                            return FilterReply.DENY;
                        }
                    }
                }
                return FilterReply.NEUTRAL;
            }
        };
    }

    public static String maskSensitiveData(String input) {
        if (input == null) return null;
        return EMAIL_PATTERN.matcher(input).replaceAll("[EMAIL_REDACTED]");
    }
}
