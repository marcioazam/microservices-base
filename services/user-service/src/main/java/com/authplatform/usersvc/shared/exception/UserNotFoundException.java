package com.authplatform.usersvc.shared.exception;

public final class UserNotFoundException extends UserServiceException {
    
    public UserNotFoundException() {
        super("User not found");
    }

    @Override
    public String getErrorCode() {
        return "USER_NOT_FOUND";
    }

    @Override
    public int getHttpStatus() {
        return 404;
    }
}
