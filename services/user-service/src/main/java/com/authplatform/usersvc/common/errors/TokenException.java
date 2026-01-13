package com.authplatform.usersvc.common.errors;

public class TokenException extends RuntimeException {
    private final TokenErrorType type;

    public TokenException(TokenErrorType type, String message) {
        super(message);
        this.type = type;
    }

    public TokenErrorType getType() {
        return type;
    }

    public enum TokenErrorType {
        INVALID,
        EXPIRED,
        ALREADY_USED
    }
}
