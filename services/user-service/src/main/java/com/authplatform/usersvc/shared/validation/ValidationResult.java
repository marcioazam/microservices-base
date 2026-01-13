package com.authplatform.usersvc.shared.validation;

import java.util.List;

/**
 * Result of validation operation.
 */
public record ValidationResult(boolean valid, List<FieldError> errors) {
    
    public static ValidationResult success() {
        return new ValidationResult(true, List.of());
    }
    
    public static ValidationResult failure(List<FieldError> errors) {
        return new ValidationResult(false, errors);
    }
    
    public static ValidationResult failure(FieldError error) {
        return new ValidationResult(false, List.of(error));
    }
}
