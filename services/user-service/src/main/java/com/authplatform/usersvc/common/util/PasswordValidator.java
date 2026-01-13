package com.authplatform.usersvc.common.util;

import org.springframework.stereotype.Component;
import java.util.ArrayList;
import java.util.List;
import java.util.regex.Pattern;

@Component
public class PasswordValidator {

    private static final int MIN_LENGTH = 8;
    private static final int MAX_LENGTH = 128;
    private static final Pattern UPPERCASE_PATTERN = Pattern.compile("[A-Z]");
    private static final Pattern LOWERCASE_PATTERN = Pattern.compile("[a-z]");
    private static final Pattern DIGIT_PATTERN = Pattern.compile("[0-9]");

    public ValidationResult validate(String password) {
        List<String> errors = new ArrayList<>();

        if (password == null || password.isEmpty()) {
            errors.add("Password is required");
            return new ValidationResult(false, errors);
        }

        if (password.length() < MIN_LENGTH) {
            errors.add("Password must be at least " + MIN_LENGTH + " characters");
        }

        if (password.length() > MAX_LENGTH) {
            errors.add("Password must be at most " + MAX_LENGTH + " characters");
        }

        if (!UPPERCASE_PATTERN.matcher(password).find()) {
            errors.add("Password must contain at least one uppercase letter");
        }

        if (!LOWERCASE_PATTERN.matcher(password).find()) {
            errors.add("Password must contain at least one lowercase letter");
        }

        if (!DIGIT_PATTERN.matcher(password).find()) {
            errors.add("Password must contain at least one digit");
        }

        return new ValidationResult(errors.isEmpty(), errors);
    }

    public boolean isValid(String password) {
        return validate(password).isValid();
    }

    public record ValidationResult(boolean isValid, List<String> errors) {
        public String getFirstError() {
            return errors.isEmpty() ? null : errors.get(0);
        }
    }
}
