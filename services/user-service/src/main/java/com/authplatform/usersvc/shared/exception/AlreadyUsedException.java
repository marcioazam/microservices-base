package com.authplatform.usersvc.shared.exception;

public final class AlreadyUsedException extends UserServiceException {
    
    public AlreadyUsedException() {
        super("Verification token has already been used");
    }

    @Override
    public String getErrorCode() {
        return "ALREADY_USED";
    }

    @Override
    public int getHttpStatus() {
        return 400;
    }
}
