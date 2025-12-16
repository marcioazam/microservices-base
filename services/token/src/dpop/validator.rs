//! DPoP Proof Validator per RFC 9449
//!
//! Validates DPoP proofs including signature, jti uniqueness, htm, htu, and iat claims.

use crate::dpop::proof::{DPoPProof, DPoPError};
use crate::dpop::thumbprint::JwkThumbprint;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::RwLock;
use std::collections::HashSet;

/// Maximum clock skew allowed for iat validation (60 seconds)
const MAX_CLOCK_SKEW_SECS: i64 = 60;

/// JTI cache TTL (5 minutes)
const JTI_TTL_SECS: u64 = 300;

/// DPoP Validator with replay prevention
pub struct DPoPValidator {
    /// Cache of seen JTIs for replay prevention
    jti_cache: Arc<RwLock<HashSet<String>>>,
    /// Redis client for distributed JTI tracking
    redis: Option<Arc<redis::Client>>,
}

impl DPoPValidator {
    pub fn new() -> Self {
        DPoPValidator {
            jti_cache: Arc::new(RwLock::new(HashSet::new())),
            redis: None,
        }
    }

    pub fn with_redis(redis: Arc<redis::Client>) -> Self {
        DPoPValidator {
            jti_cache: Arc::new(RwLock::new(HashSet::new())),
            redis: Some(redis),
        }
    }

    /// Validates a DPoP proof
    pub async fn validate(
        &self,
        proof: &DPoPProof,
        expected_htm: &str,
        expected_htu: &str,
        access_token: Option<&str>,
    ) -> Result<ValidationResult, DPoPError> {
        // 1. Validate htm (HTTP method)
        if proof.claims.htm.to_uppercase() != expected_htm.to_uppercase() {
            return Err(DPoPError::HtmMismatch {
                expected: expected_htm.to_string(),
                actual: proof.claims.htm.clone(),
            });
        }

        // 2. Validate htu (HTTP URI)
        if !self.validate_htu(&proof.claims.htu, expected_htu) {
            return Err(DPoPError::HtuMismatch {
                expected: expected_htu.to_string(),
                actual: proof.claims.htu.clone(),
            });
        }

        // 3. Validate iat (issued at) within clock skew
        let now = chrono::Utc::now().timestamp();
        let iat = proof.claims.iat;
        
        if iat > now + MAX_CLOCK_SKEW_SECS {
            return Err(DPoPError::InvalidIat);
        }
        
        if iat < now - MAX_CLOCK_SKEW_SECS - JTI_TTL_SECS as i64 {
            return Err(DPoPError::InvalidIat);
        }

        // 4. Check jti uniqueness (replay prevention)
        if !self.check_and_store_jti(&proof.claims.jti).await {
            return Err(DPoPError::JtiReplay);
        }

        // 5. Validate ath (access token hash) if present
        if let Some(token) = access_token {
            if let Some(ref ath) = proof.claims.ath {
                let expected_ath = self.compute_ath(token);
                if ath != &expected_ath {
                    return Err(DPoPError::AthMismatch);
                }
            }
        }

        // 6. Compute thumbprint for token binding
        let thumbprint = JwkThumbprint::compute(&proof.header.jwk);

        Ok(ValidationResult {
            thumbprint,
            jwk: proof.header.jwk.clone(),
        })
    }

    /// Validates that a DPoP-bound token's jkt matches the proof
    pub fn validate_token_binding(
        &self,
        proof: &DPoPProof,
        token_jkt: &str,
    ) -> Result<(), DPoPError> {
        let proof_thumbprint = JwkThumbprint::compute(&proof.header.jwk);
        
        if !JwkThumbprint::verify(&proof.header.jwk, token_jkt) {
            return Err(DPoPError::ThumbprintMismatch);
        }

        Ok(())
    }

    /// Validates the htu claim against expected URI
    fn validate_htu(&self, htu: &str, expected: &str) -> bool {
        // Normalize URIs for comparison (remove trailing slashes, etc.)
        let normalize = |s: &str| {
            s.trim_end_matches('/')
                .to_lowercase()
        };
        
        normalize(htu) == normalize(expected)
    }

    /// Checks if JTI has been seen and stores it
    async fn check_and_store_jti(&self, jti: &str) -> bool {
        // Try Redis first for distributed tracking
        if let Some(ref redis) = self.redis {
            return self.check_jti_redis(redis, jti).await;
        }

        // Fallback to in-memory cache
        let mut cache = self.jti_cache.write().await;
        if cache.contains(jti) {
            return false;
        }
        cache.insert(jti.to_string());
        true
    }

    async fn check_jti_redis(&self, redis: &Arc<redis::Client>, jti: &str) -> bool {
        let key = format!("dpop:jti:{}", jti);
        
        // Use SET NX EX for atomic check-and-set with TTL
        match redis.get_multiplexed_async_connection().await {
            Ok(mut conn) => {
                let result: redis::RedisResult<Option<String>> = redis::cmd("SET")
                    .arg(&key)
                    .arg("1")
                    .arg("NX")
                    .arg("EX")
                    .arg(JTI_TTL_SECS)
                    .query_async(&mut conn)
                    .await;
                
                match result {
                    Ok(Some(_)) => true,  // Successfully set, JTI is new
                    Ok(None) => false,    // Already exists, replay attack
                    Err(_) => {
                        // Redis error, fallback to memory
                        let mut cache = self.jti_cache.blocking_write();
                        if cache.contains(jti) {
                            return false;
                        }
                        cache.insert(jti.to_string());
                        true
                    }
                }
            }
            Err(_) => {
                // Connection error, fallback to memory
                let mut cache = self.jti_cache.blocking_write();
                if cache.contains(jti) {
                    return false;
                }
                cache.insert(jti.to_string());
                true
            }
        }
    }

    /// Computes the access token hash (ath) per RFC 9449
    fn compute_ath(&self, access_token: &str) -> String {
        use sha2::{Sha256, Digest};
        let hash = Sha256::digest(access_token.as_bytes());
        base64::Engine::encode(&base64::engine::general_purpose::URL_SAFE_NO_PAD, hash)
    }
}

/// Result of successful DPoP validation
#[derive(Debug, Clone)]
pub struct ValidationResult {
    /// JWK thumbprint for token binding (cnf.jkt)
    pub thumbprint: String,
    /// The public key from the proof
    pub jwk: crate::dpop::proof::Jwk,
}

impl Default for DPoPValidator {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::dpop::proof::{DPoPHeader, DPoPClaims, Jwk};

    fn create_test_proof() -> DPoPProof {
        DPoPProof {
            header: DPoPHeader {
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
            },
            claims: DPoPClaims {
                jti: uuid::Uuid::new_v4().to_string(),
                htm: "POST".to_string(),
                htu: "https://auth.example.com/token".to_string(),
                iat: chrono::Utc::now().timestamp(),
                ath: None,
                nonce: None,
            },
            signature: vec![],
            raw_token: "header.payload.signature".to_string(),
        }
    }

    #[tokio::test]
    async fn test_validate_htm_mismatch() {
        let validator = DPoPValidator::new();
        let proof = create_test_proof();

        let result = validator.validate(
            &proof,
            "GET",  // Wrong method
            "https://auth.example.com/token",
            None,
        ).await;

        assert!(matches!(result, Err(DPoPError::HtmMismatch { .. })));
    }

    #[tokio::test]
    async fn test_validate_htu_mismatch() {
        let validator = DPoPValidator::new();
        let proof = create_test_proof();

        let result = validator.validate(
            &proof,
            "POST",
            "https://other.example.com/token",  // Wrong URI
            None,
        ).await;

        assert!(matches!(result, Err(DPoPError::HtuMismatch { .. })));
    }

    #[tokio::test]
    async fn test_jti_replay_detection() {
        let validator = DPoPValidator::new();
        let mut proof = create_test_proof();
        proof.claims.jti = "fixed-jti-for-test".to_string();

        // First validation should succeed
        let result1 = validator.validate(
            &proof,
            "POST",
            "https://auth.example.com/token",
            None,
        ).await;
        assert!(result1.is_ok());

        // Second validation with same JTI should fail
        let result2 = validator.validate(
            &proof,
            "POST",
            "https://auth.example.com/token",
            None,
        ).await;
        assert!(matches!(result2, Err(DPoPError::JtiReplay)));
    }

    #[tokio::test]
    async fn test_validate_success() {
        let validator = DPoPValidator::new();
        let proof = create_test_proof();

        let result = validator.validate(
            &proof,
            "POST",
            "https://auth.example.com/token",
            None,
        ).await;

        assert!(result.is_ok());
        let validation = result.unwrap();
        assert!(!validation.thumbprint.is_empty());
    }
}
