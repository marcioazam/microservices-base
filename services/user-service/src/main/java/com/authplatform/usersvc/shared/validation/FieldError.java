package com.authplatform.usersvc.shared.validation;

/**
 * Represents a field-level validation error.
 */
public record FieldError(String field, String code, String message) {
    
    public static FieldError of(String field, String code, String message) {
        return new FieldError(field, code, message);
    }
}
