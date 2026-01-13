//! JWT Validator with type-state pattern support
//!
//! Provides both legacy validation API and new type-state based validation.

use crate::error::AuthEdgeError;
use crate::jwt::claims::Claims;
use crate::jwt::jwk_cache::JwkCache;
use crate::jwt::token::{Token, Unvalidated, SignatureValidated, Validated};
use std::sync::Arc;

/// JWT Validator with JWK cache integration
pub struct JwtValidator {
    jwk_cache: Arc<JwkCache>,
}

impl JwtValidator {
    /// Creates a new JWT validator with the given JWK cache
    pub fn new(jwk_cache: Arc<JwkCache>) -> Self {
        JwtValidator { jwk_cache }
    }

    /// Validates a JWT token using the type-state pattern
    /// 
    /// Returns a fully validated Token<Validated> that guarantees
    /// claims can only be accessed after validation.
    pub async fn validate_token(
        &self,
        raw_token: &str,
        required_claims: &[&str],
    ) -> Result<Token<Validated>, AuthEdgeError> {
        // Parse token (Unvalidated state)
        let unvalidated = Token::<Unvalidated>::parse(raw_token)?;
        
        // Validate signature (SignatureValidated state)
        let signature_validated = unvalidated.validate_signature(&self.jwk_cache).await?;
        
        // Validate claims (Validated state)
        let validated = signature_validated.validate_claims(required_claims)?;
        
        Ok(validated)
    }

    /// Legacy validation method for backward compatibility
    pub async fn validate(&self, token: &str, required_claims: &[String]) -> Result<Claims, AuthEdgeError> {
        let required_refs: Vec<&str> = required_claims.iter().map(|s| s.as_str()).collect();
        let validated = self.validate_token(token, &required_refs).await?;
        Ok(validated.claims().clone())
    }

    /// Validates only the token expiration
    pub fn validate_expiration(&self, claims: &Claims) -> Result<(), AuthEdgeError> {
        if claims.is_expired() {
            return Err(AuthEdgeError::TokenExpired {
                expired_at: chrono::DateTime::from_timestamp(claims.exp, 0)
                    .unwrap_or_else(chrono::Utc::now),
            });
        }
        Ok(())
    }
}
