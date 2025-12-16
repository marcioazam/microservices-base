//! DPoP Proof structure per RFC 9449
//! 
//! Implements Demonstrating Proof of Possession for OAuth 2.0

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// DPoP Proof JWT Header
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DPoPHeader {
    /// Must be "dpop+jwt"
    pub typ: String,
    /// Algorithm (ES256 or RS256)
    pub alg: String,
    /// JSON Web Key (public key)
    pub jwk: Jwk,
}

/// JSON Web Key for DPoP
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct Jwk {
    pub kty: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub crv: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub x: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub y: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub n: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub e: Option<String>,
}

/// DPoP Proof JWT Claims
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DPoPClaims {
    /// Unique identifier for the proof
    pub jti: String,
    /// HTTP method (GET, POST, etc.)
    pub htm: String,
    /// HTTP URI (target endpoint)
    pub htu: String,
    /// Issued at timestamp
    pub iat: i64,
    /// Access token hash (for resource requests)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ath: Option<String>,
    /// Server-provided nonce
    #[serde(skip_serializing_if = "Option::is_none")]
    pub nonce: Option<String>,
}

/// Complete DPoP Proof
#[derive(Debug, Clone)]
pub struct DPoPProof {
    pub header: DPoPHeader,
    pub claims: DPoPClaims,
    pub signature: Vec<u8>,
    pub raw_token: String,
}

impl DPoPProof {
    /// Parses a DPoP proof from a JWT string
    pub fn parse(token: &str) -> Result<Self, DPoPError> {
        let parts: Vec<&str> = token.split('.').collect();
        if parts.len() != 3 {
            return Err(DPoPError::MalformedToken("Invalid JWT format".to_string()));
        }

        // Decode header
        let header_bytes = base64::Engine::decode(
            &base64::engine::general_purpose::URL_SAFE_NO_PAD,
            parts[0],
        ).map_err(|e| DPoPError::MalformedToken(e.to_string()))?;
        
        let header: DPoPHeader = serde_json::from_slice(&header_bytes)
            .map_err(|e| DPoPError::MalformedToken(e.to_string()))?;

        // Validate header
        if header.typ != "dpop+jwt" {
            return Err(DPoPError::InvalidType(header.typ.clone()));
        }

        if header.alg != "ES256" && header.alg != "RS256" {
            return Err(DPoPError::UnsupportedAlgorithm(header.alg.clone()));
        }

        // Decode claims
        let claims_bytes = base64::Engine::decode(
            &base64::engine::general_purpose::URL_SAFE_NO_PAD,
            parts[1],
        ).map_err(|e| DPoPError::MalformedToken(e.to_string()))?;
        
        let claims: DPoPClaims = serde_json::from_slice(&claims_bytes)
            .map_err(|e| DPoPError::MalformedToken(e.to_string()))?;

        // Decode signature
        let signature = base64::Engine::decode(
            &base64::engine::general_purpose::URL_SAFE_NO_PAD,
            parts[2],
        ).map_err(|e| DPoPError::MalformedToken(e.to_string()))?;

        Ok(DPoPProof {
            header,
            claims,
            signature,
            raw_token: token.to_string(),
        })
    }

    /// Gets the signing input (header.payload) for signature verification
    pub fn signing_input(&self) -> String {
        let parts: Vec<&str> = self.raw_token.split('.').collect();
        format!("{}.{}", parts[0], parts[1])
    }
}

#[derive(Debug, thiserror::Error)]
pub enum DPoPError {
    #[error("Malformed DPoP token: {0}")]
    MalformedToken(String),
    
    #[error("Invalid typ header: expected 'dpop+jwt', got '{0}'")]
    InvalidType(String),
    
    #[error("Unsupported algorithm: {0}")]
    UnsupportedAlgorithm(String),
    
    #[error("Invalid signature")]
    InvalidSignature,
    
    #[error("Token expired or iat invalid")]
    InvalidIat,
    
    #[error("HTTP method mismatch: expected {expected}, got {actual}")]
    HtmMismatch { expected: String, actual: String },
    
    #[error("HTTP URI mismatch: expected {expected}, got {actual}")]
    HtuMismatch { expected: String, actual: String },
    
    #[error("JTI already used (replay attack)")]
    JtiReplay,
    
    #[error("Access token hash mismatch")]
    AthMismatch,
    
    #[error("Thumbprint mismatch")]
    ThumbprintMismatch,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_dpop_header_validation() {
        let header = DPoPHeader {
            typ: "dpop+jwt".to_string(),
            alg: "ES256".to_string(),
            jwk: Jwk {
                kty: "EC".to_string(),
                crv: Some("P-256".to_string()),
                x: Some("test-x".to_string()),
                y: Some("test-y".to_string()),
                n: None,
                e: None,
            },
        };

        assert_eq!(header.typ, "dpop+jwt");
        assert_eq!(header.alg, "ES256");
    }

    #[test]
    fn test_dpop_claims() {
        let claims = DPoPClaims {
            jti: "unique-id".to_string(),
            htm: "POST".to_string(),
            htu: "https://auth.example.com/token".to_string(),
            iat: chrono::Utc::now().timestamp(),
            ath: None,
            nonce: None,
        };

        assert_eq!(claims.htm, "POST");
        assert!(claims.ath.is_none());
    }
}
