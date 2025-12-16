//! Type-State JWT Token with compile-time validation guarantees
//!
//! This module implements the type-state pattern for JWT validation,
//! ensuring that claims can only be accessed on fully validated tokens.

use std::marker::PhantomData;
use std::sync::Arc;

use jsonwebtoken::{decode, decode_header, Algorithm, DecodingKey, Header, Validation};
use serde::{Deserialize, Serialize};

use crate::error::AuthEdgeError;
use crate::jwt::claims::Claims;
use crate::jwt::jwk_cache::JwkCache;

// ============================================================================
// Sealed Trait Pattern for Token States
// ============================================================================

mod private {
    /// Sealed trait to prevent external implementations
    pub trait Sealed {}
}

/// Marker trait for token validation states
pub trait TokenState: private::Sealed {
    /// Human-readable state name for debugging
    fn state_name() -> &'static str;
}

/// Unvalidated token - just parsed, not verified
pub struct Unvalidated;
impl private::Sealed for Unvalidated {}
impl TokenState for Unvalidated {
    fn state_name() -> &'static str {
        "Unvalidated"
    }
}

/// Signature validated - cryptographic verification passed
pub struct SignatureValidated;
impl private::Sealed for SignatureValidated {}
impl TokenState for SignatureValidated {
    fn state_name() -> &'static str {
        "SignatureValidated"
    }
}

/// Fully validated - signature + claims verified
pub struct Validated;
impl private::Sealed for Validated {}
impl TokenState for Validated {
    fn state_name() -> &'static str {
        "Validated"
    }
}


// ============================================================================
// Type-State Token Wrapper
// ============================================================================

/// Type-state token wrapper that enforces validation at compile time
#[derive(Debug)]
pub struct Token<State: TokenState> {
    /// Raw JWT string
    raw: String,
    /// Parsed header (available in all states)
    header: Header,
    /// Decoded claims (only populated after validation)
    claims: Option<Claims>,
    /// Key ID used for validation
    kid: Option<String>,
    /// Phantom marker for state
    _state: PhantomData<State>,
}

impl Token<Unvalidated> {
    /// Parse a raw JWT string into an unvalidated token
    /// 
    /// This performs zero-copy header parsing where possible.
    pub fn parse(raw: &str) -> Result<Self, AuthEdgeError> {
        let header = decode_header(raw).map_err(|e| AuthEdgeError::TokenMalformed {
            reason: format!("Invalid header: {}", e),
        })?;

        let kid = header.kid.clone();

        Ok(Token {
            raw: raw.to_string(),
            header,
            claims: None,
            kid,
            _state: PhantomData,
        })
    }

    /// Get the key ID from the token header
    pub fn kid(&self) -> Option<&str> {
        self.kid.as_deref()
    }

    /// Get the algorithm from the token header
    pub fn algorithm(&self) -> Algorithm {
        self.header.alg
    }

    /// Validate the token signature using the JWK cache
    pub async fn validate_signature(
        self,
        cache: &JwkCache,
    ) -> Result<Token<SignatureValidated>, AuthEdgeError> {
        let kid = self.kid.as_ref().ok_or_else(|| AuthEdgeError::TokenMalformed {
            reason: "Missing kid in header".to_string(),
        })?;

        let decoding_key = cache.get_key(kid).await?;

        // Set up validation (signature only, no claims validation yet)
        let mut validation = Validation::new(self.header.alg);
        validation.validate_exp = false;
        validation.validate_nbf = false;
        validation.validate_aud = false;
        validation.required_spec_claims.clear();

        let token_data = decode::<Claims>(&self.raw, &decoding_key, &validation)
            .map_err(|e| {
                if e.to_string().contains("InvalidSignature") {
                    AuthEdgeError::TokenInvalid
                } else {
                    AuthEdgeError::TokenMalformed {
                        reason: format!("Signature validation failed: {}", e),
                    }
                }
            })?;

        Ok(Token {
            raw: self.raw,
            header: self.header,
            claims: Some(token_data.claims),
            kid: self.kid,
            _state: PhantomData,
        })
    }

    /// Validate signature with a specific decoding key (for testing)
    pub fn validate_signature_with_key(
        self,
        key: &DecodingKey,
    ) -> Result<Token<SignatureValidated>, AuthEdgeError> {
        let mut validation = Validation::new(self.header.alg);
        validation.validate_exp = false;
        validation.validate_nbf = false;
        validation.validate_aud = false;
        validation.required_spec_claims.clear();

        let token_data = decode::<Claims>(&self.raw, key, &validation)
            .map_err(|e| {
                if e.to_string().contains("InvalidSignature") {
                    AuthEdgeError::TokenInvalid
                } else {
                    AuthEdgeError::TokenMalformed {
                        reason: format!("Signature validation failed: {}", e),
                    }
                }
            })?;

        Ok(Token {
            raw: self.raw,
            header: self.header,
            claims: Some(token_data.claims),
            kid: self.kid,
            _state: PhantomData,
        })
    }
}


impl Token<SignatureValidated> {
    /// Validate claims and transition to fully validated state
    pub fn validate_claims(
        self,
        required_claims: &[&str],
    ) -> Result<Token<Validated>, AuthEdgeError> {
        let claims = self.claims.as_ref().ok_or_else(|| AuthEdgeError::TokenMalformed {
            reason: "Claims not available".to_string(),
        })?;

        // Validate expiration
        if claims.is_expired() {
            return Err(AuthEdgeError::TokenExpired {
                expired_at: chrono::DateTime::from_timestamp(claims.exp, 0)
                    .unwrap_or_else(chrono::Utc::now),
            });
        }

        // Validate not-before if present
        if let Some(nbf) = claims.nbf {
            let now = chrono::Utc::now().timestamp();
            if nbf > now {
                return Err(AuthEdgeError::TokenNotYetValid {
                    valid_from: chrono::DateTime::from_timestamp(nbf, 0)
                        .unwrap_or_else(chrono::Utc::now),
                });
            }
        }

        // Validate required claims
        let missing: Vec<String> = required_claims
            .iter()
            .filter(|claim| !self.has_claim(claims, claim))
            .map(|s| s.to_string())
            .collect();

        if !missing.is_empty() {
            return Err(AuthEdgeError::ClaimsInvalid { claims: missing });
        }

        Ok(Token {
            raw: self.raw,
            header: self.header,
            claims: self.claims,
            kid: self.kid,
            _state: PhantomData,
        })
    }

    fn has_claim(&self, claims: &Claims, claim_name: &str) -> bool {
        match claim_name {
            "iss" => !claims.iss.is_empty(),
            "sub" => !claims.sub.is_empty(),
            "aud" => !claims.aud.is_empty(),
            "exp" => true,
            "iat" => true,
            "jti" => !claims.jti.is_empty(),
            "session_id" => claims.session_id.is_some(),
            "scopes" => claims.scopes.is_some(),
            _ => claims.custom.contains_key(claim_name),
        }
    }

    /// Get read-only access to claims (signature validated but not fully validated)
    pub fn peek_claims(&self) -> Option<&Claims> {
        self.claims.as_ref()
    }
}

impl Token<Validated> {
    /// Access claims - only available on fully validated tokens
    pub fn claims(&self) -> &Claims {
        self.claims.as_ref().expect("Validated token must have claims")
    }

    /// Get the subject claim
    pub fn subject(&self) -> &str {
        &self.claims().sub
    }

    /// Get the issuer claim
    pub fn issuer(&self) -> &str {
        &self.claims().iss
    }

    /// Get the audience claim
    pub fn audience(&self) -> &[String] {
        &self.claims().aud
    }

    /// Get the expiration timestamp
    pub fn expires_at(&self) -> i64 {
        self.claims().exp
    }

    /// Get the issued-at timestamp
    pub fn issued_at(&self) -> i64 {
        self.claims().iat
    }

    /// Get the JWT ID
    pub fn jti(&self) -> &str {
        &self.claims().jti
    }

    /// Get the session ID if present
    pub fn session_id(&self) -> Option<&str> {
        self.claims().session_id.as_deref()
    }

    /// Check if the token has a specific scope
    pub fn has_scope(&self, scope: &str) -> bool {
        self.claims().has_scope(scope)
    }

    /// Get the raw token string
    pub fn raw(&self) -> &str {
        &self.raw
    }

    /// Get the token header
    pub fn header(&self) -> &Header {
        &self.header
    }
}

// Common methods for all states
impl<S: TokenState> Token<S> {
    /// Get the current state name
    pub fn state_name(&self) -> &'static str {
        S::state_name()
    }
}
