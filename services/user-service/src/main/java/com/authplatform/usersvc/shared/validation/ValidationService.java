package com.authplatform.usersvc.shared.validation;

import org.springframework.stereotype.Service;
import org.springframework.web.util.HtmlUtils;

import java.util.ArrayList;
import java.util.List;
import java.util.Set;
import java.util.regex.Pattern;

/**
 * Centralized validation service for all input validation.
 * Single source of truth for email, password, and display name validation.
 */
@Service
public class ValidationService {

    private static final Pattern EMAIL_PATTERN = Pattern.compile(
            "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    );
    
    private static final Set<String> DISPOSABLE_DOMAINS = Set.of(
            "tempmail.com", "throwaway.com", "mailinator.com", "guerrillamail.com",
            "10minutemail.com", "temp-mail.org", "fakeinbox.com", "trashmail.com"
    );
    
    private static final Pattern SCRIPT_PATTERN = Pattern.compile(
            "<script[^>]*>.*?</script>|javascript:|on\\w+\\s*=",
            Pattern.CASE_INSENSITIVE | Pattern.DOTALL
    );

    private static final int PASSWORD_MIN_LENGTH = 8;
    private static final int PASSWORD_MAX_LENGTH = 128;
    private static final int DISPLAY_NAME_MIN_LENGTH = 2;
    private static final int DISPLAY_NAME_MAX_LENGTH = 50;

    /**
     * Validates email format, disposable domains, and normalizes.
     */
    public ValidationResult validateEmail(String email) {
        if (email == null || email.isBlank()) {
            return ValidationResult.failure(FieldError.of("email", "REQUIRED", "Email is required"));
        }
        
        String normalized = email.trim().toLowerCase();
        
        if (!EMAIL_PATTERN.matcher(normalized).matches()) {
            return ValidationResult.failure(FieldError.of("email", "INVALID_FORMAT", "Invalid email format"));
        }
        
        String domain = normalized.substring(normalized.indexOf('@') + 1);
        if (DISPOSABLE_DOMAINS.contains(domain)) {
            return ValidationResult.failure(FieldError.of("email", "DISPOSABLE_EMAIL", "Disposable emails not allowed"));
        }
        
        return ValidationResult.success();
    }

    /**
     * Validates password: min 8, max 128, complexity (upper, lower, digit, special).
     */
    public ValidationResult validatePassword(String password) {
        if (password == null || password.isEmpty()) {
            return ValidationResult.failure(FieldError.of("password", "REQUIRED", "Password is required"));
        }
        
        List<FieldError> errors = new ArrayList<>();
        
        if (password.length() < PASSWORD_MIN_LENGTH) {
            errors.add(FieldError.of("password", "TOO_SHORT", 
                    "Password must be at least " + PASSWORD_MIN_LENGTH + " characters"));
        }
        
        if (password.length() > PASSWORD_MAX_LENGTH) {
            errors.add(FieldError.of("password", "TOO_LONG", 
                    "Password must not exceed " + PASSWORD_MAX_LENGTH + " characters"));
        }
        
        if (!password.matches(".*[A-Z].*")) {
            errors.add(FieldError.of("password", "MISSING_UPPERCASE", "Password must contain uppercase letter"));
        }
        
        if (!password.matches(".*[a-z].*")) {
            errors.add(FieldError.of("password", "MISSING_LOWERCASE", "Password must contain lowercase letter"));
        }
        
        if (!password.matches(".*\\d.*")) {
            errors.add(FieldError.of("password", "MISSING_DIGIT", "Password must contain a digit"));
        }
        
        if (!password.matches(".*[!@#$%^&*(),.?\":{}|<>].*")) {
            errors.add(FieldError.of("password", "MISSING_SPECIAL", "Password must contain special character"));
        }
        
        return errors.isEmpty() ? ValidationResult.success() : ValidationResult.failure(errors);
    }

    /**
     * Validates display name: length limits and sanitizes HTML/script content.
     */
    public ValidationResult validateDisplayName(String displayName) {
        if (displayName == null || displayName.isBlank()) {
            return ValidationResult.failure(FieldError.of("displayName", "REQUIRED", "Display name is required"));
        }
        
        String trimmed = displayName.trim();
        List<FieldError> errors = new ArrayList<>();
        
        if (trimmed.length() < DISPLAY_NAME_MIN_LENGTH) {
            errors.add(FieldError.of("displayName", "TOO_SHORT", 
                    "Display name must be at least " + DISPLAY_NAME_MIN_LENGTH + " characters"));
        }
        
        if (trimmed.length() > DISPLAY_NAME_MAX_LENGTH) {
            errors.add(FieldError.of("displayName", "TOO_LONG", 
                    "Display name must not exceed " + DISPLAY_NAME_MAX_LENGTH + " characters"));
        }
        
        if (SCRIPT_PATTERN.matcher(trimmed).find()) {
            errors.add(FieldError.of("displayName", "INVALID_CONTENT", "Display name contains invalid content"));
        }
        
        return errors.isEmpty() ? ValidationResult.success() : ValidationResult.failure(errors);
    }

    /**
     * Sanitizes display name by escaping HTML.
     */
    public String sanitizeDisplayName(String displayName) {
        if (displayName == null) return null;
        return HtmlUtils.htmlEscape(displayName.trim());
    }

    /**
     * Normalizes email to lowercase.
     */
    public String normalizeEmail(String email) {
        if (email == null) return null;
        return email.trim().toLowerCase();
    }

    /**
     * Validates complete registration request.
     */
    public ValidationResult validateRegistration(String email, String password, String displayName) {
        List<FieldError> allErrors = new ArrayList<>();
        
        var emailResult = validateEmail(email);
        if (!emailResult.valid()) {
            allErrors.addAll(emailResult.errors());
        }
        
        var passwordResult = validatePassword(password);
        if (!passwordResult.valid()) {
            allErrors.addAll(passwordResult.errors());
        }
        
        var displayNameResult = validateDisplayName(displayName);
        if (!displayNameResult.valid()) {
            allErrors.addAll(displayNameResult.errors());
        }
        
        return allErrors.isEmpty() ? ValidationResult.success() : ValidationResult.failure(allErrors);
    }
}
