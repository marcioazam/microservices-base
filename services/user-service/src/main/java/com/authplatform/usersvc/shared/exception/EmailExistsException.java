package com.authplatform.usersvc.shared.exception;

public final class EmailExistsException extends UserServiceException {
    
    public EmailExistsException() {
        super("Email already exists");
    }

    @Override
    public String getErrorCode() {
        return "EMAIL_EXISTS";
    }

    @Override
    public int getHttpStatus() {
        return 409;
    }
}
