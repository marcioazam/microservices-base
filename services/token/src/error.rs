use thiserror::Error;

#[derive(Error, Debug)]
pub enum TokenError {
    #[error("Token refresh invalid: {0}")]
    RefreshInvalid(String),

    #[error("Token refresh expired")]
    RefreshExpired,

    #[error("Token refresh reused - replay attack detected")]
    RefreshReused,

    #[error("Token family revoked")]
    FamilyRevoked,

    #[error("KMS signing error: {0}")]
    KmsError(String),

    #[error("JWT encoding error: {0}")]
    JwtEncodingError(String),

    #[error("JWT decoding error: {0}")]
    JwtDecodingError(String),

    #[error("Redis error: {0}")]
    RedisError(String),

    #[error("Configuration error: {0}")]
    ConfigError(String),

    #[error("Internal error: {0}")]
    Internal(String),
}

impl From<redis::RedisError> for TokenError {
    fn from(err: redis::RedisError) -> Self {
        TokenError::RedisError(err.to_string())
    }
}

impl From<jsonwebtoken::errors::Error> for TokenError {
    fn from(err: jsonwebtoken::errors::Error) -> Self {
        TokenError::JwtEncodingError(err.to_string())
    }
}

// Error codes for gRPC responses
pub const TOKEN_REFRESH_INVALID: &str = "TOKEN_REFRESH_INVALID";
pub const TOKEN_REFRESH_EXPIRED: &str = "TOKEN_REFRESH_EXPIRED";
pub const TOKEN_REFRESH_REUSED: &str = "TOKEN_REFRESH_REUSED";
pub const TOKEN_FAMILY_REVOKED: &str = "TOKEN_FAMILY_REVOKED";
pub const TOKEN_KMS_ERROR: &str = "TOKEN_KMS_ERROR";
