package com.authplatform.usersvc.shared.exception;

public final class ExpiredTokenException extends UserServiceException {
    
    public ExpiredTokenException() {
        super("Verification token has expired");
    }

    @Override
    public String getErrorCode() {
        return "EXPIRED_TOKEN";
    }

    @Override
    public int getHttpStatus() {
        return 400;
    }
}
