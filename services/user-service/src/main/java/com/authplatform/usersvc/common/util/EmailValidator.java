package com.authplatform.usersvc.common.util;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import java.util.ArrayList;
import java.util.List;
import java.util.Set;
import java.util.regex.Pattern;

@Component
public class EmailValidator {

    private static final Pattern EMAIL_PATTERN = Pattern.compile(
            "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    );

    private static final Set<String> DISPOSABLE_DOMAINS = Set.of(
            "tempmail.com", "throwaway.com", "mailinator.com",
            "guerrillamail.com", "10minutemail.com", "fakeinbox.com",
            "trashmail.com", "yopmail.com", "temp-mail.org"
    );

    private final Set<String> blockedDomains;

    public EmailValidator(@Value("${app.email.blocked-domains:}") String additionalBlockedDomains) {
        if (additionalBlockedDomains != null && !additionalBlockedDomains.isBlank()) {
            Set<String> additional = Set.of(additionalBlockedDomains.split(","));
            this.blockedDomains = new java.util.HashSet<>(DISPOSABLE_DOMAINS);
            this.blockedDomains.addAll(additional);
        } else {
            this.blockedDomains = DISPOSABLE_DOMAINS;
        }
    }

    public ValidationResult validate(String email) {
        List<String> errors = new ArrayList<>();

        if (email == null || email.isBlank()) {
            errors.add("Email is required");
            return new ValidationResult(false, errors);
        }

        String normalizedEmail = email.trim().toLowerCase();

        if (!EMAIL_PATTERN.matcher(normalizedEmail).matches()) {
            errors.add("Invalid email format");
            return new ValidationResult(false, errors);
        }

        String domain = extractDomain(normalizedEmail);
        if (blockedDomains.contains(domain)) {
            errors.add("Disposable email addresses are not allowed");
        }

        return new ValidationResult(errors.isEmpty(), errors);
    }

    public boolean isValid(String email) {
        return validate(email).isValid();
    }

    private String extractDomain(String email) {
        int atIndex = email.indexOf('@');
        return atIndex >= 0 ? email.substring(atIndex + 1) : "";
    }

    public record ValidationResult(boolean isValid, List<String> errors) {
        public String getFirstError() {
            return errors.isEmpty() ? null : errors.get(0);
        }
    }
}
