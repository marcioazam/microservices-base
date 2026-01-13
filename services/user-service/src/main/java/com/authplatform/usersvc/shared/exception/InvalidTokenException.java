package com.authplatform.usersvc.shared.exception;

public final class InvalidTokenException extends UserServiceException {
    
    public InvalidTokenException() {
        super("Invalid verification token");
    }

    @Override
    public String getErrorCode() {
        return "INVALID_TOKEN";
    }

    @Override
    public int getHttpStatus() {
        return 400;
    }
}
