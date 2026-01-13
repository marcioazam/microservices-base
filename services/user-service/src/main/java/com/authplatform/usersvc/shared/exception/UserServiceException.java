package com.authplatform.usersvc.shared.exception;

/**
 * Base sealed exception for all User Service exceptions.
 */
public sealed abstract class UserServiceException extends RuntimeException
        permits EmailExistsException, InvalidTokenException, ExpiredTokenException,
                AlreadyUsedException, UserNotFoundException, RateLimitedException, 
                ValidationException {

    protected UserServiceException(String message) {
        super(message);
    }

    protected UserServiceException(String message, Throwable cause) {
        super(message, cause);
    }

    public abstract String getErrorCode();
    public abstract int getHttpStatus();
}
