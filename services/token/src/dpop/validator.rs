//! DPoP Proof Validator per RFC 9449.
//!
//! Validates DPoP proofs including signature, jti uniqueness, htm, htu, and iat claims.
//! Uses CacheStorage for distributed JTI tracking.

use crate::dpop::proof::{DPoPClaims, DPoPError, DPoPHeader, DPoPProof, Jwk};
use crate::dpop::thumbprint::JwkThumbprint;
use crate::storage::CacheStorage;
use sha2::{Digest, Sha256};
use std::sync::Arc;
use std::time::Duration;

/// DPoP Validator with replay prevention via CacheStorage.
pub struct DPoPValidator {
    storage: Arc<CacheStorage>,
    clock_skew: Duration,
    jti_ttl: Duration,
}

impl DPoPValidator {
    /// Create a new validator with cache storage.
    pub fn new(storage: Arc<CacheStorage>, clock_skew: Duration, jti_ttl: Duration) -> Self {
        Self {
            storage,
            clock_skew,
            jti_ttl,
        }
    }

    /// Validates a DPoP proof per RFC 9449.
    pub async fn validate(
        &self,
        proof: &DPoPProof,
        expected_htm: &str,
        expected_htu: &str,
        access_token: Option<&str>,
    ) -> Result<ValidationResult, DPoPError> {
        // 1. Validate htm (HTTP method)
        if !self.validate_htm(&proof.claims.htm, expected_htm) {
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
        self.validate_iat(proof.claims.iat)?;

        // 4. Check jti uniqueness (replay prevention)
        if !self.check_and_store_jti(&proof.claims.jti).await? {
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

    /// Validates that a DPoP-bound token's jkt matches the proof.
    pub fn validate_token_binding(
        &self,
        proof: &DPoPProof,
        token_jkt: &str,
    ) -> Result<(), DPoPError> {
        if !JwkThumbprint::verify(&proof.header.jwk, token_jkt) {
            return Err(DPoPError::ThumbprintMismatch);
        }
        Ok(())
    }

    /// Validates the htm claim (case-insensitive).
    fn validate_htm(&self, htm: &str, expected: &str) -> bool {
        htm.eq_ignore_ascii_case(expected)
    }

    /// Validates the htu claim against expected URI.
    fn validate_htu(&self, htu: &str, expected: &str) -> bool {
        let normalize = |s: &str| s.trim_end_matches('/').to_lowercase();
        normalize(htu) == normalize(expected)
    }

    /// Validates iat is within acceptable clock skew.
    fn validate_iat(&self, iat: i64) -> Result<(), DPoPError> {
        let now = chrono::Utc::now().timestamp();
        let skew = self.clock_skew.as_secs() as i64;
        let ttl = self.jti_ttl.as_secs() as i64;

        if iat > now + skew {
            return Err(DPoPError::InvalidIat);
        }

        if iat < now - skew - ttl {
            return Err(DPoPError::InvalidIat);
        }

        Ok(())
    }

    /// Checks if JTI has been seen and stores it for replay prevention.
    async fn check_and_store_jti(&self, jti: &str) -> Result<bool, DPoPError> {
        self.storage
            .check_and_store_dpop_jti(jti, self.jti_ttl)
            .await
            .map_err(|e| DPoPError::Internal(e.to_string()))
    }

    /// Computes the access token hash (ath) per RFC 9449.
    fn compute_ath(&self, access_token: &str) -> String {
        let hash = Sha256::digest(access_token.as_bytes());
        base64::Engine::encode(&base64::engine::general_purpose::URL_SAFE_NO_PAD, hash)
    }
}

/// Result of successful DPoP validation.
#[derive(Debug, Clone)]
pub struct ValidationResult {
    /// JWK thumbprint for token binding (cnf.jkt).
    pub thumbprint: String,
    /// The public key from the proof.
    pub jwk: Jwk,
}

#[cfg(test)]
mod tests {
    use super::*;
    use rust_common::CacheClientConfig;

    async fn create_test_validator() -> DPoPValidator {
        let config = CacheClientConfig::default().with_namespace("dpop-test");
        let storage = Arc::new(CacheStorage::new(config).await.unwrap());
        DPoPValidator::new(storage, Duration::from_secs(60), Duration::from_secs(300))
    }

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
        let validator = create_test_validator().await;
        let proof = create_test_proof();

        let result = validator
            .validate(&proof, "GET", "https://auth.example.com/token", None)
            .await;

        assert!(matches!(result, Err(DPoPError::HtmMismatch { .. })));
    }

    #[tokio::test]
    async fn test_validate_htu_mismatch() {
        let validator = create_test_validator().await;
        let proof = create_test_proof();

        let result = validator
            .validate(&proof, "POST", "https://other.example.com/token", None)
            .await;

        assert!(matches!(result, Err(DPoPError::HtuMismatch { .. })));
    }

    #[tokio::test]
    async fn test_jti_replay_detection() {
        let validator = create_test_validator().await;
        let mut proof = create_test_proof();
        proof.claims.jti = "fixed-jti-for-test".to_string();

        let result1 = validator
            .validate(&proof, "POST", "https://auth.example.com/token", None)
            .await;
        assert!(result1.is_ok());

        let result2 = validator
            .validate(&proof, "POST", "https://auth.example.com/token", None)
            .await;
        assert!(matches!(result2, Err(DPoPError::JtiReplay)));
    }

    #[tokio::test]
    async fn test_validate_success() {
        let validator = create_test_validator().await;
        let proof = create_test_proof();

        let result = validator
            .validate(&proof, "POST", "https://auth.example.com/token", None)
            .await;

        assert!(result.is_ok());
        let validation = result.unwrap();
        assert!(!validation.thumbprint.is_empty());
    }

    #[test]
    fn test_htm_case_insensitive() {
        let config = CacheClientConfig::default();
        let rt = tokio::runtime::Runtime::new().unwrap();
        let storage = rt.block_on(async {
            Arc::new(CacheStorage::new(config).await.unwrap())
        });
        let validator = DPoPValidator::new(
            storage,
            Duration::from_secs(60),
            Duration::from_secs(300),
        );

        assert!(validator.validate_htm("POST", "post"));
        assert!(validator.validate_htm("post", "POST"));
        assert!(validator.validate_htm("Get", "GET"));
    }
}
