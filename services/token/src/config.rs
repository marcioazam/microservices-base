use serde::Deserialize;
use std::env;

#[derive(Debug, Clone, Deserialize)]
pub struct Config {
    pub host: String,
    pub port: u16,
    pub redis_url: String,
    pub kms_provider: String,
    pub kms_key_id: String,
    pub access_token_ttl_seconds: i64,
    pub refresh_token_ttl_seconds: i64,
    pub jwt_issuer: String,
    pub jwt_algorithm: String,
}

impl Config {
    pub fn from_env() -> Result<Self, Box<dyn std::error::Error>> {
        dotenvy::dotenv().ok();

        Ok(Config {
            host: env::var("HOST").unwrap_or_else(|_| "0.0.0.0".to_string()),
            port: env::var("PORT")
                .unwrap_or_else(|_| "50051".to_string())
                .parse()?,
            redis_url: env::var("REDIS_URL")
                .unwrap_or_else(|_| "redis://localhost:6379".to_string()),
            kms_provider: env::var("KMS_PROVIDER").unwrap_or_else(|_| "mock".to_string()),
            kms_key_id: env::var("KMS_KEY_ID").unwrap_or_else(|_| "default-key".to_string()),
            access_token_ttl_seconds: env::var("ACCESS_TOKEN_TTL")
                .unwrap_or_else(|_| "900".to_string())
                .parse()?,
            refresh_token_ttl_seconds: env::var("REFRESH_TOKEN_TTL")
                .unwrap_or_else(|_| "604800".to_string())
                .parse()?,
            jwt_issuer: env::var("JWT_ISSUER")
                .unwrap_or_else(|_| "auth-platform".to_string()),
            jwt_algorithm: env::var("JWT_ALGORITHM").unwrap_or_else(|_| "RS256".to_string()),
        })
    }
}
