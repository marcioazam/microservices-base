package com.authplatform.usersvc.shared.exception;

import com.authplatform.usersvc.shared.validation.FieldError;

import java.util.List;

public final class ValidationException extends UserServiceException {
    
    private final List<FieldError> errors;

    public ValidationException(List<FieldError> errors) {
        super("Validation failed");
        this.errors = errors;
    }

    public ValidationException(FieldError error) {
        this(List.of(error));
    }

    public List<FieldError> getErrors() {
        return errors;
    }

    @Override
    public String getErrorCode() {
        return "VALIDATION_ERROR";
    }

    @Override
    public int getHttpStatus() {
        return 400;
    }
}
