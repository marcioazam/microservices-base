package com.authplatform.usersvc.common.util;

import org.springframework.stereotype.Component;
import java.util.ArrayList;
import java.util.List;
import java.util.regex.Pattern;

@Component
public class DisplayNameValidator {

    private static final int MIN_LENGTH = 1;
    private static final int MAX_LENGTH = 100;
    private static final Pattern ALLOWED_CHARS = Pattern.compile("^[\\p{L}\\p{N}\\s._-]+$");

    public ValidationResult validate(String displayName) {
        List<String> errors = new ArrayList<>();

        if (displayName == null || displayName.isBlank()) {
            errors.add("Display name is required");
            return new ValidationResult(false, errors);
        }

        String trimmed = displayName.trim();

        if (trimmed.length() < MIN_LENGTH) {
            errors.add("Display name must be at least " + MIN_LENGTH + " character");
        }

        if (trimmed.length() > MAX_LENGTH) {
            errors.add("Display name must be at most " + MAX_LENGTH + " characters");
        }

        if (!ALLOWED_CHARS.matcher(trimmed).matches()) {
            errors.add("Display name contains invalid characters");
        }

        return new ValidationResult(errors.isEmpty(), errors);
    }

    public boolean isValid(String displayName) {
        return validate(displayName).isValid();
    }

    public record ValidationResult(boolean isValid, List<String> errors) {
        public String getFirstError() {
            return errors.isEmpty() ? null : errors.get(0);
        }
    }
}
